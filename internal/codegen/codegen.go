package codegen

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"cometa/internal/ast"
	"cometa/internal/sema"
	"cometa/internal/stdlib"
)

type generator struct {
	receiver       string
	flow           bool
	returnType     sema.Type
	buffer         bytes.Buffer
	model          *sema.Model
	nextName       int
	loopLabel      string
	scopes         []string
	loopScopeDepth int
	defaultFlags   map[*ast.FuncDecl][]string
}

func Generate(filename string, program *ast.Program, model *sema.Model) ([]byte, error) {

	g := &generator{model: model, defaultFlags: map[*ast.FuncDecl][]string{}}
	g.flow = model.HasScopes
	if model.Game.Used {
		g.flow = true
	}
	if len(model.ListCalls) > 0 {
		g.flow = true
	}
	stringFeatures := len(model.StringCalls) > 0
	if stringFeatures {
		g.flow = true
	}
	for expr, t := range model.ExprTypes {
		if t.Wrapped() || t.Kind == sema.Map {
			g.flow = true
		}
		switch expr.(type) {
		case *ast.ReturnExpr, *ast.TryExpr, *ast.RecoverExpr, *ast.BlockExpr:
			g.flow = true
		case *ast.InterpolatedStringExpr:
			g.flow = true
			stringFeatures = true
		}
	}
	register := func(decl *ast.FuncDecl) {
		for _, param := range decl.Params {
			if param.Default != nil {
				g.defaultFlags[decl] = append(g.defaultFlags[decl], g.freshName())
			}
		}
	}
	for _, decl := range program.Decls {
		if len(model.TypeParams[decl]) > 0 {
			g.flow = true
		}
		switch d := decl.(type) {
		case *ast.InterfaceDecl:
			g.flow = true
		case *ast.FuncDecl:
			register(d)
		case *ast.TypeDecl:
			for _, method := range d.Methods {
				register(method)
			}
		}
	}
	g.line(0, "package main")
	g.line(0, "")
	runtimeSource, runtimeImports, err := stdlib.Runtime(model.Game.Modules)
	if err != nil {
		return nil, err
	}
	imports := map[string]bool{}
	for _, path := range runtimeImports {
		imports[path] = true
	}
	if usesPrint(program) {
		imports[`"fmt"`] = true
	}
	if stringFeatures {
		imports[`"strconv"`] = true
		imports[`"strings"`] = true
	}
	var paths []string
	for path := range imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if len(paths) == 1 {
		g.line(0, "import %s", paths[0])
		g.line(0, "")
	} else if len(paths) > 1 {
		g.line(0, "import (")
		for _, path := range paths {
			g.line(1, "%s", path)
		}
		g.line(0, ")")
		g.line(0, "")
	}
	if runtimeSource != "" {
		g.write("%s\n", runtimeSource)
		var nativeNames []string
		for name := range model.Types {
			if native := stdlib.GoType(name); native != "" && strings.Contains(runtimeSource, "type "+strings.TrimPrefix(native, "*")+" ") {
				nativeNames = append(nativeNames, name)
			}
		}
		sort.Strings(nativeNames)
		for _, name := range nativeNames {
			g.emitVisibilityMethods(sema.Type{Kind: sema.Named, Name: name})
		}
	}
	if stringFeatures {
		g.write("%s\n", stringRuntime)
	}
	if len(model.NumericCalls) > 0 || model.Game.Used {
		g.write("%s\n", numericRuntime)
	}

	for _, decl := range program.Decls {
		if d, ok := decl.(*ast.InterfaceDecl); ok {
			g.emitInterface(d)
		}
		if enumDecl, ok := decl.(*ast.EnumDecl); ok {
			g.emitEnum(enumDecl)
		}
		if typeDecl, ok := decl.(*ast.TypeDecl); ok {
			g.emitType(typeDecl)
		}
	}
	for _, decl := range program.Decls {
		if global, ok := decl.(*ast.GlobalDecl); ok {
			g.emitGlobal(global)
		}
	}
	for _, decl := range program.Decls {
		switch declaration := decl.(type) {
		case *ast.TypeDecl:
			for _, method := range declaration.Methods {
				g.emitFunction(method)
			}
		case *ast.FuncDecl:
			g.emitFunction(declaration)
		}
	}

	formatted, err := eliminateUnusedLocals(g.buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("no se pudo formatear el Go generado: %w\n%s", err, g.buffer.String())
	}
	if !strings.Contains(strings.Join(runtimeImports, " "), "github.com/") {
		if err = validate(filename, formatted); err != nil {
			return nil, err
		}
	}
	return formatted, nil
}

