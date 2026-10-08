package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
)

// StdlibModel is what a program uses of the standard library.
type StdlibModel struct {
	Modules []string // the imported "std/..." paths
	Used    bool
	// Calls holds the function each library call resolved to; for an
	// overloaded one, the variant chosen for its arguments.
	Calls map[*ast.CallExpr]stdlib.Function
	// Constants maps `.Rojo`, `color.Color.Rojo` and `mate.pi` to their Go expression.
	Constants map[ast.Expr]string
	// Channels maps `color.r` reads to the Go field of the color.
	Channels map[*ast.MemberExpr]string
}

type Asset struct{ Path, Data string }

// installStdlib declares the library: every type of every module with its
// fields and methods, and the interfaces programs implement. Functions are
// resolved on demand, at each call.
func (c *checker) installStdlib() error {
	c.model.Stdlib = StdlibModel{Calls: map[*ast.CallExpr]stdlib.Function{}, Constants: map[ast.Expr]string{}, Channels: map[*ast.MemberExpr]string{}}
	for _, m := range stdlib.Modules {
		for i := range m.Types {
			if t := &m.Types[i]; !t.Interface {
				c.model.Types[t.Symbol()] = &TypeInfo{Decl: t.Declaration(), Fields: map[string]FieldInfo{}, Methods: map[string]FuncInfo{}}
			}
		}
	}
	// A field or a method may name a type of another module, so every type is
	// declared before any of them is resolved.
	for _, m := range stdlib.Modules {
		for i := range m.Types {
			t := &m.Types[i]
			methods := map[string]FuncInfo{}
			for _, method := range t.Methods {
				sig, err := c.signature(method.Declaration())
				if err != nil {
					return err
				}
				methods[method.Name] = sig
			}
			if t.Interface {
				c.model.Interfaces[t.Symbol()] = &InterfaceInfo{Decl: &ast.InterfaceDecl{Name: t.Symbol(), Public: true}, Methods: methods}
				continue
			}
			info := c.model.Types[t.Symbol()]
			info.Methods = methods
			for _, field := range info.Decl.Fields {
				resolved, err := c.resolveType(field.Type)
				if err != nil {
					return err
				}
				info.Fields[field.Name] = FieldInfo{Decl: field, Type: resolved}
			}
		}
	}
	// Defaults are checked last: they may be literals of other library types.
	for _, m := range stdlib.Modules {
		for i := range m.Types {
			if info := c.model.Types[m.Types[i].Symbol()]; info != nil {
				if err := c.checkFieldDefaults(info.Decl); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// NativeMethod reports whether d is a catalog method. Its Go method takes every
// parameter, so omitted arguments are filled with their defaults at each call.
func NativeMethod(d *ast.FuncDecl) bool {
	return d != nil && d.Receiver != "" && stdlib.IsType(d.Receiver)
}

func (c *checker) stdlibCall(call *ast.CallExpr) (Type, bool, error) {
	id, ok := call.Callee.(*ast.IdentExpr)
	if !ok {
		return Type{}, false, nil
	}
	f, ok := stdlib.LookupSymbol(id.Name)
	if !ok {
		return Type{}, false, nil
	}
	c.model.Stdlib.Used = true
	c.model.Stdlib.Calls[call] = f
	if f.Visual && !stdlib.UsesPincel(c.model.Stdlib.Modules) {
		return Type{}, true, c.fail(call.Pos, "%s.%s comprueba un juego de Pincel: el proyecto debe importar std/pincel o alguno de sus módulos", f.Namespace, f.Name)
	}
	sig, err := c.signature(f.Declaration())
	if err != nil {
		return Type{}, true, err
	}
	switch f.Overload {
	case stdlib.Numeric:
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
			c.model.Stdlib.Calls[call] = f.Integer()
		}
	case stdlib.Scalar:
		if len(call.Args) < 2 {
			return Type{}, true, c.fail(call.Pos, "%s.%s requiere %s y %s", f.Namespace, f.Name, sig.Decl.Params[0].Name, sig.Decl.Params[1].Name)
		}
		kinds := [2]Kind{}
		for i := 0; i < 2; i++ {
			t, err := c.checkExpr(call.Args[i])
			if err != nil {
				return Type{}, true, err
			}
			kinds[i] = t.Kind
		}
		kind := kinds[0]
		if (kinds[0] == Integer && kinds[1] == Decimal) || (kinds[0] == Decimal && kinds[1] == Integer) {
			kind = Decimal
		}
		suffix := map[Kind]string{Integer: "Entero", Decimal: "Decimal", String: "Cadena", Boolean: "Bool"}[kind]
		if suffix == "" || (kinds[0] != kinds[1] && kind != Decimal) {
			return Type{}, true, c.fail(call.Pos, "%s.%s compara dos valores del mismo tipo (entero, decimal, cadena o bool)", f.Namespace, f.Name)
		}
		sig.Params[0], sig.Params[1] = Type{Kind: kind}, Type{Kind: kind}
		c.model.Stdlib.Calls[call] = f.Variant(suffix)
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

func (c *checker) checkStdlib(program *ast.Program) error {
	if err := c.checkPhases(program); err != nil {
		return err
	}
	ast.Inspect(program, func(n any) bool {
		if ref, ok := n.(*ast.TypeRef); ok && stdlib.IsType(ref.Name) {
			c.model.Stdlib.Used = true
		}
		return true
	})
	for _, t := range c.model.ExprTypes {
		if stdlib.IsType(t.Name) {
			c.model.Stdlib.Used = true
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
	for call, f := range c.model.Stdlib.Calls {
		if f.Resource {
			if !allowed[call] || len(call.Args) != 1 {
				return c.fail(call.Pos, "%s.%s requiere un inicializador global directo con ruta literal", f.Namespace, f.Name)
			}
			if lit, ok := call.Args[0].(*ast.LiteralExpr); !ok || lit.Kind != "cadena" {
				return c.fail(call.Pos, "la ruta del recurso debe ser una cadena literal")
			}
		}
	}
	return nil
}
