package lspserver

import (
	"context"
	"strings"

	"github.com/owenrumney/go-lsp/lsp"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/sema"
	"github.com/crsolver/cometa/internal/token"
)

// Hover reports the compiler-inferred type of variables and the declaration
// of functions. A function's immediately preceding // comment block is shown
// as documentation, matching the convention used by Go tooling.
func (h *Handler) Hover(_ context.Context, params *lsp.HoverParams) (*lsp.Hover, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	return h.moduleHover(params, text), nil
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

func resolveHover(program *ast.Program, model *sema.Model, position ast.Pos, lines []string) (hoverInfo, bool) {
	if info, ok := stdlibHover(program, model, position); ok {
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
			} else if exists && iterableType.Kind == sema.String {
				if stmt.ElementPos == position {
					return variableHover(stmt.Element, sema.Type{Kind: sema.String}), true
				}
				if stmt.IndexPos == position {
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
