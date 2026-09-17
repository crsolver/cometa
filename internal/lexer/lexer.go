package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"hacha/internal/ast"
	"hacha/internal/token"
)

type Error struct {
	Filename string
	Pos      ast.Pos
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Filename, e.Pos.Line, e.Pos.Column, e.Message)
}

var keywords = map[string]token.Kind{
	"usar":     token.Usar,
	"interfaz": token.Interfaz, "como": token.Como,
	"o": token.Fallback, "capturar": token.Catch, "intentar": token.Try, "retornar": token.Return,
	"enum": token.Enum, "casos": token.Casos,
	"tipo": token.Tipo, "fn": token.Fn, "si": token.Si, "osi": token.Osi,
	"var": token.Var, "const": token.Const,
	"sino": token.Sino, "num": token.Num, "cadena": token.Cadena, "bool": token.Bool,
	"repetir": token.Repetir, "continuar": token.Continuar, "romper": token.Romper,
	"verdadero": token.True, "falso": token.False,
}

// Lex converts source text into tokens, including Python-style INDENT and DEDENT
// markers. Tabs are the only permitted indentation characters.
func Lex(filename, source string) ([]token.Token, error) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	lines := strings.Split(source, "\n")
	var out []token.Token
	indents := []int{0}

	for lineIndex, raw := range lines {
		lineNo := lineIndex + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		runes := []rune(raw)
		i := 0
		for i < len(runes) && runes[i] == '\t' {
			i++
		}
		if i < len(runes) && runes[i] == ' ' {
			return nil, lexError(filename, lineNo, i+1, "la indentación debe usar tabuladores, no espacios")
		}
		level := i
		current := indents[len(indents)-1]
		switch {
		case level > current:
			indents = append(indents, level)
			out = append(out, token.Token{Kind: token.Indent, Pos: ast.Pos{Line: lineNo, Column: 1}})
		case level < current:
			for len(indents) > 1 && level < indents[len(indents)-1] {
				indents = indents[:len(indents)-1]
				out = append(out, token.Token{Kind: token.Dedent, Pos: ast.Pos{Line: lineNo, Column: 1}})
			}
			if level != indents[len(indents)-1] {
				return nil, lexError(filename, lineNo, 1, "la dedentación no coincide con un bloque anterior")
			}
		}

		lineTokens, err := lexLine(filename, lineNo, i, runes)
		if err != nil {
			return nil, err
		}
		out = append(out, lineTokens...)
		out = append(out, token.Token{Kind: token.Newline, Pos: ast.Pos{Line: lineNo, Column: len(runes) + 1}})
	}

	eofLine := len(lines)
	for len(indents) > 1 {
		indents = indents[:len(indents)-1]
		out = append(out, token.Token{Kind: token.Dedent, Pos: ast.Pos{Line: eofLine, Column: 1}})
	}
	out = append(out, token.Token{Kind: token.EOF, Pos: ast.Pos{Line: eofLine, Column: 1}})
	return out, nil
}

