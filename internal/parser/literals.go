package parser

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/token"
)

func (p *parser) parseStructLiteral(pos ast.Pos, typeName string) (ast.Expr, error) {
	if _, err := p.expect(token.LBrace, "se esperaba '{'"); err != nil {
		return nil, err
	}
	literal := &ast.StructLiteralExpr{Pos: pos, TypeName: typeName}
	if p.match(token.Newline) {
		if p.match(token.Indent) {
			for !p.at(token.Dedent) && !p.at(token.EOF) {
				if err := p.parseStructLiteralEntry(literal); err != nil {
					return nil, err
				}
				p.match(token.Comma)
				if _, err := p.expect(token.Newline, "se esperaba el final del valor de estructura"); err != nil {
					return nil, err
				}
			}
			if _, err := p.expect(token.Dedent, "se esperaba el final de los campos"); err != nil {
				return nil, err
			}
		}
		if _, err := p.expect(token.RBrace, "se esperaba '}' después del literal"); err != nil {
			return nil, err
		}
		return literal, nil
	}
	for !p.at(token.RBrace) {
		if err := p.parseStructLiteralEntry(literal); err != nil {
			return nil, err
		}
		if !p.match(token.Comma) {
			break
		}
	}
	if _, err := p.expect(token.RBrace, "se esperaba '}' después del literal"); err != nil {
		return nil, err
	}
	return literal, nil
}

func (p *parser) parseStructLiteralEntry(literal *ast.StructLiteralExpr) error {
	named := p.at(token.Ident) && p.peekAt(1, token.Colon)
	if named && len(literal.Values) > 0 || !named && len(literal.Fields) > 0 {
		return p.error(p.current(), "no se pueden mezclar valores posicionales y campos nombrados en un literal de estructura")
	}
	if named {
		field, err := p.parseFieldValue()
		if err != nil {
			return err
		}
		literal.Fields = append(literal.Fields, field)
		return nil
	}
	value, err := p.parseExpression(0)
	if err != nil {
		return err
	}
	literal.Values = append(literal.Values, value)
	return nil
}

func (p *parser) parseFieldValue() (ast.FieldValue, error) {
	name, err := p.expect(token.Ident, "se esperaba el nombre del campo")
	if err != nil {
		return ast.FieldValue{}, err
	}
	if _, err = p.expect(token.Colon, "se esperaba ':' después del nombre del campo"); err != nil {
		return ast.FieldValue{}, err
	}
	value, err := p.parseExpression(0)
	if err != nil {
		return ast.FieldValue{}, err
	}
	return ast.FieldValue{Pos: name.Pos, Name: name.Lexeme, Value: value}, nil
}

func (p *parser) parseListLiteral() (ast.Expr, error) {
	start := p.advance()
	literal := &ast.ListLiteralExpr{Pos: start.Pos}
	mapLiteral := &ast.MapLiteralExpr{Pos: start.Pos}
	isMap := false
	multiline := p.match(token.Newline)
	indented := false
	if multiline && !p.at(token.RBracket) {
		if _, err := p.expect(token.Indent, "se esperaba indentación para los elementos de la lista"); err != nil {
			return nil, err
		}
		indented = true
	}
	if p.match(token.Colon) {
		isMap = true
	} else if !p.at(token.RBracket) {
		for {
			value, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			if p.match(token.Colon) {
				if len(literal.Elements) != 0 {
					return nil, p.error(p.current(), "no se pueden mezclar elementos de lista y mapa")
				}
				isMap = true
				entryValue, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				mapLiteral.Entries = append(mapLiteral.Entries, ast.MapEntry{Key: value, Value: entryValue})
			} else {
				if isMap {
					return nil, p.error(p.current(), "se esperaba ':' después de la clave")
				}
				literal.Elements = append(literal.Elements, value)
			}
			if !p.match(token.Comma) {
				break
			}
			if multiline {
				p.match(token.Newline)
				if p.at(token.Dedent) {
					break
				}
			}
			if p.at(token.RBracket) {
				break
			}
		}
	}
	if indented {
		p.match(token.Newline)
		if _, err := p.expect(token.Dedent, "se esperaba el final de los elementos de la lista"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(token.RBracket, "se esperaba ']' después de la lista"); err != nil {
		return nil, err
	}
	if isMap {
		return mapLiteral, nil
	}
	return literal, nil
}
