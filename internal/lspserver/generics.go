package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"cometa/internal/ast"
	"cometa/internal/sema"
	"sort"
	"strings"
)

func typeParamDetail(params []ast.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	var parts []string
	for _, p := range params {
		s := p.Name
		if p.Constraint != nil {
			s += " " + typeRefString(*p.Constraint)
		}
		parts = append(parts, s)
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

func publicDetail(public bool, detail string) string {
	if public {
		return "pub " + detail
	}
	return detail
}

func signatureDetail(f sema.FuncInfo) string {
	if f.Decl == nil {
		return ""
	}
	name := f.Decl.Name + typeParamDetail(f.Decl.TypeParams)
	if len(f.TypeArgs) > 0 {
		var args []string
		for _, t := range f.TypeArgs {
			args = append(args, t.String())
		}
		name = f.Decl.Name + "<" + strings.Join(args, ", ") + ">"
	}
	var params []string
	for i, p := range f.Decl.Params {
		t := f.Params[i].String()
		if p.Variadic {
			t = "..." + f.Params[i].Elem.String()
		}
		s := p.Name + " " + t
		if p.Default != nil {
			s += " = " + defaultExprSource(p.Default)
		}
		params = append(params, s)
	}
	detail := "fn " + name + "(" + strings.Join(params, ", ") + ")"
	if f.Decl.Public {
		detail = "pub " + detail
	}
	if f.Return.Kind != sema.Void {
		detail += " " + f.Return.String()
	}
	return detail
}

// defaultExprSource renders a parameter default expression as Cometa source
// text for hover display, falling back to "…" for expressions too complex
// to reproduce exactly (blocks, matches, try/recover).
func defaultExprSource(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return e.Value
	case *ast.IdentExpr:
		return e.Name
	case *ast.ReceiverExpr:
		return "@" + e.Name
	case *ast.ContextualVariantExpr:
		return "." + e.Name
	case *ast.UnaryExpr:
		if isIdentifierRune(rune(e.Operator[0])) {
			return e.Operator + " " + defaultExprSource(e.Value)
		}
		return e.Operator + defaultExprSource(e.Value)
	case *ast.BinaryExpr:
		return defaultExprSource(e.Left) + " " + e.Operator + " " + defaultExprSource(e.Right)
	case *ast.MemberExpr:
		return defaultExprSource(e.Object) + "." + e.Name
	case *ast.IndexExpr:
		return defaultExprSource(e.Object) + "[" + defaultExprSource(e.Index) + "]"
	case *ast.CallExpr:
		var args []string
		for i, arg := range e.Args {
			text := defaultExprSource(arg)
			if i < len(e.ArgInfo) && e.ArgInfo[i].Name != "" {
				text = e.ArgInfo[i].Name + ": " + text
			}
			args = append(args, text)
		}
		return defaultExprSource(e.Callee) + "(" + strings.Join(args, ", ") + ")"
	case *ast.ListLiteralExpr:
		var elements []string
		for _, element := range e.Elements {
			elements = append(elements, defaultExprSource(element))
		}
		return "[" + strings.Join(elements, ", ") + "]"
	case *ast.MapLiteralExpr:
		if len(e.Entries) == 0 {
			return "[:]"
		}
		var entries []string
		for _, entry := range e.Entries {
			entries = append(entries, defaultExprSource(entry.Key)+": "+defaultExprSource(entry.Value))
		}
		return "[" + strings.Join(entries, ", ") + "]"
	case *ast.InstantiateExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, typeRefString(arg))
		}
		return e.Name + "<" + strings.Join(args, ", ") + ">"
	case *ast.StructLiteralExpr:
		var fields []string
		for _, field := range e.Fields {
			fields = append(fields, field.Name+": "+defaultExprSource(field.Value))
		}
		return e.TypeName + "(" + strings.Join(fields, ", ") + ")"
	default:
		return "…"
	}
}

// completionLabel appends call parentheses to a function/method name so the
// suggestion list shows callability at a glance; full parameter details stay
// in Detail, which most clients already surface next to or above the
// documentation panel for the selected item.
func completionLabel(name string, paramCount int) string {
	if paramCount == 0 {
		return name + "()"
	}
	return name + "(...)"
}

