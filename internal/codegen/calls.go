package codegen

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/sema"
	"strings"
)

func (g *generator) promotedCallee(callee string, signature sema.FuncInfo, expr ast.Expr) string {
	if len(signature.EmbeddedPath) == 0 {
		return callee
	}
	name := exported(signature.Decl.Name)
	prefix := strings.TrimSuffix(callee, name)
	t := sema.Type{Kind: sema.Named, Name: g.receiver}
	if member, ok := expr.(*ast.MemberExpr); ok {
		t = g.model.ExprTypes[member.Object]
	}
	for _, field := range signature.EmbeddedPath {
		prefix += g.fieldName(t, field) + "."
		if info := g.model.StructInfo(t); info != nil {
			t = info.Fields[field].Type
		}
	}
	return prefix + name
}

func (g *generator) fieldName(t sema.Type, name string) string {
	if t.Kind == sema.Named {
		if field := g.model.Members(t)[name].Field; field.Decl != nil && field.Decl.Embedded {
			return exported(field.Type.Name)
		}
	}
	return exported(name)
}
