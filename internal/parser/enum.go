package parser

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/token"
	"strings"
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
			} else if p.at(token.Number) || p.at(token.String) || p.at(token.True) || p.at(token.False) || p.at(token.Minus) {
				// Literal arms over entero, cadena or bool: `1 =>`, `2, 3 =>`, `1..5 =>`, `"a" =>`, `verdadero =>`.
				arm.NamePos = p.current().Pos
				for {
					literal, err := p.parseLabel()
					if err != nil {
						return err
					}
					arm.Literals = append(arm.Literals, literal)
					if !p.match(token.Comma) {
						break
					}
				}
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
				} else if (p.at(token.Comma) || p.at(token.Range)) && simpleName(ref) {
					// `moneda, pinchos =>` or `minimo..maximo =>`: constant names over entero, cadena or bool.
					arm.NamePos = ref.Pos
					first, err := p.parseLabelRange(&ast.IdentExpr{Pos: ref.Pos, Name: ref.Name})
					if err != nil {
						return err
					}
					arm.Literals = append(arm.Literals, first)
					for p.match(token.Comma) {
						literal, err := p.parseLabel()
						if err != nil {
							return err
						}
						arm.Literals = append(arm.Literals, literal)
					}
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

// simpleName reports whether a parsed type is just an identifier, which may
// also be the name of a constant used as a `casos` label.
func simpleName(t ast.TypeRef) bool {
	return t.Name != "" && !strings.Contains(t.Name, ".") && t.Wrapper == "" && t.Element == nil && t.Key == nil && t.Payload == nil && t.ErrorType == nil && len(t.Args) == 0
}

// parseLabel parses one value of a scalar `casos` arm: a literal, the name of
// a constant, or a range between two of those (`1..5`).
func (p *parser) parseLabel() (ast.Expr, error) {
	start, err := p.parseLabelValue()
	if err != nil {
		return nil, err
	}
	return p.parseLabelRange(start)
}

func (p *parser) parseLabelValue() (ast.Expr, error) {
	if p.at(token.Ident) {
		name := p.advance()
		return &ast.IdentExpr{Pos: name.Pos, Name: name.Lexeme}, nil
	}
	return p.parsePrefix()
}

// parseLabelRange turns an already parsed label into a range when `..` follows.
func (p *parser) parseLabelRange(start ast.Expr) (ast.Expr, error) {
	if !p.match(token.Range) {
		return start, nil
	}
	if p.at(token.Arrow) || p.at(token.Comma) {
		return nil, p.error(p.current(), "se esperaba el fin del rango después de '..'")
	}
	end, err := p.parseLabelValue()
	if err != nil {
		return nil, err
	}
	return &ast.RangeLabel{Pos: start.Position(), Start: start, End: end}, nil
}
