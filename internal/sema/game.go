package sema

import (
	"cometa/internal/ast"
	"cometa/internal/stdlib"
	"reflect"
)

type GameModel struct {
	Modules   []string
	Used      bool
	Calls     map[*ast.CallExpr]stdlib.Function
	Constants map[ast.Expr]string
}

type Asset struct{ Path, Data string }

func (c *checker) installGameAPI() error {
	c.model.Game = GameModel{Calls: map[*ast.CallExpr]stdlib.Function{}, Constants: map[ast.Expr]string{}}
	for name := range stdlib.Fields {
		c.model.Types[name] = &TypeInfo{Decl: stdlib.TypeDeclaration(name), Fields: map[string]FieldInfo{}, Methods: map[string]FuncInfo{}}
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
	for _, f := range stdlib.Methods {
		d := f.Declaration()
		d.Name = f.Name
		d.Receiver = f.Namespace
		sig, err := c.signature(d)
		if err != nil {
			return err
		}
		c.model.Types[f.Namespace].Methods[f.Name] = sig
	}
	methods := map[string]FuncInfo{}
	for _, name := range []string{"actualizar", "pintar"} {
		signature := ")"
		if name == "actualizar" {
			signature = "dt decimal)"
		}
		d := (stdlib.Function{Namespace: "juego", Name: name, Signature: signature}).Declaration()
		d.Name = name
		info, err := c.signature(d)
		if err != nil {
			return err
		}
		methods[name] = info
	}
	c.model.Interfaces[stdlib.Symbol("Juego")] = &InterfaceInfo{Decl: &ast.InterfaceDecl{Name: stdlib.Symbol("Juego"), Public: true}, Methods: methods}
	return nil
}

func (c *checker) gameCall(call *ast.CallExpr) (Type, bool, error) {
	id, ok := call.Callee.(*ast.IdentExpr)
	if !ok {
		return Type{}, false, nil
	}
	f, ok := stdlib.LookupSymbol(id.Name)
	if !ok {
		return Type{}, false, nil
	}
	c.model.Game.Used = true
	c.model.Game.Calls[call] = f
	sig, err := c.signature(f.Declaration())
	if err != nil {
		return Type{}, true, err
	}
	if f.Namespace == "mate" && (f.Name == "absoluto" || f.Name == "minimo" || f.Name == "maximo" || f.Name == "limitar") {
		kind := Integer
		for _, arg := range call.Args {
			t, err := c.checkExpr(arg)
			if err != nil {
				return Type{}, true, err
			}
			if t.Kind == Decimal {
				kind = Decimal
			}
		}
		if kind == Integer {
			for i := range sig.Params {
				sig.Params[i] = Type{Kind: Integer}
			}
			sig.Return = Type{Kind: Integer}
			f.GoName += "Entero"
			c.model.Game.Calls[call] = f
		}
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
	if err := c.checkUIPhases(program); err != nil {
		return err
	}
	WalkSyntax(program, func(n any) {
		if ref, ok := n.(*ast.TypeRef); ok && stdlib.IsType(ref.Name) {
			c.model.Game.Used = true
		}
	})
	for _, t := range c.model.ExprTypes {
		if stdlib.IsType(t.Name) {
			c.model.Game.Used = true
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
	return nil
}
