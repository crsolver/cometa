package parser

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/token"
)

func (p *parser) parseTypeRef() (ast.TypeRef, error) {
	var base ast.TypeRef
	var err error
	header := p.headerResult
	p.headerResult = false
	if p.at(token.Bang) {
		base = ast.TypeRef{Pos: p.current().Pos, Name: "$unidad"}
		header = false
	} else {
		base, err = p.parseTypeAtom()
		if err != nil {
			return base, err
		}
	}
	for p.at(token.Question) || p.at(token.Bang) {
		// `fn en_suelo() bool !@cabe()`: the result mark is written against its
		// type (`bool!`), so a `!` after a space and attached to what follows
		// negates the one-line body.
		if header && p.at(token.Bang) && p.index+1 < len(p.tokens) {
			bang, before, after := p.current(), p.previous(), p.tokens[p.index+1]
			spaced := before.Pos.Column+len([]rune(before.Lexeme)) < bang.Pos.Column
			attached := after.Kind != token.Newline && after.Pos.Line == bang.Pos.Line && after.Pos.Column == bang.Pos.Column+1
			if spaced && attached {
				break
			}
		}
		op := p.advance()
		payload := base
		base = ast.TypeRef{Pos: payload.Pos, Wrapper: op.Lexeme, Payload: &payload}
		if op.Kind == token.Bang {
			errorType := ast.TypeRef{Pos: op.Pos, Name: "cadena"}
			adjacent := p.current().Pos.Line == op.Pos.Line && p.current().Pos.Column == op.Pos.Column+1
			if adjacent && (p.at(token.Ident) || (p.at(token.Entero) || p.at(token.Decimal)) || p.at(token.Cadena) || p.at(token.Bool) || p.at(token.LParen) || p.at(token.LBracket)) {
				errorType, err = p.parseTypeAtom()
				if err != nil {
					return base, err
				}
			}
			base.ErrorType = &errorType
			return base, nil
		}
	}
	return base, nil
}

func (p *parser) parseTypeAtom() (ast.TypeRef, error) {
	if p.match(token.LParen) {
		ref, err := p.parseTypeRef()
		if err != nil {
			return ref, err
		}
		_, err = p.expect(token.RParen, "se esperaba ')' después del tipo")
		return ref, err
	}
	if p.match(token.LBracket) {
		start := p.previous()
		element, err := p.parseTypeRef()
		if err != nil {
			return ast.TypeRef{}, err
		}
		var key *ast.TypeRef
		if p.match(token.Colon) {
			k := element
			key = &k
			element, err = p.parseTypeRef()
			if err != nil {
				return ast.TypeRef{}, err
			}
		}
		if _, err = p.expect(token.RBracket, "se esperaba ']' después del tipo de elemento"); err != nil {
			return ast.TypeRef{}, err
		}
		return ast.TypeRef{Pos: start.Pos, Key: key, Element: &element}, nil
	}
	current := p.current()
	if current.Kind != token.Entero && current.Kind != token.Decimal && current.Kind != token.Cadena && current.Kind != token.Bool && current.Kind != token.Ident {
		return ast.TypeRef{}, p.error(current, "se esperaba un tipo")
	}
	p.advance()
	ref := ast.TypeRef{Pos: current.Pos, Name: current.Lexeme}
	if p.aliases[ref.Name] && p.match(token.Dot) {
		name, err := p.expect(token.Ident, "se esperaba el nombre del tipo importado")
		if err != nil {
			return ref, err
		}
		ref.Name += "." + name.Lexeme
	}
	if p.at(token.Less) {
		var err error
		ref.Args, err = p.parseTypeArgs()
		if err != nil {
			return ref, err
		}
	}
	return ref, nil
}

func (p *parser) startsDefiniteType() bool {
	return (p.at(token.Entero) || p.at(token.Decimal)) || p.at(token.Cadena) || p.at(token.Bool) || p.at(token.LBracket) || p.at(token.Bang) || p.at(token.LParen)
}
