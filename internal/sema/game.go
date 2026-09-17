package sema

import (
	"hacha/internal/ast"
	"hacha/internal/gameapi"
	"reflect"
)

type GameModel struct {
	Enabled   bool
	Used      bool
	Calls     map[*ast.CallExpr]gameapi.Function
	Constants map[ast.Expr]string
}

type Asset struct{ Path, Data string }

func (c *checker) installGameAPI() error {
	c.model.Game = GameModel{Calls: map[*ast.CallExpr]gameapi.Function{}, Constants: map[ast.Expr]string{}}
	for name := range gameapi.Fields {
		c.model.Types[name] = &TypeInfo{Decl: gameapi.TypeDeclaration(name), Fields: map[string]FieldInfo{}, Methods: map[string]FuncInfo{}}
	}
	for _, info := range c.model.Types {
		for _, field := range info.Decl.Fields {
			t, err := c.resolveType(field.Type)
			if err != nil {
				return err
			}
			info.Fields[field.Name] = FieldInfo{Decl: field, Type: t}
		}
	}
	for _, f := range gameapi.Methods {
		d := f.Declaration()
		d.Receiver = f.Namespace
		sig, err := c.signature(d)
		if err != nil {
			return err
		}
		c.model.Types[f.Namespace].Methods[f.Name] = sig
	}
	return nil
}

func (c *checker) gameCall(call *ast.CallExpr) (Type, bool, error) {
	m, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		return Type{}, false, nil
	}
	id, ok := m.Object.(*ast.IdentExpr)
	if !ok || !gameapi.IsNamespace(id.Name) {
		return Type{}, false, nil
	}
	f, ok := gameapi.Lookup(id.Name, m.Name)
	if !ok {
		return Type{}, true, c.fail(m.NamePos, "la función %s.%s no existe", id.Name, m.Name)
	}
	c.model.Game.Used = true
	c.model.Game.Calls[call] = f
	sig, err := c.signature(f.Declaration())
	if err != nil {
		return Type{}, true, err
	}
	// Check defaults against the same types used for explicit arguments.
	for i, p := range sig.Decl.Params {
		if p.Default != nil {
			if _, err := c.checkExprExpected(p.Default, &sig.Params[i]); err != nil {
				return Type{}, true, err
			}
		}
	}
	t, err := c.bindArguments(call, sig)
	return t, true, err
}

// WalkSyntax visits syntax only, never semantic links. Used for effect closure.
func WalkSyntax(node any, visit func(any)) {
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return
			}
			if v.CanInterface() {
				visit(v.Interface())
			}
			walk(v.Elem())
			return
		}
		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		}
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(node))
}

func (c *checker) checkGame(program *ast.Program) error {
	WalkSyntax(program, func(n any) {
		if ref, ok := n.(*ast.TypeRef); ok && gameapi.IsType(ref.Name) {
			c.model.Game.Used = true
		}
	})
	for _, t := range c.model.ExprTypes {
		if gameapi.IsType(t.Name) {
			c.model.Game.Used = true
		}
	}
	get := func(name string) *ast.FuncDecl { return c.model.Functions[name].Decl }
	c.model.Game.Enabled = get("actualizar") != nil || get("pintar") != nil
	if c.model.Game.Enabled {
		if get("inicio") != nil {
			return c.fail(get("inicio").Pos, "un juego no puede declarar inicio; use iniciar")
		}
		for _, name := range []string{"actualizar", "pintar", "iniciar"} {
			f := get(name)
			if f == nil {
				if name == "actualizar" || name == "pintar" {
					return c.fail(ast.Pos{Line: 1, Column: 1}, "un juego requiere fn %s", name)
				}
				continue
			}
			sig := c.model.Functions[name]
			n := 0
			if name == "actualizar" {
				n = 1
			}
			valid := len(f.TypeParams) == 0 && len(f.Params) == n
			valid = valid && sig.Return.Kind == Void
			if n == 1 && len(f.Params) == 1 {
				valid = valid && sig.Params[0].Kind == Number && !f.Params[0].Variadic && f.Params[0].Default == nil
			}
			if !valid {
				return c.fail(f.Pos, "firma inválida del callback %s", name)
			}
		}
	}
	// Only direct global resource constructors are declarations. Helpers and
	// nested resource calls cannot depend on runtime paths or execute repeatedly.
	allowed := map[*ast.CallExpr]bool{}
	for _, d := range program.Decls {
		if g, ok := d.(*ast.GlobalDecl); ok {
			if call, ok := g.Value.(*ast.CallExpr); ok {
				allowed[call] = !g.Constant
			}
		}
	}
	for call, f := range c.model.Game.Calls {
		if f.Resource {
			if !allowed[call] || len(call.Args) != 1 {
				return c.fail(call.Pos, "recursos.%s requiere un inicializador global directo con ruta literal", f.Name)
			}
			if lit, ok := call.Args[0].(*ast.LiteralExpr); !ok || lit.Kind != "cadena" {
				return c.fail(call.Pos, "la ruta del recurso debe ser una cadena literal")
			}
		}
	}
	// Resolve effects transitively, including defaults, methods and conservative
	// structural-interface dispatch. Recursion is bounded by declaration identity.
	var effect func(any, map[*ast.FuncDecl]bool) *ast.CallExpr
	effect = func(node any, seen map[*ast.FuncDecl]bool) *ast.CallExpr {
		var found *ast.CallExpr
		WalkSyntax(node, func(n any) {
			if found != nil {
				return
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return
			}
			if c.model.Game.Calls[call].Draw {
				found = call
				return
			}
			sig := c.model.Calls[call].Signature
			var candidates []*ast.FuncDecl
			if sig.Decl != nil {
				candidates = append(candidates, sig.Decl)
			}
			if m, ok := call.Callee.(*ast.MemberExpr); ok {
				t := c.model.ExprTypes[m.Object]
				if t.Kind == Interface || t.Kind == TypeParameter {
					for _, typ := range c.model.Types {
						if fn, ok := typ.Methods[m.Name]; ok {
							candidates = append(candidates, fn.Decl)
						}
					}
				}
			}
			for _, fn := range candidates {
				if fn != nil && !seen[fn] {
					seen[fn] = true
					if v := effect(fn, seen); v != nil {
						found = v
						return
					}
				}
			}
		})
		return found
	}
	for _, d := range program.Decls {
		var root any
		switch d := d.(type) {
		case *ast.GlobalDecl:
			root = d.Value
		case *ast.FuncDecl:
			if d.Name == "actualizar" || d.Name == "iniciar" || d.Name == "inicio" {
				root = d
			}
		}
		if root != nil {
			if call := effect(root, map[*ast.FuncDecl]bool{}); call != nil {
				return c.fail(call.Pos, "dibujar solo se permite desde pintar y sus helpers")
			}
		}
	}
	return c.checkConfigEffects(program)
}
