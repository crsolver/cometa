package codegen

import (
	"fmt"
	"hacha/internal/ast"
	"hacha/internal/sema"
	"strings"
)

// Flow lowering keeps every function exit in its original Go function. All
// operands are captured in source order; lazy operands live inside branches.
func (g *generator) flowBlock(body []ast.Stmt, indent int, target string) {
	for i, stmt := range body {
		dest := ""
		if i == len(body)-1 {
			dest = target
		}
		g.flowStmt(stmt, indent, dest)
		if g.model.Terminates(stmt) {
			break
		}
	}
}

func (g *generator) deliver(value string, indent int, target string) {
	if value == "" {
		return
	}
	if target == "return" {
		g.line(indent, "return %s", value)
	} else if target != "" {
		g.line(indent, "%s = %s", target, value)
	} else {
		g.line(indent, "_ = %s", value)
	}
}

func (g *generator) flowStmt(stmt ast.Stmt, indent int, target string) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		v := g.flowExpr(s.Expr, indent)
		g.deliver(v, indent, target)
	case *ast.VarDeclStmt:
		v := g.flowExpr(s.Value, indent)
		if v == "" {
			return
		}
		g.line(indent, "var %s %s = %s", localName(s.Name), goType(g.model.VarTypes[s]), v)
		g.line(indent, "_ = %s", localName(s.Name))
	case *ast.AssignStmt:
		// Capture an address before the RHS, preserving indexing and receiver order.
		address := g.freshName()
		g.line(indent, "%s := &(%s)", address, g.flowTarget(s.Target, indent))
		g.line(indent, "_ = %s", address)
		v := g.flowExpr(s.Value, indent)
		if v != "" {
			g.line(indent, "*%s = %s", address, v)
		}
	case *ast.IfStmt:
		g.flowIf(s.Branches, s.Else, indent, target)
	case *ast.MatchStmt:
		g.flowMatch(s.Match, indent, target)
	case *ast.RepeatStmt:
		g.flowLoop(s, indent)
	case *ast.ContinueStmt:
		g.line(indent, "continue")
	case *ast.BreakStmt:
		if g.loopLabel != "" {
			g.line(indent, "break %s", g.loopLabel)
		} else {
			g.line(indent, "break")
		}
	}
}

func (g *generator) flowTarget(e ast.Expr, indent int) string {
	switch v := e.(type) {
	case *ast.IdentExpr:
		return localName(v.Name)
	case *ast.ReceiverExpr:
		return "_self." + exported(v.Name)
	case *ast.MemberExpr:
		return g.flowExpr(v.Object, indent) + "." + exported(v.Name)
	case *ast.IndexExpr:
		object := g.flowExpr(v.Object, indent)
		index := g.flowExpr(v.Index, indent)
		return object + "[int(" + index + ")]"
	}
	panic("invalid assignment target")
}

func (g *generator) flowIf(branches []ast.IfBranch, otherwise []ast.Stmt, indent int, target string) {
	if len(branches) == 0 {
		g.flowBlock(otherwise, indent, target)
		return
	}
	b := branches[0]
	condition := g.flowExpr(b.Condition, indent)
	test := condition
	if b.Binding != "" {
		test += ".tag == 1"
	}
	g.line(indent, "if %s {", test)
	if b.Binding != "" {
		g.line(indent+1, "%s := %s.payload1", localName(b.Binding), condition)
		g.line(indent+1, "_ = %s", localName(b.Binding))
	}
	g.flowBlock(b.Body, indent+1, target)
	if len(branches) > 1 || len(otherwise) > 0 {
		g.line(indent, "} else {")
		g.flowIf(branches[1:], otherwise, indent+1, target)
	}
	g.line(indent, "}")
}

func (g *generator) flowMatch(m *ast.MatchExpr, indent int, target string) {
	value := g.flowExpr(m.Value, indent)
	info := g.model.EnumFor(g.model.ExprTypes[m.Value])
	g.line(indent, "switch %s.tag {", value)
	wildcard := false
	for _, arm := range m.Arms {
		if arm.Pattern == "_" {
			wildcard = true
			g.line(indent, "default:")
		} else {
			variant := info.Variants[arm.Pattern]
			g.line(indent, "case %d:", variant.Tag)
			if m.Binding != "" && variant.Payload.Kind != sema.Void {
				g.line(indent+1, "%s := %s.payload%d", localName(m.Binding), value, variant.Tag)
				g.line(indent+1, "_ = %s", localName(m.Binding))
			}
		}
		g.flowBlock(arm.Body, indent+1, target)
	}
	if !wildcard {
		g.line(indent, "default: panic(\"valor de variante inválido\")")
	}
	g.line(indent, "}")
}

func (g *generator) capture(t sema.Type, value string, indent int) string {
	if t.Kind == sema.Void {
		g.line(indent, "%s", value)
		return ""
	}
	name := g.freshName()
	g.line(indent, "var %s %s = %s", name, goType(t), value)
	g.line(indent, "_ = %s", name)
	return name
}

