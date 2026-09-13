package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"hacha/internal/ast"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"hacha/internal/sema"
	"strings"
)

// Partial checking retains payload bindings before coverage is complete.
// The full document fallback resolves declarations after the cursor too.
func completionModel(filename string, lines []string, line int, replacement string) *sema.Model {
	var best *sema.Model
	// Drop the unfinished remainder of the current declaration while retaining
	// subsequent declarations and their original source positions.
	tail := append([]string(nil), lines...)
	for i := line + 1; i < len(tail); i++ {
		if strings.TrimSpace(tail[i]) != "" && !strings.HasPrefix(tail[i], "\t") && !strings.HasPrefix(strings.TrimSpace(tail[i]), "//") {
			break
		}
		tail[i] = ""
	}
	for _, input := range [][]string{lines[:line+1], lines, tail} {
		copy := append([]string(nil), input...)
		copy[line] = replacement
		tokens, err := lexer.Lex(filename, strings.Join(copy, "\n")+"\n")
		if err != nil {
			continue
		}
		program, err := parser.Parse(filename, tokens)
		if err != nil {
			continue
		}
		model, _ := sema.CheckForTooling(filename, program)
		if best == nil || completionScore(model, line) > completionScore(best, line) {
			best = model
		}
	}
	return best
}

func completionScore(model *sema.Model, line int) int {
	score := len(model.ExprTypes) + len(model.Enums)
	for expr := range model.ExpectedTypes {
		if expr.Position().Line == line+1 {
			score += 1000
		}
	}
	for arm := range model.PatternTypes {
		if arm.Pos.Line == line+1 {
			score += 1000
		}
	}
	return score
}

func enumCompletionItems(info *sema.EnumInfo) *lsp.CompletionList {
	list := &lsp.CompletionList{}
	kind := lsp.CompletionItemKindEnumMember
	for _, decl := range info.Decl.Variants {
		variant, ok := info.Variants[decl.Name]
		if !ok {
			continue
		}
		list.Items = append(list.Items, lsp.CompletionItem{Label: decl.Name, Kind: &kind, Detail: variantHover(info, variant).detail})
	}
	return list
}

func variantHover(info *sema.EnumInfo, variant sema.VariantInfo) hoverInfo {
	detail := info.Decl.Name + "." + variant.Decl.Name
	if variant.Payload.Kind != sema.Void {
		detail += "(" + variant.Payload.String() + ")"
	}
	return hoverInfo{detail: detail}
}

func enumHover(model *sema.Model, pos ast.Pos) (hoverInfo, bool) {
	for _, info := range model.Enums {
		if info.Decl.NamePos == pos {
			return hoverInfo{detail: "enum " + info.Decl.Name}, true
		}
		for _, variant := range info.Variants {
			if variant.Decl.Pos == pos {
				return variantHover(info, variant), true
			}
		}
	}
	for expr, constructor := range model.Constructors {
		if call, ok := expr.(*ast.CallExpr); ok {
			expr = call.Callee
		}
		if member, ok := expr.(*ast.MemberExpr); ok {
			if member.NamePos == pos {
				return variantHover(constructor.Enum, constructor.Variant), true
			}
			if member.Object.Position() == pos {
				return hoverInfo{detail: "enum " + constructor.Enum.Decl.Name}, true
			}
		}
		if variant, ok := expr.(*ast.ContextualVariantExpr); ok && variant.NamePos == pos {
			return variantHover(constructor.Enum, constructor.Variant), true
		}
	}
	return hoverInfo{}, false
}
