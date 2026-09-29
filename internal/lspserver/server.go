package lspserver

import (
	"context"
	"strings"
	"sync"
	"unicode"

	"github.com/owenrumney/go-lsp/document"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/server"

	"cometa/internal/ast"
	"cometa/internal/compiler"
	"cometa/internal/lexer"
	"cometa/internal/parser"
	"cometa/internal/sema"
	"cometa/internal/token"
)

const serverVersion = "0.1.0"

type Handler struct {
	projectMu sync.Mutex
	projects  map[lsp.DocumentURI]*compiler.Project
	reports   map[lsp.DocumentURI]map[lsp.DocumentURI][]lsp.Diagnostic
	published map[lsp.DocumentURI]bool
	documents *document.Store
	client    *server.Client
}

func NewHandler() *Handler {
	return &Handler{documents: document.NewStore()}
}

func Run(ctx context.Context) error {
	srv := server.NewServer(NewHandler())
	return srv.Run(ctx, server.RunStdio())
}

func (h *Handler) SetClient(client *server.Client) {
	h.client = client
}

func (h *Handler) Initialize(_ context.Context, _ *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	return &lsp.InitializeResult{
		ServerInfo: &lsp.ServerInfo{Name: "cometa", Version: serverVersion},
		Capabilities: lsp.ServerCapabilities{
			CompletionProvider: &lsp.CompletionOptions{TriggerCharacters: []string{".", "@"}},
		},
	}, nil
}

func (h *Handler) Shutdown(_ context.Context) error { return nil }

