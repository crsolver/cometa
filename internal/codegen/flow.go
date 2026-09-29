package codegen

import (
	"cometa/internal/ast"
	"cometa/internal/sema"
	"cometa/internal/stdlib"
	"fmt"
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
		if g.leavesBlock(stmt) {
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
	g.directive(stmt.Position())
	switch s := stmt.(type) {
	case *ast.ScopeStmt:
		value := g.flowExpr(s.Value, indent)
		name := g.freshName()
		g.line(indent, "{")
		g.line(indent+1, "%s := %s", name, value)
		g.line(indent+1, "%s.Entrar()", name)
		g.scopes = append(g.scopes, name)
		g.flowBlock(s.Body, indent+1, "")
		if !g.leavesBlock(s) {
			g.line(indent+1, "%s.Salir()", name)
		}
		g.scopes = g.scopes[:len(g.scopes)-1]
		g.line(indent, "}")
	case *ast.ExprStmt:
		v := g.flowExpr(s.Expr, indent)
		g.deliver(v, indent, target)
	case *ast.VarDeclStmt:
		v := g.flowExpr(s.Value, indent)
		if v == "" {
			return
		}
		g.line(indent, "var %s %s = %s", localName(s.Name), goType(g.model.VarTypes[s]), v)
	case *ast.AssignStmt:
		if index, ok := s.Target.(*ast.IndexExpr); ok && g.model.ExprTypes[index.Object].Kind == sema.Map {
			object := g.flowExpr(index.Object, indent)
			if object == "" {
				return
			}
			key := g.flowExpr(index.Index, indent)
			if key == "" {
				return
			}
			value := g.flowExpr(s.Value, indent)
			if value != "" {
				g.line(indent, "%s[%s] = %s", object, key, value)
			}
			return
		}
		// Capture an address before the RHS, preserving indexing and receiver order.
		address := g.freshName()
		g.line(indent, "%s := &(%s)", address, g.flowTarget(s.Target, indent))
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
		g.closeScopes(indent, g.loopScopeDepth)
		g.line(indent, "continue")
	case *ast.BreakStmt:
		g.closeScopes(indent, g.loopScopeDepth)
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
		if g.model.GlobalRefs[v] != nil {
			return g.globalName(v.Name)
		}
		return localName(v.Name)
	case *ast.ReceiverExpr:
		return "_self." + g.fieldName(sema.Type{Kind: sema.Named, Name: g.receiver}, v.Name)
	case *ast.MemberExpr:
		if stdlib.IsValue(g.model.ExprTypes[v.Object].Name) {
			return g.flowTarget(v.Object, indent) + "." + g.fieldName(g.model.ExprTypes[v.Object], v.Name)
		}
		return g.flowExpr(v.Object, indent) + "." + g.fieldName(g.model.ExprTypes[v.Object], v.Name)
	case *ast.IndexExpr:
		object := g.flowExpr(v.Object, indent)
		index := g.flowExpr(v.Index, indent)
		return object + "[" + index + "]"
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
	}
	g.flowBlock(b.Body, indent+1, target)
	if len(branches) > 1 || len(otherwise) > 0 {
		g.line(indent, "} else {")
		g.flowIf(branches[1:], otherwise, indent+1, target)
	}
	g.line(indent, "}")
}

func (g *generator) flowMatch(m *ast.MatchExpr, indent int, target string) {
	if g.model.ExprTypes[m.Value].Kind == sema.Interface {
		g.flowTypeMatch(m, indent, target)
		return
	}
	if g.scalarMatch(m) {
		value := g.flowExpr(m.Value, indent)
		g.line(indent, "switch %s {", value)
		wildcard := false
		for _, arm := range m.Arms {
			g.scalarCase(arm, indent)
			wildcard = wildcard || arm.Pattern == "_"
			g.flowBlock(arm.Body, indent+1, target)
		}
		if !wildcard {
			g.line(indent, "default: panic(\"valor de casos inválido\")")
		}
		g.line(indent, "}")
		return
	}
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
	return name
}

func (g *generator) flowExpr(e ast.Expr, indent int) string {
	t := g.model.ExprTypes[e]
	rawType := t
	wrap, wrapped := g.model.Wraps[e]
	if wrapped {
		rawType = *wrap.Elem
	}
	if raw, ok := g.model.RawTypes[e]; ok {
		rawType = raw
	}
	if _, ok := g.model.NumericCoercions[e]; ok {
		rawType = sema.Type{Kind: sema.Integer}
	}
	value := g.flowRaw(e, rawType, indent)
	if value != "" {
		if _, ok := g.model.NumericCoercions[e]; ok {
			value = "float64(" + value + ")"
		}
	}
	if value != "" && rawType.Numeric() && (t.Kind == sema.Interface || wrapped && wrap.Elem.Kind == sema.Interface) {
		value = goType(rawType) + "(" + value + ")"
	}
	if wrapped {
		value = goType(wrap) + "{tag: 1, payload1: " + value + "}"
	}
	if value == "" {
		return ""
	}
	return g.capture(t, value, indent)
}

func (g *generator) flowRaw(e ast.Expr, t sema.Type, indent int) string {
	if value, ok := g.model.Game.Constants[e]; ok {
		return value
	}
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
		return "&" + namedGoType(ctor.Enum.Type) + "{" + fields + "}"
	}
	switch v := e.(type) {
	case *ast.AssertExpr:
		value := g.flowExpr(v.Value, indent)
		if value == "" {
			return ""
		}
		payload, ok, result := g.freshName(), g.freshName(), g.freshName()
		g.line(indent, "%s, %s := %s.(%s)", payload, ok, value, goType(*t.Elem))
		g.line(indent, "var %s %s", result, goType(t))
		g.line(indent, "if %s { %s = %s{tag: 1, payload1: %s} }", ok, result, goType(t), payload)
		return result
	case *ast.ReturnExpr:
		if v.Value == nil {
			g.closeScopes(indent, 0)
			g.line(indent, "return")
		} else {
			value := g.flowExpr(v.Value, indent)
			if value != "" {
				if len(g.scopes) > 0 {
					name := g.freshName()
					g.line(indent, "%s := %s", name, value)
					value = name
				}
				g.closeScopes(indent, 0)
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
		if len(g.scopes) > 0 {
			name := g.freshName()
			g.line(indent+1, "%s := %s", name, failure)
			failure = name
		}
		g.closeScopes(indent+1, 0)
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
		if g.model.GlobalRefs[v] != nil {
			return g.globalName(v.Name)
		}
		return localName(v.Name)
	case *ast.ReceiverExpr:
		if v.Name == "" {
			return "_self"
		}
		return "_self." + g.fieldName(sema.Type{Kind: sema.Named, Name: g.receiver}, v.Name)
	case *ast.LiteralExpr:
		if v.Kind == "bool" {
			if v.Value == "verdadero" {
				return "true"
			}
			return "false"
		}
		return v.Value
	case *ast.InterpolatedStringExpr:
		return g.interpolated(v, func(e ast.Expr) string { return g.flowExpr(e, indent) })
	case *ast.MemberExpr:
		return g.flowExpr(v.Object, indent) + "." + g.fieldName(g.model.ExprTypes[v.Object], v.Name)
	case *ast.IndexExpr:
		object := g.flowExpr(v.Object, indent)
		index := g.flowExpr(v.Index, indent)
		if g.model.ExprTypes[v.Object].Kind == sema.Map {
			if object == "" || index == "" {
				return ""
			}
			return g.mapLookup(object, index, t, indent)
		}
		return object + "[" + index + "]"
	case *ast.UnaryExpr:
		if lit, ok := v.Value.(*ast.LiteralExpr); ok && v.Operator == "-" && lit.Kind == "entero" && lit.Value == "9223372036854775808" {
			return "int64(-9223372036854775808)"
		}
		if t.Name == stdlib.Symbol("Vec2") {
			return "_hgscale(" + g.flowExpr(v.Value, indent) + ", -1)"
		}
		return "(" + v.Operator + g.flowExpr(v.Value, indent) + ")"
	case *ast.BinaryExpr:
		if compare, ok := g.model.EnumCompares[v]; ok {
			value := g.flowExpr(compare.Value, indent)
			if value == "" {
				return ""
			}
			return "(" + value + ".tag " + v.Operator + " " + fmt.Sprint(compare.Tag) + ")"
		}
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
		if t.Name == stdlib.Symbol("Vec2") {
			switch v.Operator {
			case "+":
				return "_hgadd(" + left + "," + right + ")"
			case "-":
				return "_hgsub(" + left + "," + right + ")"
			case "*":
				if g.model.ExprTypes[v.Left].Name == stdlib.Symbol("Vec2") {
					return "_hgscale(" + left + "," + right + ")"
				}
				return "_hgscale(" + right + "," + left + ")"
			case "/":
				return "_hgscale(" + left + ",1/" + right + ")"
			}
		}
		return "(" + left + " " + v.Operator + " " + right + ")"
	case *ast.CallExpr:
		return g.flowCall(v, indent)
	case *ast.StructLiteralExpr:
		explicit := g.literalFields(t, v)
		var fields []string
		for _, f := range explicit {
			value := g.flowExpr(f.Value, indent)
			if value == "" {
				return ""
			}
			fields = append(fields, g.fieldName(t, f.Name)+": "+value)
		}
		fields = append(fields, g.defaultFields(t, explicit, indent)...)
		if stdlib.IsValue(t.Name) {
			return goType(t) + "{" + strings.Join(fields, ", ") + "}"
		}
		return "&" + namedGoType(t) + "{" + strings.Join(fields, ", ") + "}"
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
	case *ast.MapLiteralExpr:
		if t.Kind == sema.Never {
			for _, entry := range v.Entries {
				if g.flowExpr(entry.Key, indent) == "" {
					return ""
				}
				if g.flowExpr(entry.Value, indent) == "" {
					return ""
				}
			}
			return ""
		}
		result := g.freshName()
		g.line(indent, "%s := make(%s)", result, goType(t))
		for _, entry := range v.Entries {
			key := g.flowExpr(entry.Key, indent)
			if key == "" {
				return ""
			}
			value := g.flowExpr(entry.Value, indent)
			if value == "" {
				return ""
			}
			g.line(indent, "%s[%s] = %s", result, key, value)
		}
		return result
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
	if operation, ok := g.model.MapCalls[call]; ok {
		return g.flowMapCall(call, operation, indent)
	}
	if t, ok := g.model.NumericCalls[call]; ok {
		return g.numericCall(call, t, g.flowExpr(call.Args[0], indent))
	}
	if f, ok := g.model.Game.Calls[call]; ok {
		return g.gameCall(call, f, indent)
	}
	if operation, ok := g.model.ListCalls[call]; ok {
		return g.flowListCall(call, operation, indent)
	}
	if operation, ok := g.model.StringCalls[call]; ok {
		return g.flowStringCall(call, operation, indent)
	}
	if operation, ok := g.model.NumberCalls[call]; ok {
		return g.flowNumberCall(call, operation, indent)
	}
	if _, ok := g.model.StructCopies[call]; ok {
		// Structs are pointers, so a shallow copy duplicates the pointed-to value.
		original := g.flowExpr(call.Callee.(*ast.MemberExpr).Object, indent)
		duplicate := g.freshName()
		g.line(indent, "%s := *%s", duplicate, original)
		return "&" + duplicate
	}
	callee := ""
	switch v := call.Callee.(type) {
	case *ast.InstantiateExpr:
		callee = exported(v.Name)
	case *ast.IdentExpr:
		callee = exported(v.Name)
		if v.Name == "imprimir" {
			callee = "_hsimprimir"
		}
		if v.Name == "inicio" {
			callee = "main"
		}
	case *ast.ReceiverExpr:
		callee = "_self." + exported(v.Name)
	case *ast.MemberExpr:
		callee = g.flowExpr(v.Object, indent) + "." + exported(v.Name)
	}
	info := g.model.Calls[call]
	callee = g.promotedCallee(callee, info.Signature, call.Callee)
	if info.Signature.Decl != nil && info.Signature.Decl.Receiver != "" && len(g.defaultFlags[info.Signature.Decl]) > 0 {
		callee = strings.TrimSuffix(callee, exported(info.Signature.Decl.Name)) + defaultMethod(info.Signature.Decl)
	}
	callee += typeArguments(info.Signature.TypeArgs)
	fn := g.freshName()
	g.line(indent, "%s := %s", fn, callee)
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

func (g *generator) flowListCall(call *ast.CallExpr, operation string, indent int) string {
	member := call.Callee.(*ast.MemberExpr)
	listType := g.model.ExprTypes[member.Object]
	mutating := operation == "agregar" || operation == "extender" || operation == "insertar" || operation == "eliminar" || operation == "invertir"
	var address string
	var receiver string
	if mutating {
		address = g.freshName()
		g.line(indent, "%s := &(%s)", address, g.flowTarget(member.Object, indent))
		receiver = g.capture(listType, "*"+address, indent)
	} else {
		receiver = g.flowExpr(member.Object, indent)
	}

	info := g.model.Calls[call]
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = g.flowExpr(arg, indent)
		if args[i] == "" {
			return ""
		}
	}
	ordered := make([]string, len(info.Signature.Params))
	for i, parameter := range info.Parameters {
		ordered[parameter] = args[i]
	}

	switch operation {
	case "longitud":
		return "int64(len(" + receiver + "))"
	case "esta_vacia":
		return "len(" + receiver + ") == 0"
	case "contiene", "buscar_indice":
		found := g.freshName()
		if operation == "contiene" {
			g.line(indent, "%s := false", found)
		} else {
			g.line(indent, "var %s %s", found, goType(info.Signature.Return))
		}
		index := g.freshName()
		g.line(indent, "for %s := range %s {", index, receiver)
		g.line(indent+1, "if %s[%s] == %s {", receiver, index, ordered[0])
		if operation == "contiene" {
			g.line(indent+2, "%s = true", found)
		} else {
			g.line(indent+2, "%s = %s{tag: 1, payload1: int64(%s)}", found, goType(info.Signature.Return), index)
		}
		g.line(indent+2, "break")
		g.line(indent+1, "}")
		g.line(indent, "}")
		return found
	case "obtener":
		result := g.freshName()
		g.line(indent, "var %s %s", result, goType(info.Signature.Return))
		condition := validListIndex(ordered[0], receiver, true)
		g.line(indent, "if %s {", condition)
		g.line(indent+1, "%s = %s{tag: 1, payload1: %s[int(%s)]}", result, goType(info.Signature.Return), receiver, ordered[0])
		g.line(indent, "}")
		return result
	case "primero", "ultimo":
		result := g.freshName()
		g.line(indent, "var %s %s", result, goType(info.Signature.Return))
		g.line(indent, "if len(%s) > 0 {", receiver)
		index := "0"
		if operation == "ultimo" {
			index = "len(" + receiver + ")-1"
		}
		g.line(indent+1, "%s = %s{tag: 1, payload1: %s[%s]}", result, goType(info.Signature.Return), receiver, index)
		g.line(indent, "}")
		return result
	case "agregar":
		g.line(indent, "*%s = append(%s, %s)", address, receiver, ordered[0])
		return ""
	case "extender":
		g.line(indent, "*%s = append(%s, %s...)", address, receiver, ordered[0])
		return ""
	case "insertar":
		success := g.freshName()
		g.line(indent, "%s := false", success)
		g.line(indent, "if %s {", validListIndex(ordered[0], receiver, false))
		index := "int(" + ordered[0] + ")"
		updated := g.freshName()
		g.line(indent+1, "%s := append(%s, %s)", updated, receiver, ordered[1])
		g.line(indent+1, "copy(%s[%s+1:], %s[%s:])", updated, index, updated, index)
		g.line(indent+1, "%s[%s] = %s", updated, index, ordered[1])
		g.line(indent+1, "*%s = %s", address, updated)
		g.line(indent+1, "%s = true", success)
		g.line(indent, "}")
		return success
	case "eliminar":
		success := g.freshName()
		g.line(indent, "%s := false", success)
		g.line(indent, "if %s {", validListIndex(ordered[0], receiver, true))
		index := "int(" + ordered[0] + ")"
		g.line(indent+1, "*%s = append(%s[:%s], %s[%s+1:]...)", address, receiver, index, receiver, index)
		g.line(indent+1, "%s = true", success)
		g.line(indent, "}")
		return success
	case "copiar":
		return "append(make(" + goType(listType) + ", 0, len(" + receiver + ")), " + receiver + "...)"
	case "invertir":
		left, right := g.freshName(), g.freshName()
		g.line(indent, "for %s, %s := 0, len(%s)-1; %s < %s; %s, %s = %s+1, %s-1 {", left, right, receiver, left, right, left, right, left, right)
		g.line(indent+1, "%s[%s], %s[%s] = %s[%s], %s[%s]", receiver, left, receiver, right, receiver, right, receiver, left)
		g.line(indent, "}")
		g.line(indent, "*%s = %s", address, receiver)
		return ""
	}
	panic("unknown list operation: " + operation)
}

func validListIndex(index, receiver string, existing bool) string {
	bound := "<="
	if existing {
		bound = "<"
	}
	return index + " >= 0 && " + index + " " + bound + " int64(len(" + receiver + "))"
}

func (g *generator) flowStringCall(call *ast.CallExpr, operation string, indent int) string {
	member := call.Callee.(*ast.MemberExpr)
	receiver := g.capture(sema.Type{Kind: sema.String}, g.flowExpr(member.Object, indent), indent)
	info := g.model.Calls[call]
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = g.capture(g.model.ExprTypes[arg], g.flowExpr(arg, indent), indent)
	}
	ordered := make([]string, len(info.Signature.Params))
	for i, parameter := range info.Parameters {
		ordered[parameter] = args[i]
	}
	switch operation {
	case "longitud":
		return "int64(len([]rune(" + receiver + ")))"
	case "esta_vacia":
		return "len(" + receiver + ") == 0"
	case "contiene":
		return "strings.Contains(" + receiver + ", " + ordered[0] + ")"
	case "empieza_con":
		return "strings.HasPrefix(" + receiver + ", " + ordered[0] + ")"
	case "termina_con":
		return "strings.HasSuffix(" + receiver + ", " + ordered[0] + ")"
	case "mayusculas":
		return "strings.ToUpper(" + receiver + ")"
	case "minusculas":
		return "strings.ToLower(" + receiver + ")"
	case "recortar":
		return "strings.TrimSpace(" + receiver + ")"
	case "reemplazar":
		return "strings.ReplaceAll(" + receiver + ", " + ordered[0] + ", " + ordered[1] + ")"
	case "dividir":
		return "strings.Split(" + receiver + ", " + ordered[0] + ")"
	case "buscar_indice":
		result, index := g.freshName(), g.freshName()
		g.line(indent, "var %s %s", result, goType(info.Signature.Return))
		g.line(indent, "if %s := _hsindice(%s, %s); %s >= 0 {", index, receiver, ordered[0], index)
		g.line(indent+1, "%s = %s{tag: 1, payload1: int64(%s)}", result, goType(info.Signature.Return), index)
		g.line(indent, "}")
		return result
	case "a_entero", "a_decimal":
		result, parsed := g.freshName(), g.freshName()
		g.line(indent, "var %s %s", result, goType(info.Signature.Return))
		if operation == "a_entero" {
			g.line(indent, "if %s, err := strconv.ParseInt(%s, 10, 64); err == nil {", parsed, receiver)
		} else {
			// v-v is 0 only for finite numbers, which rejects "inf" and "nan".
			g.line(indent, "if %s, err := strconv.ParseFloat(%s, 64); err == nil && %s-%s == 0 {", parsed, receiver, parsed, parsed)
		}
		g.line(indent+1, "%s = %s{tag: 1, payload1: %s}", result, goType(info.Signature.Return), parsed)
		g.line(indent, "}")
		return result
	case "obtener", "subcadena":
		result, runes := g.freshName(), g.freshName()
		g.line(indent, "var %s %s", result, goType(info.Signature.Return))
		g.line(indent, "%s := []rune(%s)", runes, receiver)
		condition := validRuneIndex(ordered[0], "len("+runes+")", operation == "obtener")
		value := "string(" + runes + "[int(" + ordered[0] + ")])"
		if operation == "subcadena" {
			condition += " && " + validRuneEnd(ordered[1], "len("+runes+")") + " && " + ordered[0] + " <= " + ordered[1]
			value = "string(" + runes + "[int(" + ordered[0] + "):int(" + ordered[1] + ")])"
		}
		g.line(indent, "if %s {", condition)
		g.line(indent+1, "%s = %s{tag: 1, payload1: %s}", result, goType(info.Signature.Return), value)
		g.line(indent, "}")
		return result
	}
	panic("unknown string operation: " + operation)
}

func (g *generator) flowNumberCall(call *ast.CallExpr, operation string, indent int) string {
	member := call.Callee.(*ast.MemberExpr)
	receiverType := g.model.ExprTypes[member.Object]
	receiver := g.capture(receiverType, g.flowExpr(member.Object, indent), indent)
	info := g.model.Calls[call]
	args := make([]string, len(call.Args))
	for i, arg := range call.Args {
		args[i] = g.capture(g.model.ExprTypes[arg], g.flowExpr(arg, indent), indent)
	}
	ordered := make([]string, len(info.Signature.Params))
	for i, parameter := range info.Parameters {
		ordered[parameter] = args[i]
	}
	switch operation {
	case "formato":
		return "_hsformatoNumero(float64(" + receiver + "), " + ordered[0] + ")"
	}
	panic("unknown number operation: " + operation)
}

func validRuneIndex(index, length string, existing bool) string {
	op := "<="
	if existing {
		op = "<"
	}
	return index + " >= 0 && " + index + " " + op + " int64(" + length + ")"
}

func validRuneEnd(index, length string) string { return validRuneIndex(index, length, false) }

func (g *generator) literalFields(t sema.Type, literal *ast.StructLiteralExpr) []ast.FieldValue {
	if len(literal.Values) == 0 {
		return literal.Fields
	}
	info := g.model.StructInfo(t)
	fields := make([]ast.FieldValue, len(literal.Values))
	for index, value := range literal.Values {
		fields[index] = ast.FieldValue{Pos: value.Position(), Name: info.Decl.Fields[index].Name, Value: value}
	}
	return fields
}

func (g *generator) defaultFields(t sema.Type, supplied []ast.FieldValue, indent int) []string {
	seen := map[string]bool{}
	for _, f := range supplied {
		seen[f.Name] = true
	}
	var fields []string
	info := g.model.StructInfo(t)
	for _, field := range info.Decl.Fields {
		if seen[field.Name] {
			continue
		}
		ft := info.Fields[field.Name].Type
		if field.Default != nil {
			value := ""
			if g.flow {
				value = g.flowExpr(field.Default, indent)
			}
			if value == "" {
				value = g.expr(field.Default)
			}
			fields = append(fields, g.fieldName(t, field.Name)+": "+value)
		} else if ft.Kind == sema.Map {
			fields = append(fields, g.fieldName(t, field.Name)+": make("+goType(ft)+")")
		} else if ft.Kind == sema.Named {
			if stdlib.IsValue(ft.Name) {
				fields = append(fields, g.fieldName(t, field.Name)+": "+goType(ft)+"{"+strings.Join(g.defaultFields(ft, nil, indent), ", ")+"}")
				continue
			}
			fields = append(fields, g.fieldName(t, field.Name)+": &"+namedGoType(ft)+"{"+strings.Join(g.defaultFields(ft, nil, indent), ", ")+"}")
		}
	}
	if t.Name == stdlib.Symbol("Camara2D") && !seen["zoom"] {
		fields = append(fields, "Zoom: 1")
	}
	return fields
}

func (g *generator) flowLoop(s *ast.RepeatStmt, indent int) {
	outerDepth := g.loopScopeDepth
	g.loopScopeDepth = len(g.scopes)
	defer func() { g.loopScopeDepth = outerDepth }()
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
		g.line(indent, "%s := int64(1)", step)
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
		g.line(indent, "for %s, %s := %s, int64(0); (%s > 0 && %s < %s) || (%s < 0 && %s > %s); %s, %s = %s + %s, %s + 1 {", current, index, start, step, current, end, step, current, end, current, index, current, step, index)
		if s.Element != "" {
			g.line(indent+1, "%s := %s", localName(s.Element), current)
		}
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
			if g.model.ExprTypes[s.Iterable].Kind == sema.Map {
				g.line(indent+1, "%s := %s", localName(s.Index), index)
			} else {
				g.line(indent+1, "%s := int64(%s)", localName(s.Index), index)
			}
		}
	}
	if s.Element != "" {
	}
	if s.Index != "" {
	}
	g.flowBlock(s.Body, indent+1, "")
	g.line(indent, "}")
}

func (g *generator) closeScopes(indent, depth int) {
	for i := len(g.scopes) - 1; i >= depth; i-- {
		g.line(indent, "%s.Salir()", g.scopes[i])
	}
}

// Local block exits also include loop control, which must not be treated as
// function returns by semantic result checking.
func (g *generator) leavesBlock(stmt ast.Stmt) bool {
	block := func(body []ast.Stmt) bool {
		for _, s := range body {
			if g.leavesBlock(s) {
				return true
			}
		}
		return false
	}
	switch s := stmt.(type) {
	case *ast.BreakStmt, *ast.ContinueStmt:
		return true
	case *ast.ScopeStmt:
		return block(s.Body)
	case *ast.IfStmt:
		if len(s.Else) == 0 || !block(s.Else) {
			return false
		}
		for _, b := range s.Branches {
			if !block(b.Body) {
				return false
			}
		}
		return true
	case *ast.MatchStmt:
		if len(s.Match.Arms) == 0 {
			return false
		}
		for _, a := range s.Match.Arms {
			if !block(a.Body) {
				return false
			}
		}
		return true
	}
	return g.model.Terminates(stmt)
}
