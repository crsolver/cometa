package lspserver

import (
	"strings"
	"unicode/utf8"

	"github.com/owenrumney/go-lsp/lsp"
	"hacha/internal/ast"
	"hacha/internal/sema"
)

// Ignore dots in strings and comments, including escaped quotes.
func completionCodePosition(prefix string) bool {
	quoted, escaped := false, false
	for i := 0; i < len(prefix); i++ {
		ch := prefix[i]
		if quoted {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				quoted = false
			}
		} else if ch == '"' {
			quoted = true
		} else if ch == '/' && i+1 < len(prefix) && prefix[i+1] == '/' {
			return false
		}
	}
	return !quoted
}

// Close unfinished expression delimiters without changing the user's document.
func closeCompletionDelimiters(line string) string {
	var stack []byte
	quoted, escaped := false, false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if quoted {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				quoted = false
			}
			continue
		}
		if ch == '"' {
			quoted = true
			continue
		}
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			line = line[:i]
			break
		}
		switch ch {
		case '(':
			stack = append(stack, ')')
		case '[':
			stack = append(stack, ']')
		case '{':
			stack = append(stack, '}')
		case ')', ']', '}':
			if len(stack) > 0 && stack[len(stack)-1] == ch {
				stack = stack[:len(stack)-1]
			}
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		line += string(stack[i])
	}
	return line
}

func contextualCompletion(filename string, lines []string, line int, prefix string, dot, receiverStart int) (*lsp.CompletionList, bool) {
	// A dotted label starts at indentation; ordinary member expressions keep
	// their existing completion path.
	pattern := strings.TrimSpace(prefix[:receiverStart]) == ""
	if pattern {
		pattern = false
		indent := len(lines[line]) - len(strings.TrimLeft(lines[line], "\t"))
		for i := line - 1; i >= 0; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			depth := len(lines[i]) - len(strings.TrimLeft(lines[i], "\t"))
			if depth < indent {
				pattern = strings.HasPrefix(trimmed, "casos ") || strings.Contains(trimmed, " casos ") || strings.Contains(trimmed, "(casos ")
				break
			}
		}
	}
	if receiverStart != dot && !pattern {
		return nil, false
	}
	for _, r := range prefix[dot+1:] {
		if !isIdentifierRune(r) {
			return &lsp.CompletionList{}, true
		}
	}
	end := len(prefix)
	for end < len(lines[line]) {
		r, size := utf8.DecodeRuneInString(lines[line][end:])
		if !isIdentifierRune(r) {
			break
		}
		end += size
	}
	const probe = "__hacha_completion_variant__"
	replacement := prefix[:dot+1] + probe + lines[line][end:]
	if pattern {
		replacement = prefix[:dot+1] + probe + " => 0"
	}
	replacement = closeCompletionDelimiters(replacement)
	model := completionModel(filename, lines, line, replacement)
	if model == nil {
		return &lsp.CompletionList{}, true
	}
	var expected sema.Type
	if pattern {
		for arm, typ := range model.PatternTypes {
			if arm.Pattern == probe && arm.Pos.Line == line+1 && (arm.Qualifier == "" || arm.Qualifier == typ.Name) {
				expected = typ
			}
		}
	} else {
		for expr, typ := range model.ExpectedTypes {
			if variant, ok := expr.(*ast.ContextualVariantExpr); ok && variant.Name == probe && variant.Pos.Line == line+1 {
				expected = typ
			}
		}
	}
	if expected.Kind != sema.Enum {
		return &lsp.CompletionList{}, true
	}
	list := enumCompletionItems(model.Enums[expected.Name])
	for i := range list.Items {
		list.Items[i].InsertText = list.Items[i].Label
		list.Items[i].TextEdit = lsp.NewCompletionTextEdit(lsp.TextEdit{
			Range:   lsp.Range{Start: lsp.Position{Line: line, Character: utf16Length(lines[line][:dot+1])}, End: lsp.Position{Line: line, Character: utf16Length(lines[line][:end])}},
			NewText: list.Items[i].Label,
		})
	}
	return list, true
}
