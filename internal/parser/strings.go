package parser

import (
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/lexer"
	"github.com/crsolver/cometa/internal/token"
)

func (p *parser) parseString(tok token.Token) (ast.Expr, error) {
	runes := []rune(tok.Lexeme)
	var parts []ast.InterpolatedStringPart
	segment := 1
	interpolated := false
	for i := 1; i < len(runes)-1; {
		if runes[i] == '\\' {
			i += 2
			continue
		}
		if runes[i] != '$' || i+1 >= len(runes)-1 || runes[i+1] != '{' {
			i++
			continue
		}
		interpolated = true
		text, err := decodeStringSegment(runes[segment:i])
		if err != nil {
			return nil, p.error(tok, "cadena inválida: "+err.Error())
		}
		parts = append(parts, ast.InterpolatedStringPart{Text: text})
		end, ok := interpolationEnd(runes, i+2)
		if !ok {
			return nil, &diagnostic.Error{Stage: diagnostic.Parser, Filename: p.filename, Pos: ast.Pos{Line: tok.Pos.Line, Column: tok.Pos.Column + i}, Message: "interpolación sin cerrar"}
		}
		if end == i+2 {
			return nil, &diagnostic.Error{Stage: diagnostic.Parser, Filename: p.filename, Pos: ast.Pos{Line: tok.Pos.Line, Column: tok.Pos.Column + i}, Message: "la interpolación no puede estar vacía"}
		}
		expression, err := p.parseInterpolation(string(runes[i+2:end]), tok.Pos.Column+i+2, tok.Pos.Line)
		if err != nil {
			return nil, err
		}
		parts = append(parts, ast.InterpolatedStringPart{Expr: expression})
		i, segment = end+1, end+1
	}
	if !interpolated {
		if hasEscapedInterpolation(runes[1 : len(runes)-1]) {
			text, err := decodeStringSegment(runes[1 : len(runes)-1])
			if err != nil {
				return nil, p.error(tok, "cadena inválida: "+err.Error())
			}
			return &ast.LiteralExpr{Pos: tok.Pos, Kind: "cadena", Value: strconv.Quote(text)}, nil
		}
		return &ast.LiteralExpr{Pos: tok.Pos, Kind: "cadena", Value: tok.Lexeme}, nil
	}
	text, err := decodeStringSegment(runes[segment : len(runes)-1])
	if err != nil {
		return nil, p.error(tok, "cadena inválida: "+err.Error())
	}
	parts = append(parts, ast.InterpolatedStringPart{Text: text})
	return &ast.InterpolatedStringExpr{Pos: tok.Pos, Parts: parts}, nil
}

func hasEscapedInterpolation(raw []rune) bool {
	for i := 0; i+2 < len(raw); i++ {
		if raw[i] == '\\' {
			if raw[i+1] == '$' && raw[i+2] == '{' {
				return true
			}
			i++
		}
	}
	return false
}

func decodeStringSegment(raw []rune) (string, error) {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+2 < len(raw) && raw[i+1] == '$' && raw[i+2] == '{' {
			b.WriteString("${")
			i += 2
			continue
		}
		b.WriteRune(raw[i])
	}
	return strconv.Unquote("\"" + b.String() + "\"")
}

func interpolationEnd(runes []rune, start int) (int, bool) {
	depth := 1
	for i := start; i < len(runes)-1; {
		if runes[i] == '"' {
			end, ok := stringEnd(runes, i)
			if !ok {
				return 0, false
			}
			i = end
			continue
		}
		if runes[i] == '{' {
			depth++
		}
		if runes[i] == '}' {
			depth--
			if depth == 0 {
				return i, true
			}
		}
		i++
	}
	return 0, false
}

func stringEnd(runes []rune, start int) (int, bool) {
	for i := start + 1; i < len(runes); i++ {
		if runes[i] == '\\' {
			i++
			continue
		}
		if runes[i] == '$' && i+1 < len(runes) && runes[i+1] == '{' {
			end, ok := interpolationEnd(runes, i+2)
			if !ok {
				return 0, false
			}
			i = end
			continue
		}
		if runes[i] == '"' {
			return i + 1, true
		}
	}
	return 0, false
}

func (p *parser) parseInterpolation(source string, column, line int) (ast.Expr, error) {
	tokens, err := lexer.LexExpression(p.filename, source, ast.Pos{Filename: p.tokens[0].Pos.Filename, Line: line, Column: column})
	if err != nil {
		failure := *err.(*diagnostic.Error)
		failure.Stage = diagnostic.Parser
		return nil, &failure
	}
	sub := &parser{filename: p.filename, tokens: tokens, aliases: p.aliases, interpolation: true}
	expr, err := sub.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if !sub.at(token.EOF) {
		return nil, sub.error(sub.current(), "contenido inesperado en la interpolación")
	}
	return expr, nil
}