func (g *generator) emitGlobal(decl *ast.GlobalDecl) {
	info := g.model.Globals[decl.Name]
	if decl.Constant {
		g.line(0, "const %s %s = %s", g.globalName(decl.Name), goType(info.Type), g.expr(decl.Value))
		g.line(0, "")
		return
	}
	child := &generator{model: g.model, flow: true, nextName: g.nextName, defaultFlags: g.defaultFlags}
	child.line(0, "func() %s {", goType(info.Type))
	value := child.flowExpr(decl.Value, 1)
	child.line(1, "return %s", value)
	child.line(0, "}()")
	g.nextName = child.nextName
	g.line(0, "var %s %s = %s", g.globalName(decl.Name), goType(info.Type), strings.TrimSpace(child.buffer.String()))
	g.line(0, "")
}

func (g *generator) emitType(decl *ast.TypeDecl) {
	g.line(0, "type %s%s struct {", exported(decl.Name), g.typeParams(decl))
	info := g.model.Types[decl.Name]
	for _, field := range decl.Fields {
		if field.Embedded {
			g.line(1, "%s", goType(info.Fields[field.Name].Type))
			continue
		}
		g.line(1, "%s %s", exported(field.Name), goType(info.Fields[field.Name].Type))
	}
	g.line(0, "}")
	g.line(0, "")
	g.emitVisibilityMethods(sema.Type{Kind: sema.Named, Name: decl.Name, Args: g.model.TypeParams[decl]})
}

func (g *generator) emitFunction(decl *ast.FuncDecl) {
	g.receiver = decl.Receiver
	var signature sema.FuncInfo
	if decl.Receiver != "" {
		signature = g.model.Types[decl.Receiver].Methods[decl.Name]
		name := exported(decl.Name)
		if len(g.defaultFlags[decl]) > 0 {
			g.emitDefaultBridge(decl, signature)
			name = defaultMethod(decl)
		}
		g.write("func (_self *%s%s) %s(", exported(decl.Receiver), typeArguments(g.model.TypeParams[g.model.Types[decl.Receiver].Decl]), name)
	} else {
		signature = g.model.Functions[decl.Name]
		name := exported(decl.Name)
		if decl.Name == "inicio" {
			name = "main"
		}
		g.write("func %s%s(", name, g.typeParams(decl))
	}
	for _, flag := range g.defaultFlags[decl] {
		g.write("%s bool, ", flag)
	}
	for index, param := range decl.Params {
		if index > 0 {
			g.write(", ")
		}
		paramType := goType(signature.Params[index])
		if param.Variadic {
			paramType = "..." + goType(*signature.Params[index].Elem)
		}
		g.write("%s %s", localName(param.Name), paramType)
	}
	g.write(")")
	if signature.Return.Kind != sema.Void {
		g.write(" %s", goType(signature.Return))
	}
	g.write(" {\n")
	g.returnType = signature.Return
	flagIndex := 0
	for _, param := range decl.Params {
		if param.Default != nil {
			g.line(1, "if %s {", g.defaultFlags[decl][flagIndex])
			value := ""
			if g.flow {
				value = g.flowExpr(param.Default, 2)
			} else {
				value = g.expr(param.Default)
			}
			g.line(2, "%s = %s", localName(param.Name), value)
			g.line(1, "}")
			flagIndex++
		}
	}
	if g.flow {
		target := ""
		if signature.Return.Kind != sema.Void {
			target = "return"
		}
		g.flowBlock(decl.Body, 1, target)
	} else {
		g.emitBlock(decl.Body, 1, signature.Return.Kind != sema.Void)
	}
	g.line(0, "}")
	g.line(0, "")
}

func (g *generator) emitBlock(body []ast.Stmt, indent int, returnFinal bool) {
	for index, stmt := range body {
		g.emitStmt(stmt, indent, returnFinal && index == len(body)-1)
	}
}

