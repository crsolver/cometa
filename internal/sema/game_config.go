package sema

import "hacha/internal/ast"

// Include defaults and conservative interface dispatch in the startup call graph.
func (c *checker) gameCallees(call *ast.CallExpr) []*ast.FuncDecl {
	var result []*ast.FuncDecl
	if d := c.model.Calls[call].Signature.Decl; d != nil {
		result = append(result, d)
	}
	if m, ok := call.Callee.(*ast.MemberExpr); ok {
		t := c.model.ExprTypes[m.Object]
		if t.Kind == Interface || t.Kind == TypeParameter {
			for _, typ := range c.model.Types {
				if f, ok := typ.Methods[m.Name]; ok {
					result = append(result, f.Decl)
				}
			}
		}
	}
	return result
}

func (c *checker) checkConfigEffects(program *ast.Program) error {
	reachable := map[*ast.FuncDecl]bool{}
	var visit func(any, map[*ast.FuncDecl]bool) *ast.CallExpr
	visit = func(node any, seen map[*ast.FuncDecl]bool) *ast.CallExpr {
		var found *ast.CallExpr
		WalkSyntax(node, func(n any) {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return
			}
			f := c.model.Game.Calls[call]
			if f.Namespace == "juego" && f.Name == "configuracion" {
				found = call
			}
			for _, d := range c.gameCallees(call) {
				if d != nil && !seen[d] {
					seen[d] = true
					if nested := visit(d, seen); nested != nil {
						found = nested
					}
				}
			}
		})
		return found
	}
	start := c.model.Functions["iniciar"].Decl
	if start != nil && c.model.Game.Enabled {
		reachable[start] = true
		visit(start, reachable)
	}
	var roots []any
	for _, d := range program.Decls {
		switch d := d.(type) {
		case *ast.GlobalDecl:
			roots = append(roots, d.Value)
		case *ast.FuncDecl:
			if !reachable[d] || d.Name == "actualizar" || d.Name == "pintar" || d.Name == "inicio" {
				roots = append(roots, d)
			}
		case *ast.TypeDecl:
			for _, m := range d.Methods {
				if !reachable[m] {
					roots = append(roots, m)
				}
			}
		}
	}
	for _, root := range roots {
		if call := visit(root, map[*ast.FuncDecl]bool{}); call != nil {
			return c.fail(call.Pos, "configuración solo se permite desde iniciar y sus helpers")
		}
	}
	return nil
}
