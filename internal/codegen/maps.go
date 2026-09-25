package codegen

import (
	"cometa/internal/ast"
	"cometa/internal/sema"
)

func (g *generator) mapLookup(object, key string, resultType sema.Type, indent int) string {
	value, exists, result := g.freshName(), g.freshName(), g.freshName()
	g.line(indent, "%s, %s := %s[%s]", value, exists, object, key)
	g.line(indent, "var %s %s", result, goType(resultType))
	g.line(indent, "if %s { %s = %s{tag: 1, payload1: %s} }", exists, result, goType(resultType), value)
	return result
}

func (g *generator) flowMapCall(call *ast.CallExpr, operation string, indent int) string {
	member := call.Callee.(*ast.MemberExpr)
	t := g.model.ExprTypes[member.Object]
	object := g.flowExpr(member.Object, indent)
	key := ""
	if len(call.Args) != 0 {
		key = g.flowExpr(call.Args[0], indent)
		if key == "" {
			return ""
		}
	}
	switch operation {
	case "longitud":
		return "int64(len(" + object + "))"
	case "esta_vacia":
		return "(len(" + object + ") == 0)"
	case "obtener":
		return g.mapLookup(object, key, sema.Type{Kind: sema.Optional, Elem: t.Elem}, indent)
	case "contiene", "eliminar":
		exists := g.freshName()
		g.line(indent, "_, %s := %s[%s]", exists, object, key)
		if operation == "eliminar" {
			g.line(indent, "delete(%s, %s)", object, key)
		}
		return exists
	case "vaciar":
		g.line(indent, "clear(%s)", object)
		return ""
	case "copiar":
		result, k, v := g.freshName(), g.freshName(), g.freshName()
		g.line(indent, "%s := make(%s, len(%s))", result, goType(t), object)
		g.line(indent, "for %s, %s := range %s { %s[%s] = %s }", k, v, object, result, k, v)
		return result
	case "claves", "valores":
		element := t.Key
		if operation == "valores" {
			element = t.Elem
		}
		result, item := g.freshName(), g.freshName()
		g.line(indent, "%s := make([]%s, 0, len(%s))", result, goType(*element), object)
		binding := item
		if operation == "valores" {
			binding = "_, " + item
		}
		g.line(indent, "for %s := range %s { %s = append(%s, %s) }", binding, object, result, result, item)
		return result
	}
	panic("unknown map method")
}
