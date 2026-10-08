package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
)

// checkPhases enforces the parts of the library that only work from one game
// callback (stdlib Function.Phase and Type.Phase): it follows inicio,
// actualizar, pintar and the global initializers through the helpers they
// call, their default values and the methods of their `con` blocks. Calls
// that cannot be followed are guarded by the runtime instead.
func (c *checker) checkPhases(program *ast.Program) error {
	used := false
	for _, f := range c.model.Stdlib.Calls {
		if f.Phased() {
			used = true
			break
		}
	}
	if !used {
		return nil
	}
	var inspect func(any, string, map[*ast.FuncDecl]bool) error
	inspect = func(node any, phase string, seen map[*ast.FuncDecl]bool) error {
		var failure error
		follow := func(f *ast.FuncDecl) {
			if f != nil && !seen[f] {
				seen[f] = true
				if err := inspect(f, phase, seen); err != nil {
					failure = err
				}
			}
		}
		// Syntax only: the calls a function makes are followed explicitly.
		ast.Inspect(node, func(node any) bool {
			if failure != nil {
				return false
			}
			switch n := node.(type) {
			case *ast.ScopeStmt:
				typ := c.model.ExprTypes[n.Value]
				if native, ok := stdlib.LookupType(typ.Name); ok && native.Phase != "" && native.Phase != phase {
					failure = c.fail(n.Pos, "los bloques con de %s solo se permiten desde %s", native.Module().Name, native.Phase)
					return false
				}
				for _, name := range []string{"entrar", "salir"} {
					follow(c.model.Methods(typ)[name].Decl)
				}
			case *ast.CallExpr:
				if f, ok := c.model.Stdlib.Calls[n]; ok && f.Phase != "" && f.Phase != phase {
					failure = c.fail(n.Pos, "%s.%s solo se permite desde %s", f.Namespace, f.Name, f.Phase)
				}
				info := c.model.Calls[n].Signature
				follow(info.Decl)
				if member, ok := n.Callee.(*ast.MemberExpr); ok && c.model.ExprTypes[member.Object].Kind == Interface {
					for _, t := range c.model.Types {
						follow(t.Methods[member.Name].Decl)
					}
				}
			}
			return failure == nil
		})
		return failure
	}
	for _, decl := range program.Decls {
		if global, ok := decl.(*ast.GlobalDecl); ok {
			if err := inspect(global.Value, "inicialización", map[*ast.FuncDecl]bool{}); err != nil {
				return err
			}
		}
		functions := []*ast.FuncDecl{}
		switch d := decl.(type) {
		case *ast.FuncDecl:
			functions = append(functions, d)
		case *ast.TypeDecl:
			functions = append(functions, d.Methods...)
		}
		for _, f := range functions {
			if f.Name == "inicio" || f.Name == "actualizar" || f.Name == "pintar" {
				if err := inspect(f, f.Name, map[*ast.FuncDecl]bool{f: true}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
