package codegen

import (
	"cometa/internal/ast"
	"cometa/internal/sema"
)

func (g *generator) numericCall(call *ast.CallExpr, to sema.Type, value string) string {
	from := g.model.ExprTypes[call.Args[0]]
	if from.Equal(to) {
		return value
	}
	if to.Kind == sema.Decimal {
		return "float64(" + value + ")"
	}
	return "_hentero(" + value + ")"
}

const numericRuntime = `
func _hentero(v float64) int64 {
 if v != v || v < -9223372036854775808.0 || v >= 9223372036854775808.0 {
  panic("conversión a entero inválida: valor no finito o fuera del rango de entero")
 }
 return int64(v)
}
`
