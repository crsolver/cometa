package parser

import (
	"cometa/internal/ast"
	"cometa/internal/token"
	"strings"
)

func baseName(name string) string {
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

func qualifiedName(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		return e.Name, true
	case *ast.MemberExpr:
		if id, ok := e.Object.(*ast.IdentExpr); ok {
			return id.Name + "." + e.Name, true
		}
	}
	return "", false
}

func qualifiedPosition(expr ast.Expr) ast.Pos {
	if member, ok := expr.(*ast.MemberExpr); ok {
		return member.Object.Position()
	}
	return expr.Position()
}

func (p *parser) parseImport() (*ast.ImportDecl, error) {
	start := p.advance()
	i := &ast.ImportDecl{Pos: start.Pos, PathPos: p.current().Pos}
	end := p.current().Pos.Column
	consume := func() token.Token {
		t := p.advance()
		end = t.Pos.Column + len([]rune(t.Lexeme))
		i.Path += t.Lexeme
		return t
	}
	if p.at(token.Dot) || p.at(token.Range) {
		for p.at(token.Dot) || p.at(token.Range) {
			if p.current().Pos.Column != end {
				return nil, p.error(p.current(), "ruta de módulo inválida")
			}
			prefix := consume()
			if !p.at(token.Slash) || p.current().Pos.Column != end {
				return nil, p.error(p.current(), "se esperaba '/' en la ruta")
			}
			consume()
			if prefix.Kind == token.Dot {
				break
			}
		}
	}
	for {
		if !p.at(token.Ident) || p.current().Pos.Column != end {
			return nil, p.error(p.current(), "se esperaba un nombre de módulo sin extensión")
		}
		name := consume()
		i.Alias, i.AliasPos = name.Lexeme, name.Pos
		if !p.at(token.Slash) {
			break
		}
		if p.current().Pos.Column != end {
			return nil, p.error(p.current(), "la ruta no admite espacios")
		}
		consume()
	}
	if p.match(token.Como) {
		alias, err := p.expect(token.Ident, "se esperaba un alias después de 'como'")
		if err != nil {
			return nil, err
		}
		i.Alias, i.AliasPos = alias.Lexeme, alias.Pos
	}
	if i.Alias == "_" {
		return nil, p.error(start, "el alias no puede ser '_'")
	}
	if _, err := p.expect(token.Newline, "la ruta de 'usar' debe omitir la extensión y terminar en una línea nueva"); err != nil {
		return nil, err
	}
	return i, nil
}
