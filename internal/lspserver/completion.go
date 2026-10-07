package lspserver

import (
	"context"

	"github.com/owenrumney/go-lsp/lsp"

	"github.com/crsolver/cometa/internal/ast"
)

// Completion offers fields and methods after a dot or the receiver marker @.
// The current line is made syntactically complete before analysis so
// completion still works while a member name is incomplete.
func (h *Handler) Completion(_ context.Context, params *lsp.CompletionParams) (*lsp.CompletionList, error) {
	text, ok := h.documents.Text(params.TextDocument.URI)
	if !ok {
		return &lsp.CompletionList{}, nil
	}
	return h.moduleCompletion(params, text), nil
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
