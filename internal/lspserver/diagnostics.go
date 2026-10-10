package lspserver

import (
	"context"
	"strings"
	"unicode"

	"github.com/owenrumney/go-lsp/lsp"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/lexer"
)

func (h *Handler) publishDiagnostics(ctx context.Context, uri lsp.DocumentURI) error {
	return h.publishProjectDiagnostics(ctx, uri)
}

func diagnosticFromError(err error, text string) lsp.Diagnostic {
	position := ast.Pos{Line: 1, Column: 1}
	message := err.Error()
	if located, ok := err.(diagnostic.Located); ok {
		_, position, _, message = located.Diagnostic()
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
