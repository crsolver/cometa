package codegen

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
	"strconv"
	"strings"
)

func (g *generator) gameCall(call *ast.CallExpr, f stdlib.Function, indent int) string {
	if f.Resource {
		asset := g.model.Assets[call]
		return "_hg" + f.GoName + "(" + strconv.Quote(asset.Data) + "," + strconv.Quote(asset.Path) + ")"
	}
	info := g.model.Calls[call]
	args := make([]string, len(info.Signature.Params))
	for i, arg := range call.Args {
		args[info.Parameters[i]] = g.flowExpr(arg, indent)
	}
	for i, p := range info.Signature.Decl.Params {
		if args[i] == "" {
			args[i] = g.flowExpr(p.Default, indent)
		}
	}
	return "_hg" + f.GoName + "(" + strings.Join(args, ",") + ")"
}