func (g *generator) emitStmt(stmt ast.Stmt, indent int, returnValue bool) {
	switch statement := stmt.(type) {
	case *ast.MatchStmt:
		g.emitMatch(statement.Match, indent, returnValue)
	case *ast.ExprStmt:
		expression := g.expr(statement.Expr)
		_, constructor := g.model.Constructors[statement.Expr]
		if call, ok := statement.Expr.(*ast.CallExpr); ok {
			_, numeric := g.model.NumericCalls[call]
			constructor = constructor || numeric
		}
		if returnValue {
			g.line(indent, "return %s", expression)
		} else if _, call := statement.Expr.(*ast.CallExpr); call && !constructor {
			g.line(indent, "%s", expression)
		} else {
			g.line(indent, "_ = %s", expression)
		}
	case *ast.AssignStmt:
		g.line(indent, "%s = %s", g.expr(statement.Target), g.expr(statement.Value))
	case *ast.VarDeclStmt:
		name := localName(statement.Name)
		value := g.expr(statement.Value)
		if g.model.VarTypes[statement].Numeric() {
			value = goType(g.model.VarTypes[statement]) + "(" + value + ")"
		}
		g.line(indent, "%s := %s", name, value)
	case *ast.IfStmt:
		g.emitIf(statement, indent, returnValue)
	case *ast.RepeatStmt:
		g.emitRepeat(statement, indent)
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

func (g *generator) emitRepeat(stmt *ast.RepeatStmt, indent int) {
	outerLabel := g.loopLabel
	g.loopLabel = ""
	defer func() { g.loopLabel = outerLabel }()
	if stmt.RangeEnd != nil {
		g.emitRange(stmt, indent)
		return
	}
	if needsLoopLabel(stmt.Body, false) {
		g.loopLabel = g.freshName()
		g.line(indent, "%s:", g.loopLabel)
	}
	if stmt.Iterable == nil {
		g.line(indent, "for {")
		g.emitBlock(stmt.Body, indent+1, false)
		g.line(indent, "}")
		return
	}

	element := localName(stmt.Element)
	if stmt.Index == "" {
		g.line(indent, "for _, %s := range %s {", element, g.expr(stmt.Iterable))
	} else {
		index := localName(stmt.Index)
		g.line(indent, "for %s, %s := range %s {", index, element, g.expr(stmt.Iterable))
		g.line(indent+1, "%s := int64(%s)", index, index)
	}
	g.emitBlock(stmt.Body, indent+1, false)
	g.line(indent, "}")
}

func (g *generator) emitRange(stmt *ast.RepeatStmt, indent int) {
	current, end, step, index := g.freshName(), g.freshName(), g.freshName(), g.freshName()
	// Evaluate bounds once, in source order, before introducing loop bindings.
	g.line(indent, "{")
	indent++
	g.line(indent, "%s := int64(%s)", current, g.expr(stmt.Iterable))
	g.line(indent, "%s := int64(%s)", end, g.expr(stmt.RangeEnd))
	g.line(indent, "%s := int64(1)", step)
	g.line(indent, "if %s > %s { %s = -1 }", current, end, step)
	if needsLoopLabel(stmt.Body, false) {
		g.loopLabel = g.freshName()
		g.line(indent, "%s:", g.loopLabel)
	}
	g.line(indent, "for %s := int64(0); (%s > 0 && %s < %s) || (%s < 0 && %s > %s); %s, %s = %s + %s, %s + 1 {", index, step, current, end, step, current, end, current, index, current, step, index)
	g.line(indent+1, "%s := %s", localName(stmt.Element), current)
	if stmt.Index != "" {
		g.line(indent+1, "%s := %s", localName(stmt.Index), index)
	}
	g.emitBlock(stmt.Body, indent+1, false)
	g.line(indent, "}")
	g.line(indent-1, "}")
}

func (g *generator) emitIf(stmt *ast.IfStmt, indent int, returnValue bool) {
	for index, branch := range stmt.Branches {
		prefix := "if"
		if index > 0 {
			prefix = "} else if"
		}
		g.line(indent, "%s %s {", prefix, g.expr(branch.Condition))
		g.emitBlock(branch.Body, indent+1, returnValue)
	}
	if len(stmt.Else) > 0 {
		g.line(indent, "} else {")
		g.emitBlock(stmt.Else, indent+1, returnValue)
	}
	g.line(indent, "}")
}

func (g *generator) expr(expr ast.Expr) string {
	value := g.rawExpr(expr)
	if _, ok := g.model.NumericCoercions[expr]; ok {
		return "float64(" + value + ")"
	}
	return value
}

func (g *generator) rawExpr(expr ast.Expr) string {
	if value, ok := g.model.Game.Constants[expr]; ok {
		return value
	}
	if constructor, ok := g.model.Constructors[expr]; ok {
		fields := fmt.Sprintf("tag: %d", constructor.Variant.Tag)
		if call, ok := expr.(*ast.CallExpr); ok {
			fields += fmt.Sprintf(", payload%d: %s", constructor.Variant.Tag, g.expr(call.Args[0]))
		}
		return "(&" + exported(constructor.Enum.Decl.Name) + "{" + fields + "})"
	}
	switch expression := expr.(type) {
	case *ast.MatchExpr:
		child := &generator{model: g.model, nextName: g.nextName}
		raw := g.model.ExprTypes[expr]
		if _, ok := g.model.NumericCoercions[expr]; ok {
			raw = sema.Type{Kind: sema.Integer}
		}
		child.line(0, "func() %s {", goType(raw))
		child.emitMatch(expression, 1, true)
		child.line(0, "}()")
		g.nextName = child.nextName
		return strings.TrimSpace(child.buffer.String())
	case *ast.IdentExpr:
		if g.model.GlobalRefs[expression] != nil {
			return g.globalName(expression.Name)
		}
		return localName(expression.Name)
	case *ast.ReceiverExpr:
		if expression.Name == "" {
			return "_self"
		}
		return "_self." + g.fieldName(sema.Type{Kind: sema.Named, Name: g.receiver}, expression.Name)
	case *ast.MemberExpr:
		return g.expr(expression.Object) + "." + g.fieldName(g.model.ExprTypes[expression.Object], expression.Name)
	case *ast.IndexExpr:
		return g.expr(expression.Object) + "[" + g.expr(expression.Index) + "]"
	case *ast.LiteralExpr:
		if expression.Kind == "bool" {
			if expression.Value == "verdadero" {
				return "true"
			}
			return "false"
		}
		return expression.Value
	case *ast.InterpolatedStringExpr:
		return g.interpolated(expression, func(e ast.Expr) string { return g.expr(e) })
	case *ast.UnaryExpr:
		return "(" + expression.Operator + g.expr(expression.Value) + ")"
	case *ast.BinaryExpr:
		return "(" + g.expr(expression.Left) + " " + expression.Operator + " " + g.expr(expression.Right) + ")"
	case *ast.CallExpr:
		if t, ok := g.model.NumericCalls[expression]; ok {
			return g.numericCall(expression, t, g.expr(expression.Args[0]))
		}
		var callee string
		switch called := expression.Callee.(type) {
		case *ast.IdentExpr:
			if called.Name == "imprimir" {
				callee = "fmt.Println"
			} else if called.Name == "inicio" {
				callee = "main"
			} else {
				callee = exported(called.Name)
			}
		case *ast.ReceiverExpr:
			callee = "_self." + exported(called.Name)
		case *ast.MemberExpr:
			callee = g.expr(called)
		}
		return g.call(expression, callee)
	case *ast.StructLiteralExpr:
		typeInfo := g.model.ExprTypes[expr]
		explicit := g.literalFields(typeInfo, expression)
		fields := make([]string, len(explicit))
		for index, field := range explicit {
			fields[index] = g.fieldName(typeInfo, field.Name) + ": " + g.expr(field.Value)
		}
		fields = append(fields, g.defaultFields(typeInfo, explicit)...)
		if stdlib.IsValue(typeInfo.Name) {
			return goType(typeInfo) + "{" + strings.Join(fields, ", ") + "}"
		}
		return "&" + namedGoType(typeInfo) + "{" + strings.Join(fields, ", ") + "}"
	case *ast.ListLiteralExpr:
		typeInfo := g.model.ExprTypes[expr]
		elements := make([]string, len(expression.Elements))
		for index, element := range expression.Elements {
			elements[index] = g.expr(element)
		}
		return goType(typeInfo) + "{" + strings.Join(elements, ", ") + "}"
	case *ast.IfExpr:
		resultType := goType(g.model.ExprTypes[expr])
		if _, ok := g.model.NumericCoercions[expr]; ok {
			resultType = "int64"
		}
		var out strings.Builder
		out.WriteString("func() ")
		out.WriteString(resultType)
		out.WriteString(" { ")
		for index, branch := range expression.Branches {
			if index == 0 {
				out.WriteString("if ")
			} else {
				out.WriteString(" else if ")
			}
			out.WriteString(g.expr(branch.Condition))
			out.WriteString(" { return ")
			out.WriteString(g.expr(branch.Value))
			out.WriteString(" }")
		}
		out.WriteString("; return ")
		out.WriteString(g.expr(expression.Else))
		out.WriteString(" }()")
		return out.String()
	default:
		panic(fmt.Sprintf("unsupported expression %T", expr))
	}
}

func (g *generator) interpolated(expression *ast.InterpolatedStringExpr, emit func(ast.Expr) string) string {
	parts := make([]string, 0, len(expression.Parts))
	for _, part := range expression.Parts {
		if part.Expr == nil {
			parts = append(parts, strconv.Quote(part.Text))
			continue
		}
		value := emit(part.Expr)
		if value == "" {
			return ""
		}
		switch g.model.ExprTypes[part.Expr].Kind {
		case sema.Decimal:
			value = "_hsnumero(" + value + ")"
		case sema.Integer:
			value = "strconv.FormatInt(" + value + ", 10)"
		case sema.Boolean:
			value = "_hsbool(" + value + ")"
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return `""`
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

func goType(t sema.Type) string {
	if t.Name == stdlib.Symbol("Juego") && t.Kind == sema.Interface {
		return "_hgJuego"
	}
	if name := stdlib.GoType(t.Name); name != "" && t.Kind == sema.Named {
		if stdlib.IsValue(t.Name) {
			return name
		}
		return "*" + name
	}
	switch t.Kind {
	case sema.Void, sema.Never:
		return "struct{}"
	case sema.Optional, sema.Result:
		fields := "struct { tag int"
		if t.Elem.Kind != sema.Void {
			fields += "; payload1 " + goType(*t.Elem)
		}
		if t.Kind == sema.Result {
			fields += "; payload2 " + goType(*t.Err)
		}
		return fields + " }"
	case sema.Decimal:
		return "float64"
	case sema.Integer:
		return "int64"
	case sema.String:
		return "string"
	case sema.Boolean:
		return "bool"
	case sema.TypeParameter:
		return "_T_" + t.Name
	case sema.Interface:
		if t.Name == "" {
			return "any"
		}
		return namedGoType(t)
	case sema.Named, sema.Enum:
		return "*" + namedGoType(t)
	case sema.Slice:
		return "[]" + goType(*t.Elem)
	case sema.Map:
		return "map[" + goType(*t.Key) + "]" + goType(*t.Elem)
	default:
		panic("invalid Go type")
	}
}

var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}

func exported(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return name
	}
	runes[0] = unicode.ToUpper(runes[0])
	result := string(runes)
	if goKeywords[result] {
		return "H_" + result
	}
	return result
}

func localName(name string) string {
	if goKeywords[name] || name == "_self" {
		return "h_" + name
	}
	return name
}

func usesPrint(program *ast.Program) bool {
	for _, decl := range program.Decls {
		switch declaration := decl.(type) {
		case *ast.FuncDecl:
			if functionUsesPrint(declaration) {
				return true
			}
		case *ast.TypeDecl:
			for _, method := range declaration.Methods {
				if functionUsesPrint(method) {
					return true
				}
			}
		case *ast.GlobalDecl:
			if exprUsesPrint(declaration.Value) {
				return true
			}
		}
	}
	return false
}

func functionUsesPrint(decl *ast.FuncDecl) bool {
	for _, param := range decl.Params {
		if param.Default != nil && exprUsesPrint(param.Default) {
			return true
		}
	}
	return blockUsesPrint(decl.Body)
}

func blockUsesPrint(body []ast.Stmt) bool {
	for _, stmt := range body {
		switch statement := stmt.(type) {
		case *ast.MatchStmt:
			if exprUsesPrint(statement.Match) {
				return true
			}
		case *ast.ExprStmt:
			if exprUsesPrint(statement.Expr) {
				return true
			}
		case *ast.AssignStmt:
			if exprUsesPrint(statement.Target) || exprUsesPrint(statement.Value) {
				return true
			}
		case *ast.VarDeclStmt:
			if exprUsesPrint(statement.Value) {
				return true
			}
		case *ast.IfStmt:
			for _, branch := range statement.Branches {
				if exprUsesPrint(branch.Condition) || blockUsesPrint(branch.Body) {
					return true
				}
			}
			if blockUsesPrint(statement.Else) {
				return true
			}
		case *ast.ScopeStmt:
			if exprUsesPrint(statement.Value) || blockUsesPrint(statement.Body) {
				return true
			}
		case *ast.RepeatStmt:
			if statement.RangeEnd != nil && exprUsesPrint(statement.RangeEnd) {
				return true
			}
			if statement.Iterable != nil && exprUsesPrint(statement.Iterable) {
				return true
			}
			if blockUsesPrint(statement.Body) {
				return true
			}
		}
	}
	return false
}

func exprUsesPrint(expr ast.Expr) bool {
	switch expression := expr.(type) {
	case *ast.AssertExpr:
		return exprUsesPrint(expression.Value)
	case *ast.ReturnExpr:
		return exprUsesPrint(expression.Value)
	case *ast.TryExpr:
		return exprUsesPrint(expression.Value)
	case *ast.RecoverExpr:
		return exprUsesPrint(expression.Value) || blockUsesPrint(expression.Body)
	case *ast.BlockExpr:
		return blockUsesPrint(expression.Body)
	case *ast.MatchExpr:
		if exprUsesPrint(expression.Value) {
			return true
		}
		for _, arm := range expression.Arms {
			if blockUsesPrint(arm.Body) {
				return true
			}
		}
	case *ast.IndexExpr:
		return exprUsesPrint(expression.Object) || exprUsesPrint(expression.Index)
	case *ast.CallExpr:
		if ident, ok := expression.Callee.(*ast.IdentExpr); ok && ident.Name == "imprimir" {
			return true
		}
		if exprUsesPrint(expression.Callee) {
			return true
		}
		for _, arg := range expression.Args {
			if exprUsesPrint(arg) {
				return true
			}
		}
	case *ast.UnaryExpr:
		return exprUsesPrint(expression.Value)
	case *ast.BinaryExpr:
		return exprUsesPrint(expression.Left) || exprUsesPrint(expression.Right)
	case *ast.IfExpr:
		for _, branch := range expression.Branches {
			if exprUsesPrint(branch.Condition) || exprUsesPrint(branch.Value) {
				return true
			}
		}
		return exprUsesPrint(expression.Else)
	case *ast.MemberExpr:
		return exprUsesPrint(expression.Object)
	case *ast.StructLiteralExpr:
		for _, field := range expression.Fields {
			if exprUsesPrint(field.Value) {
				return true
			}
		}
		for _, value := range expression.Values {
			if exprUsesPrint(value) {
				return true
			}
		}
	case *ast.ListLiteralExpr:
		for _, element := range expression.Elements {
			if exprUsesPrint(element) {
				return true
			}
		}
	case *ast.MapLiteralExpr:
		for _, entry := range expression.Entries {
			if exprUsesPrint(entry.Key) || exprUsesPrint(entry.Value) {
				return true
			}
		}
	}
	return false
}

func validate(filename string, source []byte) error {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, filename+".go", source, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("el Go generado no es válido: %w", err)
	}
	var typeErrors []string
	config := types.Config{
		Importer: importer.Default(),
		Error:    func(err error) { typeErrors = append(typeErrors, err.Error()) },
	}
	if _, err = config.Check("main", set, []*goast.File{file}, nil); err != nil {
		return fmt.Errorf("el Go generado no pasa la verificación de tipos: %s", strings.Join(typeErrors, "; "))
	}
	return nil
}

func (g *generator) write(format string, args ...any) {
	fmt.Fprintf(&g.buffer, format, args...)
}

func (g *generator) line(indent int, format string, args ...any) {
	g.buffer.WriteString(strings.Repeat("\t", indent))
	fmt.Fprintf(&g.buffer, format, args...)
	g.buffer.WriteByte('\n')
}
