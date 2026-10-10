package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
)

// EnumCompare records `valor == Enum.Variante`: the generated code compares
// the value's variant tag with Tag. Payload is set only for an optional or a
// result compared with `.Alguno(x)`, `.Ok(x)` or `.Error(x)`: the payload of
// the value must then equal that expression too.
type EnumCompare struct {
	Value   ast.Expr
	Tag     int
	Payload ast.Expr
}

// contextualVariant reports whether e is `.Variante` or `.Variante(x)`.
func contextualVariant(e ast.Expr) bool {
	if call, ok := e.(*ast.CallExpr); ok {
		e = call.Callee
	}
	_, ok := e.(*ast.ContextualVariantExpr)
	return ok
}

// checkContextualEnumEquality handles `estado == .Corriendo` (either order),
// `opcional == .Ninguno`, `opcional == .Alguno(3)`, `resultado == .Ok`,
// and `tecla == .Espacio` for library types with constants.
// The contextual side needs the other side's type, so it cannot go
// through the general path, which checks both operands without expectations.
// handled is false when the expression is not of that form or the other side
// is not an enum, an optional or a result, leaving the usual checks to report
// the problem.
func (c *checker) checkContextualEnumEquality(expr *ast.BinaryExpr) (Type, bool, error) {
	leftContextual := contextualVariant(expr.Left)
	rightContextual := contextualVariant(expr.Right)
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
	if stdlib.Constants[value.Name] != nil {
		// tecla == .Espacio: library constants are plain comparable values.
		if _, err := c.checkExprExpected(variantSide, &value); err != nil {
			return Type{}, true, err
		}
		return Type{Kind: Boolean}, true, nil
	}
	if value.Kind != Enum && !value.Wrapped() {
		return Type{}, false, nil
	}
	if _, err := c.checkExprExpected(variantSide, &value); err != nil {
		return Type{}, true, err
	}
	constructor, ok := c.model.Constructors[variantSide]
	compare := EnumCompare{Value: valueSide, Tag: constructor.Variant.Tag}
	if payload := constructor.Variant.Payload; ok && payload.Kind != Void && value.Wrapped() {
		// The payload of an optional or a result is compared when it is a
		// plain value; an enum declares its own variants and goes to casos.
		if payload.Kind != Integer && payload.Kind != Decimal && payload.Kind != String && payload.Kind != Boolean {
			return Type{}, true, c.fail(expr.Pos, "%s solo se compara con .%s(valor) cuando el valor es entero, decimal, cadena o bool, no %s; usa casos o si (...) |valor|", value.String(), constructor.Variant.Decl.Name, payload.String())
		}
		compare.Payload = variantSide.(*ast.CallExpr).Args[0]
	} else if !ok || payload.Kind != Void {
		return Type{}, true, c.fail(expr.Pos, "los enums solo se comparan con una variante sin payload (estado == .Corriendo); para lo demás usa casos")
	}
	c.model.EnumCompares[expr] = compare
	return Type{Kind: Boolean}, true, nil
}