func (g *generator) flowExpr(e ast.Expr, indent int) string {
	t := g.model.ExprTypes[e]
	rawType := t
	wrap, wrapped := g.model.Wraps[e]
	if wrapped {
		rawType = *wrap.Elem
	}
	value := g.flowRaw(e, rawType, indent)
	if wrapped {
		value = goType(wrap) + "{tag: 1, payload1: " + value + "}"
	}
	if value == "" {
		return ""
	}
	return g.capture(t, value, indent)
}

func (g *generator) flowRaw(e ast.Expr, t sema.Type, indent int) string {
	if ctor, ok := g.model.Constructors[e]; ok {
		fields := fmt.Sprintf("tag: %d", ctor.Variant.Tag)
		if call, ok := e.(*ast.CallExpr); ok {
			payload := g.flowExpr(call.Args[0], indent)
			if payload == "" {
				return ""
			}
			fields += fmt.Sprintf(", payload%d: %s", ctor.Variant.Tag, payload)
		}
		if t.Wrapped() {
			return goType(t) + "{" + fields + "}"
		}
		return "&" + exported(ctor.Enum.Decl.Name) + "{" + fields + "}"
	}
	switch v := e.(type) {
	case *ast.ReturnExpr:
		if v.Value == nil {
			g.line(indent, "return")
		} else {
			value := g.flowExpr(v.Value, indent)
			if value != "" {
				g.line(indent, "return %s", value)
			}
		}
		return ""
	case *ast.TryExpr:
		value := g.flowExpr(v.Value, indent)
		g.line(indent, "if %s.tag != 1 {", value)
		failure := goType(g.returnType) + "{}"
		if g.returnType.Kind == sema.Result {
			failure = goType(g.returnType) + "{tag: 2, payload2: " + value + ".payload2}"
		}
		g.line(indent+1, "return %s", failure)
		g.line(indent, "}")
		if t.Kind == sema.Void {
			return ""
		}
		return value + ".payload1"
	case *ast.RecoverExpr:
		value := g.flowExpr(v.Value, indent)
		result := ""
		if t.Kind != sema.Void {
			result = g.freshName()
			g.line(indent, "var %s %s", result, goType(t))
		}
		g.line(indent, "if %s.tag == 1 {", value)
		if result != "" {
			g.line(indent+1, "%s = %s.payload1", result, value)
		}
		g.line(indent, "} else {")
		if v.Binding != "" {
			g.line(indent+1, "%s := %s.payload2", localName(v.Binding), value)
			g.line(indent+1, "_ = %s", localName(v.Binding))
		}
		g.flowBlock(v.Body, indent+1, result)
		g.line(indent, "}")
		return result
	case *ast.BlockExpr:
		return g.flowValueBlock(v.Body, t, indent)
	case *ast.MatchExpr:
		result := ""
		if t.Kind != sema.Never {
			result = g.freshName()
			g.line(indent, "var %s %s", result, goType(t))
		}
		g.flowMatch(v, indent, result)
		return result
	case *ast.IfExpr:
		branches := make([]ast.IfBranch, len(v.Branches))
		for i, branch := range v.Branches {
			branches[i] = ast.IfBranch{Condition: branch.Condition, Binding: branch.Binding, Body: []ast.Stmt{&ast.ExprStmt{Expr: branch.Value}}}
		}
		return g.flowValueBlock([]ast.Stmt{&ast.IfStmt{Branches: branches, Else: []ast.Stmt{&ast.ExprStmt{Expr: v.Else}}}}, t, indent)
	case *ast.IdentExpr:
		return localName(v.Name)
	case *ast.ReceiverExpr:
		return "_self." + exported(v.Name)
	case *ast.LiteralExpr:
		if v.Kind == "bool" {
			if v.Value == "verdadero" {
				return "true"
			}
			return "false"
		}
		return v.Value
	case *ast.MemberExpr:
		return g.flowExpr(v.Object, indent) + "." + exported(v.Name)
	case *ast.IndexExpr:
		object := g.flowExpr(v.Object, indent)
		index := g.flowExpr(v.Index, indent)
		return object + "[int(" + index + ")]"
	case *ast.UnaryExpr:
		return "(" + v.Operator + g.flowExpr(v.Value, indent) + ")"
	case *ast.BinaryExpr:
		left := g.flowExpr(v.Left, indent)
		if v.Operator == "&&" || v.Operator == "||" {
			result := g.freshName()
			g.line(indent, "%s := %s", result, left)
			condition := result
			if v.Operator == "||" {
				condition = "!" + result
			}
			g.line(indent, "if %s {", condition)
			right := g.flowExpr(v.Right, indent+1)
			if right != "" {
				g.line(indent+1, "%s = %s", result, right)
			}
			g.line(indent, "}")
			return result
		}
		right := g.flowExpr(v.Right, indent)
		return "(" + left + " " + v.Operator + " " + right + ")"
	case *ast.CallExpr:
		return g.flowCall(v, indent)
	case *ast.StructLiteralExpr:
		var fields []string
		for _, f := range v.Fields {
			value := g.flowExpr(f.Value, indent)
			if value == "" {
				return ""
			}
			fields = append(fields, exported(f.Name)+": "+value)
		}
		fields = append(fields, g.defaultFields(t, v.Fields)...)
		return "&" + exported(t.Name) + "{" + strings.Join(fields, ", ") + "}"
	case *ast.ListLiteralExpr:
		var values []string
		for _, item := range v.Elements {
			value := g.flowExpr(item, indent)
			if value == "" {
				return ""
			}
			values = append(values, value)
		}
		return goType(t) + "{" + strings.Join(values, ", ") + "}"
	}
	panic(fmt.Sprintf("unsupported flow expression %T", e))
}

