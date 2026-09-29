package sema

import "cometa/internal/ast"

// EnumCompare records `valor == Enum.Variante`: the generated code compares
// the value's variant tag with Tag and never looks at payloads.
type EnumCompare struct {
	Value ast.Expr
	Tag   int
}

// checkContextualEnumEquality handles `estado == .Corriendo` (either order).
// The contextual side needs the other side's enum type, so it cannot go
// through the general path, which checks both operands without expectations.
// handled is false when the expression is not of that form or the other side
// is not an enum, leaving the usual checks to report the problem.
func (c *checker) checkContextualEnumEquality(expr *ast.BinaryExpr) (Type, bool, error) {
	_, leftContextual := expr.Left.(*ast.ContextualVariantExpr)
	_, rightContextual := expr.Right.(*ast.ContextualVariantExpr)
	var variantSide, valueSide ast.Expr
	switch {
	case rightContextual && !leftContextual:
		variantSide, valueSide = expr.Right, expr.Left
	case leftContextual && !rightContextual:
		variantSide, valueSide = expr.Left, expr.Right
	default:
		return Type{}, false, nil
	}
	value, err := c.checkExpr(valueSide)
	if err != nil {
		return Type{}, true, errInvalid
	}
	if value.Kind != Enum {
		return Type{}, false, nil
	}
	if _, err := c.checkExprExpected(variantSide, &value); err != nil {
		return Type{}, true, err
	}
	constructor, ok := c.model.Constructors[variantSide]
	if !ok || constructor.Variant.Payload.Kind != Void {
		return Type{}, true, c.fail(expr.Pos, "los enums solo se comparan con una variante sin payload (estado == .Corriendo); para lo demás usa casos")
	}
	c.model.EnumCompares[expr] = EnumCompare{Value: valueSide, Tag: constructor.Variant.Tag}
	return Type{Kind: Boolean}, true, nil
}
