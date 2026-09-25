package parser

import (
	"hacha/internal/ast"
	"hacha/internal/token"
)

func (p *parser) startsNamedResult() bool {
	if p.at(token.Ident) && p.aliases[p.current().Lexeme] && p.peekAt(1, token.Dot) {
		saved := p.index
		_, err := p.parseTypeRef()
		result := err == nil && !p.at(token.LParen) && !p.at(token.Dot)
		p.index = saved
		return result
	}
	if !p.at(token.Ident) || p.peekAt(1, token.LParen) || p.peekAt(1, token.Dot) {
		return false
	}
	if p.peekAt(1, token.Less) {
		saved := p.index
		p.advance()
		_, err := p.parseTypeArgs()
		call := err == nil && (p.at(token.LParen) || p.at(token.Dot))
		p.index = saved
		if call {
			return false
		}
	}
	return true
}

func (p *parser) parseTypeParams() ([]ast.TypeParam, error) {
	if !p.match(token.Less) {
		return nil, nil
	}
	var params []ast.TypeParam
	for {
		name, err := p.expect(token.Ident, "se esperaba un parámetro de tipo")
		if err != nil {
			return nil, err
		}
		param := ast.TypeParam{Pos: name.Pos, Name: name.Lexeme}
		if p.at(token.Ident) {
			ref, err := p.parseTypeAtom()
			if err != nil {
				return nil, err
			}
			param.Constraint = &ref
		}
		params = append(params, param)
		if !p.match(token.Comma) {
			break
		}
	}
	_, err := p.expect(token.Greater, "se esperaba '>' después de los parámetros de tipo")
	return params, err
}

func (p *parser) parseTypeArgs() ([]ast.TypeRef, error) {
	if _, err := p.expect(token.Less, "se esperaba '<'"); err != nil {
		return nil, err
	}
	var args []ast.TypeRef
	for {
		ref, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		args = append(args, ref)
		if !p.match(token.Comma) {
			break
		}
	}
	_, err := p.expect(token.Greater, "se esperaba '>' después de los argumentos de tipo")
	return args, err
}

func (p *parser) parseInterfaceDecl() (*ast.InterfaceDecl, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre de la interfaz")
	if err != nil {
		return nil, err
	}
	params, err := p.parseTypeParams()
	if err != nil {
		return nil, err
	}
	d := &ast.InterfaceDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, TypeParams: params}
	if _, err = p.expect(token.Newline, "se esperaba una línea nueva después de la interfaz"); err != nil {
		return nil, err
	}
	if !p.match(token.Indent) {
		return d, nil
	}
	for !p.at(token.Dedent) && !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		start := p.index
		err := func() error {
			if p.at(token.Fn) {
				p.interfaceSignature = true
				method, err := p.parseFuncDecl("")
				p.interfaceSignature = false
				if err != nil {
					return err
				}
				d.Methods = append(d.Methods, method)
			} else {
				ref, err := p.parseTypeRef()
				if err != nil {
					return err
				}
				d.Embeds = append(d.Embeds, ref)
				if _, err = p.expect(token.Newline, "se esperaba el final de la interfaz incrustada"); err != nil {
					return err
				}
			}
			return nil
		}()
		if err != nil {
			p.report(start, err)
			p.invalidNames[d.Name] = d.Pos
			p.synchronize(start, false)
		}
	}
	_, err = p.expect(token.Dedent, "se esperaba el final de la interfaz")
	return d, err
}
