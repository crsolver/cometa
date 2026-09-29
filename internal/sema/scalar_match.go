package sema

import "cometa/internal/ast"

// scalarLiteralKey identifies a literal arm value, or reports why the
// expression is not one. Only plain literals are allowed as patterns, so the
// generated Go can use them directly as switch cases.
func (c *checker) scalarLiteralKey(expr ast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		// A top-level constant whose value is itself a plain literal.
		global, ok := c.model.Globals[e.Name]
		if !ok || !global.Constant || global.Decl == nil || depth > 8 {
			return "", false
		}
		return c.scalarLiteralKey(global.Decl.Value, depth+1)
	case *ast.LiteralExpr:
		switch e.Kind {
		case "entero":
			return "#" + e.Value, true
		case "cadena":
			return "s" + e.Value, true
		case "bool":
			return e.Value, true // "verdadero" / "falso"
		}
	case *ast.UnaryExpr:
		if lit, ok := e.Value.(*ast.LiteralExpr); ok && e.Operator == "-" && lit.Kind == "entero" {
			return "#-" + lit.Value, true
		}
	}
	return "", false
}

// checkScalarArm validates one arm of a `casos` over entero, cadena or bool.
func (c *checker) checkScalarArm(arm *ast.MatchArm, t Type, seen map[string]bool, wildcard *bool) error {
	if t.Kind != Integer && t.Kind != String && t.Kind != Boolean {
		return c.fail(arm.Pos, "los valores literales en casos solo sirven para entero, cadena o bool, no %s", t.String())
	}
	if *wildcard {
		return c.fail(arm.Pos, "no se permiten ramas después de '_'")
	}
	if arm.Pattern == "_" && len(arm.Literals) == 0 {
		*wildcard = true
		return nil
	}
	if len(arm.Literals) == 0 {
		return c.fail(arm.Pos, "casos sobre %s usa valores literales (por ejemplo 1, \"a\" o verdadero) o '_'", t.String())
	}
	for _, literal := range arm.Literals {
		key, ok := c.scalarLiteralKey(literal, 0)
		if !ok {
			if id, isName := literal.(*ast.IdentExpr); isName {
				return c.fail(literal.Position(), "%q no es una constante con un valor literal (declárala con const y un número, texto o bool)", id.Name)
			}
			return c.fail(literal.Position(), "los valores de casos deben ser literales simples (número, texto sin ${}, verdadero o falso) o constantes con uno de esos valores")
		}
		literalType, err := c.checkExpr(literal)
		if err != nil {
			return err
		}
		if !literalType.Equal(t) {
			return c.fail(literal.Position(), "el valor debe ser %s, no %s", t.String(), literalType.String())
		}
		if seen[key] {
			return c.fail(literal.Position(), "el valor está duplicado en casos")
		}
		seen[key] = true
	}
	return nil
}

// constantLabel turns a lone identifier arm that names a constant into a
// literal arm; the parser cannot tell it apart from a type pattern.
func (c *checker) constantLabel(arm *ast.MatchArm) {
	t := arm.TypePattern
	if t == nil || t.Element != nil || t.Key != nil || t.Wrapper != "" || len(t.Args) > 0 {
		return
	}
	if global, ok := c.model.Globals[t.Name]; ok && global.Constant {
		arm.Literals = []ast.Expr{&ast.IdentExpr{Pos: t.Pos, Name: t.Name}}
		arm.TypePattern = nil
	}
}