func (g *generator) flowValueBlock(body []ast.Stmt, t sema.Type, indent int) string {
	result := ""
	if t.Kind != sema.Never {
		result = g.freshName()
		g.line(indent, "var %s %s", result, goType(t))
	}
	g.line(indent, "{")
	g.flowBlock(body, indent+1, result)
	g.line(indent, "}")
	return result
}

func (g *generator) flowCall(call *ast.CallExpr, indent int) string {
	callee := ""
	switch v := call.Callee.(type) {
	case *ast.IdentExpr:
		callee = exported(v.Name)
		if v.Name == "imprimir" {
			callee = "fmt.Println"
		}
		if v.Name == "inicio" {
			callee = "main"
		}
	case *ast.ReceiverExpr:
		callee = "_self." + exported(v.Name)
	case *ast.MemberExpr:
		callee = g.flowExpr(v.Object, indent) + "." + exported(v.Name)
	}
	fn := g.freshName()
	g.line(indent, "%s := %s", fn, callee)
	g.line(indent, "_ = %s", fn)
	info := g.model.Calls[call]
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = g.flowExpr(arg, indent)
		if args[i] == "" {
			return ""
		}
		if i < len(info.Spread) && info.Spread[i] {
			args[i] += "..."
		}
	}
	if info.Signature.Decl == nil {
		return fn + "(" + strings.Join(args, ", ") + ")"
	}
	var ordered []string
	for p, param := range info.Signature.Decl.Params {
		if param.Default == nil {
			continue
		}
		missing := true
		for _, index := range info.Parameters {
			if index == p {
				missing = false
			}
		}
		ordered = append(ordered, fmt.Sprint(missing))
	}
	for p, param := range info.Signature.Decl.Params {
		found := false
		for i, index := range info.Parameters {
			if index == p {
				ordered = append(ordered, args[i])
				found = true
			}
		}
		if !found && param.Default != nil {
			name := g.freshName()
			g.line(indent, "var %s %s", name, goType(info.Signature.Params[p]))
			ordered = append(ordered, name)
		}
	}
	return fn + "(" + strings.Join(ordered, ", ") + ")"
}

func (g *generator) defaultFields(t sema.Type, supplied []ast.FieldValue) []string {
	seen := map[string]bool{}
	for _, f := range supplied {
		seen[f.Name] = true
	}
	var fields []string
	info := g.model.Types[t.Name]
	for _, field := range info.Decl.Fields {
		ft := info.Fields[field.Name].Type
		if !seen[field.Name] && ft.Kind == sema.Named {
			fields = append(fields, exported(field.Name)+": &"+exported(ft.Name)+"{"+strings.Join(g.defaultFields(ft, nil), ", ")+"}")
		}
	}
	return fields
}

func (g *generator) flowLoop(s *ast.RepeatStmt, indent int) {
	outer := g.loopLabel
	g.loopLabel = ""
	defer func() { g.loopLabel = outer }()
	start, end := "", ""
	if s.Iterable != nil {
		start = g.flowExpr(s.Iterable, indent)
	}
	if s.RangeEnd != nil {
		end = g.flowExpr(s.RangeEnd, indent)
	}
	step := ""
	if end != "" {
		step = g.freshName()
		g.line(indent, "%s := float64(1)", step)
		g.line(indent, "if %s > %s { %s = -1 }", start, end, step)
	}
	if needsLoopLabel(s.Body, false) {
		g.loopLabel = g.freshName()
		g.line(indent, "%s:", g.loopLabel)
	}
	if start == "" {
		g.line(indent, "for {")
	} else if end != "" {
		current, index := g.freshName(), g.freshName()
		g.line(indent, "for %s, %s := %s, float64(0); (%s > 0 && %s < %s) || (%s < 0 && %s > %s); %s, %s = %s + %s, %s + 1 {", current, index, start, step, current, end, step, current, end, current, index, current, step, index)
		g.line(indent+1, "%s := %s", localName(s.Element), current)
		if s.Index != "" {
			g.line(indent+1, "%s := %s", localName(s.Index), index)
		}
	} else {
		index := "_"
		if s.Index != "" {
			index = g.freshName()
		}
		g.line(indent, "for %s, %s := range %s {", index, localName(s.Element), start)
		if s.Index != "" {
			g.line(indent+1, "%s := float64(%s)", localName(s.Index), index)
		}
	}
	if s.Element != "" {
		g.line(indent+1, "_ = %s", localName(s.Element))
	}
	if s.Index != "" {
		g.line(indent+1, "_ = %s", localName(s.Index))
	}
	g.flowBlock(s.Body, indent+1, "")
	g.line(indent, "}")
}