func lexLine(filename string, lineNo, start int, runes []rune) ([]token.Token, error) {
	var out []token.Token
	for i := start; i < len(runes); {
		ch := runes[i]
		if ch == ' ' || ch == '\t' {
			i++
			continue
		}
		pos := ast.Pos{Line: lineNo, Column: i + 1}
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			break
		}
		if unicode.IsLetter(ch) || ch == '_' {
			j := i + 1
			for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			word := string(runes[i:j])
			kind := token.Ident
			if keyword, ok := keywords[word]; ok {
				kind = keyword
			}
			out = append(out, token.Token{Kind: kind, Lexeme: word, Pos: pos})
			i = j
			continue
		}
		if unicode.IsDigit(ch) {
			j := i + 1
			for j < len(runes) && unicode.IsDigit(runes[j]) {
				j++
			}
			if j < len(runes) && runes[j] == '.' && !(j+1 < len(runes) && runes[j+1] == '.') {
				j++
				if j >= len(runes) || !unicode.IsDigit(runes[j]) {
					return nil, lexError(filename, lineNo, j+1, "se esperaba un dígito después del punto decimal")
				}
				for j < len(runes) && unicode.IsDigit(runes[j]) {
					j++
				}
			}
			out = append(out, token.Token{Kind: token.Number, Lexeme: string(runes[i:j]), Pos: pos})
			i = j
			continue
		}
		if ch == '"' {
			j, ok := scanString(runes, i)
			if !ok {
				if at := unclosedInterpolation(runes, i+1); at >= 0 {
					return nil, lexError(filename, lineNo, at+1, "interpolación sin cerrar")
				}
				return nil, lexError(filename, lineNo, i+1, "cadena sin cerrar")
			}
			out = append(out, token.Token{Kind: token.String, Lexeme: string(runes[i:j]), Pos: pos})
			i = j
			continue
		}

		if i+2 < len(runes) && string(runes[i:i+3]) == "..." {
			out = append(out, token.Token{Kind: token.Ellipsis, Lexeme: "...", Pos: pos})
			i += 3
			continue
		}
		if i+1 < len(runes) {
			pair := string(runes[i : i+2])
			pairs := map[string]token.Kind{"..": token.Range, "=>": token.Arrow, "==": token.Equal, "!=": token.NotEqual, "<=": token.LessEq, ">=": token.GreaterEq, "&&": token.And, "||": token.Or}
			if kind, ok := pairs[pair]; ok {
				out = append(out, token.Token{Kind: kind, Lexeme: pair, Pos: pos})
				i += 2
				continue
			}
		}
		singles := map[rune]token.Kind{
			'(': token.LParen, ')': token.RParen, '[': token.LBracket, ']': token.RBracket,
			'{': token.LBrace, '}': token.RBrace, ',': token.Comma, ':': token.Colon, '.': token.Dot,
			'|': token.Pipe,
			'@': token.At, '=': token.Assign, '+': token.Plus, '-': token.Minus,
			'*': token.Star, '/': token.Slash, '%': token.Percent, '<': token.Less,
			'>': token.Greater, '!': token.Bang, '?': token.Question,
		}
		kind, ok := singles[ch]
		if !ok {
			return nil, lexError(filename, lineNo, i+1, fmt.Sprintf("carácter inesperado %q", ch))
		}
		out = append(out, token.Token{Kind: kind, Lexeme: string(ch), Pos: pos})
		i++
	}
	return out, nil
}

func unclosedInterpolation(runes []rune, start int) int {
	for i := start; i+1 < len(runes); i++ {
		if runes[i] == '\\' {
			i++
			continue
		}
		if runes[i] == '$' && runes[i+1] == '{' {
			if _, ok := scanInterpolation(runes, i+2); !ok {
				return i
			}
		}
	}
	return -1
}

// scanString understands interpolation so quotes and braces inside ${...}
// cannot accidentally terminate the containing string.
func scanString(runes []rune, start int) (int, bool) {
	for i := start + 1; i < len(runes); {
		if runes[i] == '\\' {
			i += 2
			continue
		}
		if runes[i] == '"' {
			return i + 1, true
		}
		if runes[i] == '$' && i+1 < len(runes) && runes[i+1] == '{' {
			end, ok := scanInterpolation(runes, i+2)
			if !ok {
				return 0, false
			}
			i = end
			continue
		}
		i++
	}
	return 0, false
}

func scanInterpolation(runes []rune, start int) (int, bool) {
	depth := 1
	for i := start; i < len(runes); {
		switch runes[i] {
		case '"':
			end, ok := scanString(runes, i)
			if !ok {
				return 0, false
			}
			i = end
			continue
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
		i++
	}
	return 0, false
}

func lexError(filename string, line, column int, message string) error {
	return &Error{Filename: filename, Pos: ast.Pos{Line: line, Column: column}, Message: message}
}
