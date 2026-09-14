package codegen

import (
	"hacha/internal/ast"
	"hacha/internal/sema"
	"strings"
)

func (g *generator) call(call *ast.CallExpr, callee string) string {
	info := g.model.Calls[call]
	hasDefaults := len(g.defaultFlags[info.Signature.Decl]) > 0
	capture := info.Named || hasDefaults
	args := make([]string, len(call.Args))
	var setup strings.Builder
	if capture {
		// Capture the function/method (and its receiver) before evaluating arguments.
		name := g.freshName()
		setup.WriteString(name + " := " + callee + "; ")
		callee = name
	}
	for i, arg := range call.Args {
		value := g.expr(arg)
		if capture {
			name := g.freshName()
			setup.WriteString("var " + name + " " + goType(g.model.ExprTypes[arg]) + " = " + value + "; ")
			value = name
		}
		if i < len(info.Spread) && info.Spread[i] {
			value += "..."
		}
		args[i] = value
	}
	if !capture {
		return callee + "(" + strings.Join(args, ", ") + ")"
	}
	var ordered []string
	// Presence flags distinguish omitted arguments from explicit zero values.
	for parameter, param := range info.Signature.Decl.Params {
		if param.Default == nil {
			continue
		}
		missing := "true"
		for _, index := range info.Parameters {
			if index == parameter {
				missing = "false"
				break
			}
		}
		ordered = append(ordered, missing)
	}
	for parameter := range info.Signature.Params {
		found := false
		for i, index := range info.Parameters {
			if index == parameter {
				found = true
				ordered = append(ordered, args[i])
			}
		}
		if !found && info.Signature.Decl.Params[parameter].Default != nil {
			name := g.freshName()
			setup.WriteString("var " + name + " " + goType(info.Signature.Params[parameter]) + "; ")
			ordered = append(ordered, name)
		}
	}
	result := ""
	ret := ""
	if info.Signature.Return.Kind != sema.Void {
		result = " " + goType(info.Signature.Return)
		ret = "return "
	}
	return "func()" + result + " { " + setup.String() + ret + callee + "(" + strings.Join(ordered, ", ") + ") }()"
}
