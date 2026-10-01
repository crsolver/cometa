package codegen

import (
	"fmt"
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
	"strconv"
	"strings"
)

// Preserve public names when possible, but allow the common Jugador/jugador pair.
func (g *generator) globalName(name string) string {
	candidate := exported(name)
	occupied := map[string]bool{}
	for n := range g.model.Types {
		occupied[exported(n)] = true
	}
	for n := range g.model.Enums {
		occupied[exported(n)] = true
	}
	for n := range g.model.Interfaces {
		occupied[exported(n)] = true
	}
	for n := range g.model.Functions {
		occupied[exported(n)] = true
	}
	for n := range g.model.Globals {
		if n != name {
			occupied[exported(n)] = true
		}
	}
	if !occupied[candidate] {
		return candidate
	}
	candidate = fmt.Sprintf("CometaGlobal_%x", []byte(name))
	for occupied[candidate] {
		candidate += "_"
	}
	return candidate
}

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
