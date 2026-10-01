package sema

import (
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
)

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
func (c *checker) checkScalarArm(arm *ast.MatchArm, t Type, seen map[string]bool, ranges *[][2]int64, wildcard *bool) error {
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
		if r, isRange := literal.(*ast.RangeLabel); isRange {
			if err := c.checkRangeLabel(r, t, seen, ranges); err != nil {
				return err
			}
			continue
		}
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
		if n, isInteger := integerKey(key); isInteger {
			for _, r := range *ranges {
				if n >= r[0] && n < r[1] {
					return c.fail(literal.Position(), "el valor ya está cubierto por un rango de casos")
				}
			}
		}
		seen[key] = true
	}
	return nil
}

// integerKey recovers the number behind an entero key of scalarLiteralKey.
func integerKey(key string) (int64, bool) {
	if !strings.HasPrefix(key, "#") {
		return 0, false
	}
	n, err := strconv.ParseInt(key[1:], 10, 64)
	return n, err == nil
}

// checkRangeLabel validates a `1..5` label: integer literal or constant
// bounds, a non-empty range and no overlap with earlier values or ranges.
func (c *checker) checkRangeLabel(r *ast.RangeLabel, t Type, seen map[string]bool, ranges *[][2]int64) error {
	if t.Kind != Integer {
		return c.fail(r.Pos, "los rangos en casos solo sirven para entero, no %s", t.String())
	}
	var bounds [2]int64
	for i, bound := range []ast.Expr{r.Start, r.End} {
		key, _ := c.scalarLiteralKey(bound, 0)
		n, ok := integerKey(key)
		if !ok {
			return c.fail(bound.Position(), "los extremos de un rango de casos deben ser números enteros o constantes con un valor entero")
		}
		if _, err := c.checkExpr(bound); err != nil {
			return err
		}
		bounds[i] = n
	}
	if bounds[0] >= bounds[1] {
		return c.fail(r.Pos, "el rango %d..%d está vacío: el inicio debe ser menor que el fin, que no se incluye", bounds[0], bounds[1])
	}
	for _, other := range *ranges {
		if bounds[0] < other[1] && other[0] < bounds[1] {
			return c.fail(r.Pos, "el rango %d..%d se solapa con el rango %d..%d", bounds[0], bounds[1], other[0], other[1])
		}
	}
	for key := range seen {
		if n, ok := integerKey(key); ok && n >= bounds[0] && n < bounds[1] {
			return c.fail(r.Pos, "el rango %d..%d incluye el valor %d, que ya aparece en casos", bounds[0], bounds[1], n)
		}
	}
	*ranges = append(*ranges, bounds)
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