// completionDocumentation renders a completion item's documentation panel:
// the signature as a syntax-highlighted ```cometa code block, followed by
// any doc comment. Function/method items leave Detail empty and put the
// signature here instead — carrying it in both fields would show it twice
// wherever a client renders Detail as this panel's own header.
func completionDocumentation(detail, documentation string) *lsp.MarkupContent {
	if detail == "" {
		return nil
	}
	value := "```cometa\n" + wrapSignature(detail) + "\n```"
	if documentation != "" {
		value += "\n\n" + documentation
	}
	return &lsp.MarkupContent{Kind: lsp.Markdown, Value: value}
}

// wrapSignature breaks a long single-line signature onto multiple lines, one
// parameter per line, so the documentation panel doesn't clip it — panels
// don't soft-wrap code blocks, so a long line just gets cut off or requires
// horizontal scrolling.
const signatureWrapWidth = 36

func wrapSignature(detail string) string {
	if len(detail) <= signatureWrapWidth {
		return detail
	}
	open := strings.Index(detail, "(")
	if open < 0 {
		return detail
	}
	depth := 0
	close := -1
	for i := open; i < len(detail); i++ {
		switch detail[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				close = i
				break
			}
		}
		if close >= 0 {
			break
		}
	}
	if close < 0 {
		return detail
	}
	inner := strings.TrimSpace(detail[open+1 : close])
	if inner == "" {
		return detail
	}
	params := splitTopLevel(detail[open+1 : close])
	var b strings.Builder
	b.WriteString(detail[:open+1])
	for _, param := range params {
		b.WriteString("\n\t")
		b.WriteString(strings.TrimSpace(param))
		b.WriteString(",")
	}
	b.WriteString("\n")
	b.WriteString(detail[close:])
	return b.String()
}

// splitTopLevel splits s on commas that aren't nested inside (), [] or <>,
// so parameter defaults and generic type arguments stay intact.
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(', '[', '<':
			depth++
		case ')', ']', '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func resolvedMemberItems(model *sema.Model, t sema.Type, positions ...ast.Pos) *lsp.CompletionList {
	pos := ast.Pos{}
	if len(positions) > 0 {
		pos = positions[0]
	}
	items := []lsp.CompletionItem{}
	if t.Kind == sema.Named {
		if info := model.StructInfo(t); info != nil {
			kind := lsp.CompletionItemKindField
			members := model.Members(t)
			seen := map[string]bool{}
			for _, f := range info.Decl.Fields {
				if !members[f.Name].Accessible(pos) {
					continue
				}
				items = append(items, lsp.CompletionItem{Label: f.Name, Kind: &kind, Detail: info.Fields[f.Name].Type.String()})
				seen[f.Name] = true
			}
			var names []string
			for name, member := range members {
				if !seen[name] && !member.Ambiguous && member.Field.Decl != nil && member.Accessible(pos) {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			for _, name := range names {
				items = append(items, lsp.CompletionItem{Label: name, Kind: &kind, Detail: members[name].Field.Type.String()})
			}
		}
	}
	methods := model.Methods(t)
	var names []string
	for name := range methods {
		if t.Kind == sema.Named && !model.Members(t)[name].Accessible(pos) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	kind := lsp.CompletionItemKindMethod
	for _, name := range names {
		detail := signatureDetail(methods[name])
		label := completionLabel(name, len(methods[name].Params))
		items = append(items, lsp.CompletionItem{Label: label, Kind: &kind, Documentation: completionDocumentation(detail, ""), InsertText: name})
	}
	return &lsp.CompletionList{Items: items}
}

func genericEnumCompletion(filename string, lines []string, line int, prefix string, dot int) *lsp.CompletionList {
	model := completionModel(filename, lines, line, closeCompletionDelimiters(prefix[:dot+1]+"__cometa_generic_probe__"))
	if model != nil {
		for expr, t := range model.ExprTypes {
			if _, ok := expr.(*ast.InstantiateExpr); ok && expr.Position().Line == line+1 && t.Kind == sema.Enum {
				return enumCompletionItems(model.EnumFor(t))
			}
		}
	}
	return &lsp.CompletionList{}
}
