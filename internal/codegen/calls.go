package codegen

import (
	"hacha/internal/ast"
	"hacha/internal/sema"
	"strings"
)

func (g *generator) call(call *ast.CallExpr, callee string) string {
	info := g.model.Calls[call]
	callee = g.promotedCallee(callee, info.Signature, call.Callee)
	if info.Signature.Decl != nil && info.Signature.Decl.Receiver != "" && len(g.defaultFlags[info.Signature.Decl]) > 0 {
		callee = strings.TrimSuffix(callee, exported(info.Signature.Decl.Name)) + defaultMethod(info.Signature.Decl)
	}
	callee += typeArguments(info.Signature.TypeArgs)
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