func (h *Handler) DidOpen(ctx context.Context, params *lsp.DidOpenTextDocumentParams) error {
	if params.TextDocument.LanguageID != "" && params.TextDocument.LanguageID != "cometa" {
		return nil
	}
	if _, err := h.documents.Open(params); err != nil {
		return err
	}
	return h.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (h *Handler) DidChange(ctx context.Context, params *lsp.DidChangeTextDocumentParams) error {
	if _, err := h.documents.Change(params); err != nil {
		return err
	}
	return h.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (h *Handler) DidSave(ctx context.Context, params *lsp.DidSaveTextDocumentParams) error {
	return h.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (h *Handler) DidClose(ctx context.Context, params *lsp.DidCloseTextDocumentParams) error {
	h.documents.Close(params)
	if h.client == nil {
		return nil
	}
	return h.publishProjectDiagnostics(ctx, params.TextDocument.URI)
}

// Completion offers fields and methods after a dot or the receiver marker @.
// The current line is made syntactically complete before analysis so
// completion still works while a member name is incomplete.
func (h *Handler) Completion(_ context.Context, params *lsp.CompletionParams) (*lsp.CompletionList, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return &lsp.CompletionList{}, nil
	}
	gameLines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if params.Position.Line >= 0 && params.Position.Line < len(gameLines) {
		if list := gameNamespaceCompletion(utf16Prefix(gameLines[params.Position.Line], params.Position.Character)); list != nil {
			return list, nil
		}
	}
	if hasImports(text) {
		return h.moduleCompletion(params, text), nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lineIndex := params.Position.Line
	if lineIndex < 0 || lineIndex >= len(lines) {
		return &lsp.CompletionList{}, nil
	}
	prefix := utf16Prefix(lines[lineIndex], params.Position.Character)
	if !completionCodePosition(prefix) {
		return &lsp.CompletionList{}, nil
	}
	dot := strings.LastIndex(prefix, ".")
	at := strings.LastIndex(prefix, "@")
	if at > dot {
		return receiverCompletion(string(params.TextDocument.URI), lines, lineIndex, prefix, at), nil
	}
	if dot < 0 {
		return gameTopCompletion(prefix), nil
	}
	if dot > 0 && prefix[dot-1] == '>' {
		return genericEnumCompletion(string(params.TextDocument.URI), lines, lineIndex, prefix, dot), nil
	}
	end := dot
	start := memberReceiverStart(prefix, end)
	receiverName := prefix[start:end]
	if list, handled := contextualCompletion(string(params.TextDocument.URI), lines, lineIndex, prefix, dot, start); handled {
		return list, nil
	}
	if receiverName == "" {
		return &lsp.CompletionList{}, nil
	}
	indentEnd := 0
	for indentEnd < len(lines[lineIndex]) && lines[lineIndex][indentEnd] == '\t' {
		indentEnd++
	}
	replacement := lines[lineIndex][:indentEnd]
	if arrow := strings.Index(prefix, "=>"); arrow >= 0 {
		replacement = prefix[:arrow+2] + " "
	}
	const receiverProbe = "__cometa_completion_receiver__"
	replacement += "var " + receiverProbe + " = " + receiverName
	model := completionModel(string(params.TextDocument.URI), lines, lineIndex, replacement)
	if model == nil {
		return &lsp.CompletionList{}, nil
	}
	var receiverType sema.Type
	for declaration, declarationType := range model.VarTypes {
		if declaration.Name == receiverProbe && declaration.Pos.Line == lineIndex+1 {
			receiverType = declarationType
			break
		}
	}
	if receiverType.Kind == sema.Interface || receiverType.Kind == sema.TypeParameter {
		return resolvedMemberItems(model, receiverType), nil
	}
	if receiverType.Kind != sema.Named && receiverType.Kind != sema.Slice && receiverType.Kind != sema.Map && receiverType.Kind != sema.String {
		if receiverType.Kind == sema.Invalid {
			if info := model.Enums[receiverName]; info != nil {
				return enumCompletionItems(info), nil
			}
		}
		return &lsp.CompletionList{}, nil
	}
	return resolvedMemberItems(model, receiverType), nil
}

func memberReceiverStart(prefix string, end int) int {
	start := end
	for start > 0 {
		for start > 0 && isIdentifierRune(rune(prefix[start-1])) {
			start--
		}
		if start > 0 && (prefix[start-1] == ']' || prefix[start-1] == ')' || prefix[start-1] == '}') {
			closing := prefix[start-1]
			opening := map[byte]byte{']': '[', ')': '(', '}': '{'}[closing]
			depth := 1
			start--
			for start > 0 && depth > 0 {
				start--
				switch prefix[start] {
				case closing:
					depth++
				case opening:
					depth--
				}
			}
			if closing == '}' {
				for start > 0 && prefix[start-1] == ' ' {
					start--
				}
			}
			continue
		}
		if start > 0 && prefix[start-1] == '"' {
			start--
			for start > 0 {
				start--
				if prefix[start] == '"' {
					backslashes := 0
					for i := start - 1; i >= 0 && prefix[i] == '\\'; i-- {
						backslashes++
					}
					if backslashes%2 == 0 {
						break
					}
				}
			}
			continue
		}
		if start > 0 && prefix[start-1] == '.' {
			start--
			continue
		}
		if start > 0 && prefix[start-1] == '@' {
			start--
		}
		break
	}
	return start
}

func receiverCompletion(filename string, lines []string, lineIndex int, prefix string, at int) *lsp.CompletionList {
	for _, value := range prefix[at+1:] {
		if !isIdentifierRune(value) {
			return &lsp.CompletionList{}
		}
	}
	indentEnd := 0
	for indentEnd < len(lines[lineIndex]) && lines[lineIndex][indentEnd] == '\t' {
		indentEnd++
	}
	if indentEnd == 0 {
		return &lsp.CompletionList{}
	}

	analysisLines := append([]string(nil), lines...)
	if indentEnd == 1 && strings.HasPrefix(strings.TrimPrefix(strings.TrimSpace(lines[lineIndex]), "pub "), "fn ") {
		// @ may be typed as the inline body of a method declaration.
		analysisLines[lineIndex] = prefix[:at] + "imprimir(verdadero)"
	} else if indentEnd >= 2 {
		analysisLines[lineIndex] = strings.Repeat("\t", indentEnd) + "imprimir(verdadero)"
	} else {
		return &lsp.CompletionList{}
	}
	tokens, _ := lexer.Lex(filename, strings.Join(analysisLines, "\n"))
	program, _ := parser.Parse(filename, tokens)
	typeDecl := enclosingTypeAt(program, lineIndex+1)
	if typeDecl == nil {
		return &lsp.CompletionList{}
	}
	if model, _ := sema.CheckForTooling(filename, program); model != nil {
		return resolvedMemberItems(model, sema.Type{Kind: sema.Named, Name: typeDecl.Name, Args: model.TypeParams[typeDecl]})
	}
	return memberCompletionItems(typeDecl)
}

func enclosingTypeAt(program *ast.Program, line int) *ast.TypeDecl {
	var enclosing ast.Decl
	for _, declaration := range program.Decls {
		if declaration.Position().Line > line {
			break
		}
		enclosing = declaration
	}
	typeDecl, _ := enclosing.(*ast.TypeDecl)
	return typeDecl
}

func memberCompletionItems(typeDecl *ast.TypeDecl) *lsp.CompletionList {
	items := make([]lsp.CompletionItem, 0, len(typeDecl.Fields)+len(typeDecl.Methods))
	fieldKind := lsp.CompletionItemKindField
	for _, field := range typeDecl.Fields {
		items = append(items, lsp.CompletionItem{Label: field.Name, Kind: &fieldKind, Detail: typeRefString(field.Type)})
	}
	methodKind := lsp.CompletionItemKindMethod
	for _, method := range typeDecl.Methods {
		detail := functionDetail(method)
		label := completionLabel(method.Name, len(method.Params))
		items = append(items, lsp.CompletionItem{Label: label, Kind: &methodKind, Documentation: completionDocumentation(detail, ""), InsertText: method.Name})
	}
	return &lsp.CompletionList{Items: items}
}

// Hover reports the compiler-inferred type of variables and the declaration
// of functions. A function's immediately preceding // comment block is shown
// as documentation, matching the convention used by Go tooling.
func (h *Handler) Hover(_ context.Context, params *lsp.HoverParams) (*lsp.Hover, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	if hasImports(text) {
		return h.moduleHover(params, text), nil
	}
	filename := analysisFilename(params.TextDocument.URI)
	program, model, _ := compiler.Analyze(filename, []byte(text))
	if program == nil || model == nil {
		return nil, nil
	}
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	tokens, _ := lexer.Lex(string(params.TextDocument.URI), normalized)
	identifier, hoverRange, ok := identifierAt(tokens, lines, params.Position)
	if !ok {
		return nil, nil
	}
	info, ok := resolveHover(program, model, identifier.Pos, lines)
	if !ok {
		return nil, nil
	}
	value := "```cometa\n" + wrapSignature(info.detail) + "\n```"
	if info.documentation != "" {
		value += "\n\n" + info.documentation
	}
	return &lsp.Hover{Contents: lsp.NewHoverContents(lsp.Markdown, value), Range: &hoverRange}, nil
}

type hoverInfo struct {
	detail        string
	documentation string
}

func identifierAt(tokens []token.Token, lines []string, position lsp.Position) (token.Token, lsp.Range, bool) {
	for _, candidate := range tokens {
		if candidate.Kind != token.Ident || candidate.Pos.Line-1 != position.Line {
			continue
		}
		lineIndex := candidate.Pos.Line - 1
		if lineIndex < 0 || lineIndex >= len(lines) {
			continue
		}
		lineRunes := []rune(lines[lineIndex])
		startRune := candidate.Pos.Column - 1
		endRune := startRune + len([]rune(candidate.Lexeme))
		if startRune < 0 || endRune > len(lineRunes) {
			continue
		}
		start := utf16Length(string(lineRunes[:startRune]))
		end := utf16Length(string(lineRunes[:endRune]))
		if position.Character < start || position.Character >= end {
			continue
		}
		hoverRange := lsp.Range{
			Start: lsp.Position{Line: lineIndex, Character: start},
			End:   lsp.Position{Line: lineIndex, Character: end},
		}
		return candidate, hoverRange, true
	}
	// Interpolation is represented by one outer string token. Recover the word
	// under the cursor here; AST resolution below confirms that it is code.
	if position.Line >= 0 && position.Line < len(lines) {
		runes := []rune(lines[position.Line])
		cursor := runeIndexAtUTF16(lines[position.Line], position.Character)
		start, end := cursor, cursor
		if start == len(runes) || start < len(runes) && !isIdentifierRune(runes[start]) {
			start--
		}
		for start >= 0 && isIdentifierRune(runes[start]) {
			start--
		}
		start++
		for end < len(runes) && isIdentifierRune(runes[end]) {
			end++
		}
		if start < end {
			lexeme := string(runes[start:end])
			return token.Token{Kind: token.Ident, Lexeme: lexeme, Pos: ast.Pos{Line: position.Line + 1, Column: start + 1}}, lsp.Range{Start: lsp.Position{Line: position.Line, Character: utf16Length(string(runes[:start]))}, End: lsp.Position{Line: position.Line, Character: utf16Length(string(runes[:end]))}}, true
		}
	}
	return token.Token{}, lsp.Range{}, false
}

func runeIndexAtUTF16(line string, column int) int {
	units, index := 0, 0
	for _, r := range line {
		width := 1
		if r > 0xffff {
			width = 2
		}
		if units+width > column {
			break
		}
		units += width
		index++
	}
	return index
}

func resolveHover(program *ast.Program, model *sema.Model, position ast.Pos, lines []string) (hoverInfo, bool) {
	if info, ok := gameHover(program, model, position); ok {
		return info, true
	}
	if info, ok := enumHover(model, position); ok {
		return info, true
	}
	// Identifier expressions are the common case and already carry their exact
	// inferred type in the semantic model.
	for expression, expressionType := range model.ExprTypes {
		if identifier, ok := expression.(*ast.IdentExpr); ok && identifier.Pos == position {
			if raw, ok := model.RawTypes[expression]; ok {
				expressionType = raw
			}
			if global := model.GlobalRefs[identifier]; global != nil {
				return globalHover(identifier.Name, expressionType, global.Constant), true
			}
			return variableHover(identifier.Name, expressionType), true
		}
	}

	for _, declaration := range program.Decls {
		switch decl := declaration.(type) {
		case *ast.InterfaceDecl:
			if decl.NamePos == position {
				return hoverInfo{detail: publicDetail(decl.Public, "interfaz "+decl.Name+typeParamDetail(decl.TypeParams))}, true
			}
			for _, method := range decl.Methods {
				if info, ok := hoverInFunction(method, model.Interfaces[decl.Name].Methods[method.Name], model, position, lines); ok {
					return info, true
				}
			}
		case *ast.TypeDecl:
			if decl.NamePos == position {
				return hoverInfo{detail: publicDetail(decl.Public, "tipo "+decl.Name+typeParamDetail(decl.TypeParams))}, true
			}
			for _, method := range decl.Methods {
				if info, ok := hoverInFunction(method, model.Types[decl.Name].Methods[method.Name], model, position, lines); ok {
					return info, true
				}
			}
		case *ast.FuncDecl:
			if info, ok := hoverInFunction(decl, model.Functions[decl.Name], model, position, lines); ok {
				return info, true
			}
		case *ast.GlobalDecl:
			if decl.NamePos == position {
				global := model.Globals[decl.Name]
				info := globalHover(decl.Name, global.Type, decl.Constant)
				info.detail = publicDetail(decl.Public, info.detail)
				return info, true
			}
			if info, ok := hoverInExpression(decl.Value, model, position, lines, ""); ok {
				return info, true
			}
		}
	}
	return hoverInfo{}, false
}

func hoverInFunction(function *ast.FuncDecl, signature sema.FuncInfo, model *sema.Model, position ast.Pos, lines []string) (hoverInfo, bool) {
	if function.NamePos == position {
		return functionHover(signature, lines), true
	}
	for index, param := range function.Params {
		if param.Pos == position {
			return variableHover(param.Name, signature.Params[index]), true
		}
		if param.Default != nil {
			if info, ok := hoverInExpression(param.Default, model, position, lines, function.Receiver); ok {
				return info, true
			}
		}
	}
	return hoverInStatements(function.Body, model, position, lines, function.Receiver)
}

func hoverInStatements(statements []ast.Stmt, model *sema.Model, position ast.Pos, lines []string, receiverName string) (hoverInfo, bool) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.MatchStmt:
			if info, ok := hoverInExpression(stmt.Match, model, position, lines, receiverName); ok {
				return info, true
			}
		case *ast.ExprStmt:
			if info, ok := hoverInExpression(stmt.Expr, model, position, lines, receiverName); ok {
				return info, true
			}
		case *ast.AssignStmt:
			if info, ok := hoverInExpression(stmt.Target, model, position, lines, receiverName); ok {
				return info, true
			}
			if info, ok := hoverInExpression(stmt.Value, model, position, lines, receiverName); ok {
				return info, true
			}
		case *ast.VarDeclStmt:
			if stmt.NamePos == position {
				if variableType, exists := model.VarTypes[stmt]; exists {
					return variableHover(stmt.Name, variableType), true
				}
			}
			if info, ok := hoverInExpression(stmt.Value, model, position, lines, receiverName); ok {
				return info, true
			}
		case *ast.IfStmt:
			for _, branch := range stmt.Branches {
				if branch.Binding != "" && branch.BindingPos == position {
					if typ := model.ExprTypes[branch.Condition]; typ.Kind == sema.Optional {
						return variableHover(branch.Binding, *typ.Elem), true
					}
				}
				if info, ok := hoverInExpression(branch.Condition, model, position, lines, receiverName); ok {
					return info, true
				}
				if info, ok := hoverInStatements(branch.Body, model, position, lines, receiverName); ok {
					return info, true
				}
			}
			if info, ok := hoverInStatements(stmt.Else, model, position, lines, receiverName); ok {
				return info, true
			}
		case *ast.ScopeStmt:
			if info, ok := hoverInExpression(stmt.Value, model, position, lines, receiverName); ok {
				return info, true
			}
			if info, ok := hoverInStatements(stmt.Body, model, position, lines, receiverName); ok {
				return info, true
			}
		case *ast.RepeatStmt:
			if info, ok := hoverInExpression(stmt.RangeEnd, model, position, lines, receiverName); ok {
				return info, true
			}
			if stmt.RangeEnd != nil && (stmt.ElementPos == position || stmt.IndexPos == position) {
				name := stmt.Element
				if stmt.IndexPos == position {
					name = stmt.Index
				}
				return variableHover(name, sema.Type{Kind: sema.Integer}), true
			}
			if info, ok := hoverInExpression(stmt.Iterable, model, position, lines, receiverName); ok {
				return info, true
			}
			if iterableType, exists := model.ExprTypes[stmt.Iterable]; exists && (iterableType.Kind == sema.Slice || iterableType.Kind == sema.Map) {
				if stmt.ElementPos == position {
					return variableHover(stmt.Element, *iterableType.Elem), true
				}
				if stmt.IndexPos == position {
					if iterableType.Kind == sema.Map {
						return variableHover(stmt.Index, *iterableType.Key), true
					}
					return variableHover(stmt.Index, sema.Type{Kind: sema.Integer}), true
				}
			}
			if info, ok := hoverInStatements(stmt.Body, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	}
	return hoverInfo{}, false
}

func hoverInExpression(expression ast.Expr, model *sema.Model, position ast.Pos, lines []string, receiverName string) (hoverInfo, bool) {
	if expression == nil {
		return hoverInfo{}, false
	}
	if identifier, ok := expression.(*ast.IdentExpr); ok && identifier.Pos == position {
		if variableType, exists := model.ExprTypes[identifier]; exists {
			if wrapper, converted := model.Wraps[identifier]; converted {
				variableType = *wrapper.Elem
			}
			return variableHover(identifier.Name, variableType), true
		}
	}
	switch expr := expression.(type) {
	case *ast.AssertExpr:
		if expr.Target.Pos == position {
			return hoverInfo{detail: typeRefString(expr.Target)}, true
		}
		return hoverInExpression(expr.Value, model, position, lines, receiverName)
	case *ast.ReturnExpr:
		return hoverInExpression(expr.Value, model, position, lines, receiverName)
	case *ast.TryExpr:
		return hoverInExpression(expr.Value, model, position, lines, receiverName)
	case *ast.BlockExpr:
		return hoverInStatements(expr.Body, model, position, lines, receiverName)
	case *ast.RecoverExpr:
		if info, ok := hoverInExpression(expr.Value, model, position, lines, receiverName); ok {
			return info, true
		}
		if expr.Binding != "" && expr.BindingPos == position {
			if typ := model.ExprTypes[expr.Value]; typ.Kind == sema.Result {
				return variableHover(expr.Binding, *typ.Err), true
			}
		}
		return hoverInStatements(expr.Body, model, position, lines, receiverName)
	case *ast.MatchExpr:
		if info, ok := hoverInExpression(expr.Value, model, position, lines, receiverName); ok {
			return info, true
		}
		for _, arm := range expr.Arms {
			if arm.Qualifier != "" && arm.QualifierPos == position {
				return hoverInfo{detail: "enum " + arm.Qualifier}, true
			}
			if arm.NamePos == position && arm.TypePattern != nil {
				return hoverInfo{detail: model.PatternTypes[arm].String()}, true
			}
			if arm.NamePos == position && arm.Pattern != "_" {
				if enum := model.EnumFor(model.ExprTypes[expr.Value]); enum != nil {
					return variantHover(enum, enum.Variants[arm.Pattern]), true
				}
			}
			if info, ok := hoverInStatements(arm.Body, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	case *ast.UnaryExpr:
		return hoverInExpression(expr.Value, model, position, lines, receiverName)
	case *ast.BinaryExpr:
		if info, ok := hoverInExpression(expr.Left, model, position, lines, receiverName); ok {
			return info, true
		}
		return hoverInExpression(expr.Right, model, position, lines, receiverName)
	case *ast.InterpolatedStringExpr:
		for _, part := range expr.Parts {
			if info, ok := hoverInExpression(part.Expr, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	case *ast.CallExpr:
		callPosition := ast.Pos{}
		switch callee := expr.Callee.(type) {
		case *ast.IdentExpr:
			callPosition = callee.Pos
		case *ast.InstantiateExpr:
			callPosition = callee.Pos
		case *ast.MemberExpr:
			callPosition = callee.NamePos
		case *ast.ReceiverExpr:
			callPosition = callee.NamePos
		}
		if callPosition == position {
			if call, ok := model.Calls[expr]; ok {
				info := functionHover(call.Signature, lines)
				if operation, builtin := model.ListCalls[expr]; builtin {
					info.documentation = sema.ListMethodDocumentation(operation)
				}
				if operation, builtin := model.MapCalls[expr]; builtin {
					info.documentation = sema.MapMethodDocumentation(operation)
				}
				if operation, builtin := model.StringCalls[expr]; builtin {
					info.documentation = sema.StringMethodDocumentation(operation)
				}
				if operation, builtin := model.NumberCalls[expr]; builtin {
					info.documentation = sema.NumberMethodDocumentation(operation)
				}
				return info, true
			}
		}
		switch callee := expr.Callee.(type) {
		case *ast.IdentExpr:
			if callee.Pos == position {
				if function, exists := model.Functions[callee.Name]; exists {
					return functionHover(function, lines), true
				}
			}
		case *ast.MemberExpr:
			if callee.NamePos == position {
				if receiverType, exists := model.ExprTypes[callee.Object]; exists && receiverType.Kind == sema.Named {
					member := model.Members(receiverType)[callee.Name]
					if !member.Ambiguous && member.Accessible(position) && member.Method.Decl != nil {
						return functionHover(member.Method, lines), true
					}
				}
			}
		case *ast.ReceiverExpr:
			if callee.NamePos == position && receiverName != "" {
				if typeInfo, exists := model.Types[receiverName]; exists {
					if method, exists := typeInfo.Methods[callee.Name]; exists {
						return functionHover(method, lines), true
					}
				}
			}
		}
		if info, ok := hoverInExpression(expr.Callee, model, position, lines, receiverName); ok {
			return info, true
		}
		for _, argument := range expr.Args {
			if info, ok := hoverInExpression(argument, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	case *ast.MemberExpr:
		if expr.NamePos == position {
			if typ, ok := model.ExprTypes[expr]; ok {
				return variableHover(expr.Name, typ), true
			}
		}
		return hoverInExpression(expr.Object, model, position, lines, receiverName)
	case *ast.ReceiverExpr:
		if expr.NamePos == position {
			if typ, ok := model.ExprTypes[expr]; ok {
				return variableHover(expr.Name, typ), true
			}
		}
	case *ast.IndexExpr:
		if info, ok := hoverInExpression(expr.Object, model, position, lines, receiverName); ok {
			return info, true
		}
		return hoverInExpression(expr.Index, model, position, lines, receiverName)
	case *ast.StructLiteralExpr:
		for _, field := range expr.Fields {
			if info, ok := hoverInExpression(field.Value, model, position, lines, receiverName); ok {
				return info, true
			}
		}
		for _, value := range expr.Values {
			if info, ok := hoverInExpression(value, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	case *ast.ListLiteralExpr:
		for _, element := range expr.Elements {
			if info, ok := hoverInExpression(element, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	case *ast.MapLiteralExpr:
		for _, entry := range expr.Entries {
			if info, ok := hoverInExpression(entry.Key, model, position, lines, receiverName); ok {
				return info, true
			}
			if info, ok := hoverInExpression(entry.Value, model, position, lines, receiverName); ok {
				return info, true
			}
		}
	case *ast.IfExpr:
		for _, branch := range expr.Branches {
			if branch.Binding != "" && branch.BindingPos == position {
				if typ := model.ExprTypes[branch.Condition]; typ.Kind == sema.Optional {
					return variableHover(branch.Binding, *typ.Elem), true
				}
			}
			if info, ok := hoverInExpression(branch.Condition, model, position, lines, receiverName); ok {
				return info, true
			}
			if info, ok := hoverInExpression(branch.Value, model, position, lines, receiverName); ok {
				return info, true
			}
		}
		return hoverInExpression(expr.Else, model, position, lines, receiverName)
	}
	return hoverInfo{}, false
}

func variableHover(name string, variableType sema.Type) hoverInfo {
	return hoverInfo{detail: "var " + name + " " + variableType.String()}
}

func globalHover(name string, variableType sema.Type, constant bool) hoverInfo {
	keyword := "var"
	if constant {
		keyword = "const"
	}
	return hoverInfo{detail: keyword + " " + name + " " + variableType.String()}
}

func functionHover(function sema.FuncInfo, lines []string) hoverInfo {
	return hoverInfo{
		detail:        signatureDetail(function),
		documentation: documentationBefore(lines, function.Decl.NamePos.Line),
	}
}

func documentationBefore(lines []string, declarationLine int) string {
	lineIndex := declarationLine - 1
	if lineIndex <= 0 || lineIndex >= len(lines) {
		return ""
	}
	declaration := lines[lineIndex]
	indentLength := 0
	for indentLength < len(declaration) && declaration[indentLength] == '\t' {
		indentLength++
	}
	indent := declaration[:indentLength]
	var reversed []string
	for index := lineIndex - 1; index >= 0; index-- {
		raw := lines[index]
		if !strings.HasPrefix(raw, indent+"//") {
			break
		}
		comment := strings.TrimPrefix(raw[len(indent):], "//")
		comment = strings.TrimPrefix(comment, " ")
		reversed = append(reversed, comment)
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return strings.Join(reversed, "\n")
}

func isIdentifierRune(value rune) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value > 127
}

func utf16Prefix(value string, units int) string {
	if units <= 0 {
		return ""
	}
	used := 0
	for byteIndex, r := range value {
		width := 1
		if r > 0xFFFF {
			width = 2
		}
		if used+width > units {
			return value[:byteIndex]
		}
		used += width
		if used == units {
			return value[:byteIndex+len(string(r))]
		}
	}
	return value
}

func (h *Handler) publishDiagnostics(ctx context.Context, uri lsp.DocumentURI) error {
	return h.publishProjectDiagnostics(ctx, uri)
}

func diagnosticFromError(err error, text string) lsp.Diagnostic {
	position := ast.Pos{Line: 1, Column: 1}
	message := err.Error()
	if located, ok := err.(interface {
		Diagnostic() (string, ast.Pos, string, string)
	}); ok {
		_, position, _, message = located.Diagnostic()
	}
	switch typed := err.(type) {
	case *lexer.Error:
		position, message = typed.Pos, typed.Message
	case *parser.Error:
		position, message = typed.Pos, typed.Message
	case *sema.Error:
		position, message = typed.Pos, typed.Message
	}
	start, end := diagnosticRange(position, text)
	severity := lsp.SeverityError
	return lsp.Diagnostic{
		Range:    lsp.Range{Start: start, End: end},
		Severity: &severity,
		Source:   "cometa",
		Message:  message,
	}
}

func diagnosticRange(position ast.Pos, text string) (lsp.Position, lsp.Position) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	lineIndex := max(position.Line-1, 0)
	runeStart := max(position.Column-1, 0)
	if lineIndex >= len(lines) {
		point := lsp.Position{Line: lineIndex, Character: runeStart}
		return point, point
	}

	lineRunes := []rune(lines[lineIndex])
	runeStart = min(runeStart, len(lineRunes))
	runeLength := tokenRuneLengthAt(position, normalized)
	if runeLength == 0 {
		runeLength = fallbackTokenRuneLength(lineRunes, runeStart)
	}
	runeEnd := min(runeStart+runeLength, len(lineRunes))
	start := lsp.Position{Line: lineIndex, Character: utf16Length(string(lineRunes[:runeStart]))}
	end := lsp.Position{Line: lineIndex, Character: utf16Length(string(lineRunes[:runeEnd]))}
	return start, end
}

// tokenRuneLengthAt uses the compiler's lexer as the source of truth whenever
// lexing succeeds. This keeps diagnostic ranges aligned with Cometa tokens,
// including quoted strings and Unicode identifiers.
func tokenRuneLengthAt(position ast.Pos, source string) int {
	tokens, err := lexer.Lex("", source)
	if err != nil {
		return 0
	}
	for _, candidate := range tokens {
		if candidate.Pos == position && candidate.Lexeme != "" {
			return len([]rune(candidate.Lexeme))
		}
	}
	return 0
}

// fallbackTokenRuneLength handles diagnostics produced before a complete token
// stream exists, most notably lexer errors.
func fallbackTokenRuneLength(line []rune, start int) int {
	if start >= len(line) {
		return 0
	}
	if line[start] == ' ' || line[start] == '\t' {
		end := start + 1
		for end < len(line) && line[end] == line[start] {
			end++
		}
		return end - start
	}
	if unicode.IsLetter(line[start]) || line[start] == '_' {
		end := start + 1
		for end < len(line) && (unicode.IsLetter(line[end]) || unicode.IsDigit(line[end]) || line[end] == '_') {
			end++
		}
		return end - start
	}
	if unicode.IsDigit(line[start]) {
		end := start + 1
		for end < len(line) && (unicode.IsDigit(line[end]) || line[end] == '.') {
			end++
		}
		return end - start
	}
	if line[start] == '"' {
		escaped := false
		for end := start + 1; end < len(line); end++ {
			if line[end] == '"' && !escaped {
				return end - start + 1
			}
			if line[end] == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		}
		return len(line) - start
	}
	if start+1 < len(line) {
		switch string(line[start : start+2]) {
		case "==", "!=", "<=", ">=", "&&", "||":
			return 2
		}
	}
	return 1
}

func (h *Handler) DocumentSymbol(_ context.Context, params *lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	program, model, _ := compiler.Analyze(analysisFilename(params.TextDocument.URI), []byte(text))
	if program == nil {
		return nil, nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var symbols []lsp.DocumentSymbol
	for _, decl := range program.Decls {
		switch declaration := decl.(type) {
		case *ast.InterfaceDecl:
			symbol := symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindInterface, "interfaz"+typeParamDetail(declaration.TypeParams))
			for _, method := range declaration.Methods {
				symbol.Children = append(symbol.Children, symbolFor(lines, method.Name, method.Pos, lsp.SymbolKindMethod, functionDetail(method)))
			}
			symbols = append(symbols, symbol)
		case *ast.EnumDecl:
			symbol := symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindEnum, "enum"+typeParamDetail(declaration.TypeParams))
			for _, variant := range declaration.Variants {
				detail := "variante"
				if variant.Payload != nil {
					detail = typeRefString(*variant.Payload)
				}
				symbol.Children = append(symbol.Children, symbolFor(lines, variant.Name, variant.Pos, lsp.SymbolKindEnumMember, detail))
			}
			symbols = append(symbols, symbol)
		case *ast.TypeDecl:
			symbol := symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindStruct, "tipo"+typeParamDetail(declaration.TypeParams))
			for _, field := range declaration.Fields {
				symbol.Children = append(symbol.Children, symbolFor(lines, field.Name, field.Pos, lsp.SymbolKindField, typeRefString(field.Type)))
			}
			for _, method := range declaration.Methods {
				symbol.Children = append(symbol.Children, symbolFor(lines, method.Name, method.Pos, lsp.SymbolKindMethod, functionDetail(method)))
			}
			symbols = append(symbols, symbol)
		case *ast.FuncDecl:
			symbols = append(symbols, symbolFor(lines, declaration.Name, declaration.Pos, lsp.SymbolKindFunction, functionDetail(declaration)))
		case *ast.GlobalDecl:
			kind := lsp.SymbolKindVariable
			detail := "var"
			if declaration.Constant {
				kind = lsp.SymbolKindConstant
				detail = "const"
			}
			if model != nil && model.Globals[declaration.Name] != nil {
				detail += " " + model.Globals[declaration.Name].Type.String()
			}
			symbols = append(symbols, symbolFor(lines, declaration.Name, declaration.Pos, kind, detail))
		}
	}
	return symbols, nil
}

func symbolFor(lines []string, name string, position ast.Pos, kind lsp.SymbolKind, detail string) lsp.DocumentSymbol {
	lineIndex := max(position.Line-1, 0)
	lineLength := 0
	nameStart := max(position.Column-1, 0)
	if lineIndex < len(lines) {
		lineLength = utf16Length(lines[lineIndex])
		if byteIndex := strings.Index(lines[lineIndex], name); byteIndex >= 0 {
			nameStart = utf16Length(lines[lineIndex][:byteIndex])
		}
	}
	wholeLine := lsp.Range{
		Start: lsp.Position{Line: lineIndex, Character: 0},
		End:   lsp.Position{Line: lineIndex, Character: lineLength},
	}
	selection := lsp.Range{
		Start: lsp.Position{Line: lineIndex, Character: nameStart},
		End:   lsp.Position{Line: lineIndex, Character: nameStart + utf16Length(name)},
	}
	return lsp.DocumentSymbol{Name: name, Detail: detail, Kind: kind, Range: wholeLine, SelectionRange: selection}
}

func typeRefString(ref ast.TypeRef) string {
	if ref.Wrapper != "" {
		payload := typeRefString(*ref.Payload)
		if ref.Payload.Name == "$unidad" {
			payload = ""
		}
		if ref.Payload.Wrapper == "!" {
			payload = "(" + payload + ")"
		}
		if ref.Wrapper == "?" {
			return payload + "?"
		}
		if ref.ErrorType.Name == "cadena" && ref.ErrorType.Wrapper == "" {
			return payload + "!"
		}
		errorType := typeRefString(*ref.ErrorType)
		if ref.ErrorType.Wrapper != "" {
			errorType = "(" + errorType + ")"
		}
		return payload + "!" + errorType
	}
	if ref.Element != nil {
		if ref.Key != nil {
			return "[" + typeRefString(*ref.Key) + ": " + typeRefString(*ref.Element) + "]"
		}
		return "[" + typeRefString(*ref.Element) + "]"
	}
	name := ref.Name
	if len(ref.Args) > 0 {
		var args []string
		for _, arg := range ref.Args {
			args = append(args, typeRefString(arg))
		}
		name += "<" + strings.Join(args, ", ") + ">"
	}
	return name
}

func functionDetail(function *ast.FuncDecl) string {
	var params []string
	for _, param := range function.Params {
		prefix := ""
		if param.Variadic {
			prefix = "..."
		}
		detail := param.Name + " " + prefix + typeRefString(param.Type)
		if param.Default != nil {
			detail += " = " + defaultExprSource(param.Default)
		}
		params = append(params, detail)
	}
	detail := "fn " + function.Name + typeParamDetail(function.TypeParams) + "(" + strings.Join(params, ", ") + ")"
	if function.Public {
		detail = "pub " + detail
	}
	if function.ReturnType != nil {
		detail += " " + typeRefString(*function.ReturnType)
	}
	return detail
}

func utf16Length(value string) int {
	length := 0
	for _, r := range value {
		if r > 0xFFFF {
			length += 2
		} else {
			length++
		}
	}
	return length
}
