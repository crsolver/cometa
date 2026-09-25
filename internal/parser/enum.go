package parser

import (
	"hacha/internal/ast"
	"hacha/internal/token"
)

func (p *parser) parseEnumDecl() (*ast.EnumDecl, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre del enum")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParams()
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Newline, "se esperaba una línea nueva después del enum"); err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Indent, "se esperaba un bloque indentado de variantes"); err != nil {
		return nil, err
	}
	decl := &ast.EnumDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, TypeParams: typeParams}
	for !p.at(token.Dedent) && !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		start := p.index
		err := func() error {
			variant, err := p.expect(token.Ident, "se esperaba el nombre de una variante")
			if err != nil {
				return err
			}
			v := &ast.VariantDecl{Pos: variant.Pos, Name: variant.Lexeme}
			if !p.at(token.Newline) {
				payload, err := p.parseTypeRef()
				if err != nil {
					return err
				}
				v.Payload = &payload
			}
			if _, err = p.expect(token.Newline, "se esperaba el final de la variante"); err != nil {
				return err
			}
			decl.Variants = append(decl.Variants, v)
			return nil
		}()
		if err != nil {
			p.report(start, err)
			p.invalidNames[decl.Name] = decl.Pos
			p.synchronize(start, false)
		}
	}
	if len(decl.Variants) == 0 && p.invalidNames[decl.Name].Line == 0 {
		return nil, p.error(p.current(), "un enum requiere variantes")
	}
	_, err = p.expect(token.Dedent, "se esperaba el final del enum")
	return decl, err
}

func (p *parser) parseMatch() (*ast.MatchExpr, error) {
	start := p.advance()
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	m := &ast.MatchExpr{Pos: start.Pos, Value: value}
	if p.match(token.Pipe) {
		binding, err := p.expect(token.Ident, "se esperaba el nombre del payload")
		if err != nil {
			return nil, err
		}
		m.Binding, m.BindingPos = binding.Lexeme, binding.Pos
		if _, err = p.expect(token.Pipe, "se esperaba '|' después del nombre del payload"); err != nil {
			return nil, err
		}
	}
	if _, err = p.expect(token.Newline, "se esperaba una línea nueva después de casos"); err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Indent, "se esperaba un bloque indentado de casos"); err != nil {
		return nil, err
	}
	for !p.at(token.Dedent) && !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		start := p.index
		err := func() error {
			arm := &ast.MatchArm{Pos: p.current().Pos}
			if p.match(token.Dot) {
				pattern, err := p.expect(token.Ident, "se esperaba una variante")
				if err != nil {
					return err
				}
				if pattern.Lexeme == "_" {
					return p.error(pattern, "el comodín debe escribirse '_'")
				}
				arm.Pattern, arm.NamePos = pattern.Lexeme, pattern.Pos
			} else if p.at(token.Ident) && p.current().Lexeme == "_" {
				pattern := p.advance()
				arm.Pattern, arm.NamePos = "_", pattern.Pos
			} else {
				ref, err := p.parseTypeRef()
				if err != nil {
					return err
				}
				if p.match(token.Dot) {
					arm.Qualifier, arm.QualifierPos, arm.QualifierType = ref.Name, ref.Pos, &ref
					pattern, err := p.expect(token.Ident, "se esperaba una variante después de '.'")
					if err != nil {
						return err
					}
					if pattern.Lexeme == "_" {
						return p.error(pattern, "el comodín debe escribirse '_'")
					}
					arm.Pattern, arm.NamePos = pattern.Lexeme, pattern.Pos
				} else {
					arm.TypePattern, arm.NamePos = &ref, ref.Pos
				}
			}
			if _, err = p.expect(token.Arrow, "se esperaba '=>' después del patrón"); err != nil {
				return err
			}
			body, err := p.parseSuite()
			if err != nil {
				return err
			}
			arm.Body = body
			m.Arms = append(m.Arms, arm)
			return nil
		}()
		if err != nil {
			p.report(start, err)
			m.Arms = append(m.Arms, &ast.MatchArm{Pos: p.tokens[start].Pos, Invalid: true})
			p.synchronize(start, false)
		}
	}
	if len(m.Arms) == 0 {
		return nil, p.error(p.current(), "casos requiere al menos una rama")
	}
	_, err = p.expect(token.Dedent, "se esperaba el final de casos")
	return m, err
}
