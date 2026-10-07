package parser

import (
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/token"
)

var precedence = map[token.Kind]int{
	token.Or: 1, token.And: 2,
	token.Equal: 3, token.NotEqual: 3,
	token.Less: 4, token.LessEq: 4, token.Greater: 4, token.GreaterEq: 4,
	token.Plus: 5, token.Minus: 5,
	token.Star: 6, token.Slash: 6, token.Percent: 6,
}

// parseCondition reads the condition of a `si` or `osi`. A short conditional
// has no mark between its condition and its value, so `si izquierda -1 sino 0`
// first reads as the subtraction `izquierda - 1` with no value left. When the
// plain reading runs straight into `sino`/`osi`, the condition is read again
// with the spacing rule: a `.`, `-` or `(` written after a space and attached
// to what follows (` .Claro`, ` -1`, ` (-1)`) starts the branch value instead.
func (p *parser) parseCondition() (ast.Expr, error) {
	start := p.index
	condition, err := p.parseExpression(0)
	if err != nil || !p.at(token.Sino) && !p.at(token.Osi) {
		return condition, err
	}
	end := p.index
	outerSpaced, outerStart := p.spacedCondition, p.conditionStart
	p.index, p.spacedCondition, p.conditionStart = start, true, start
	spaced, spacedErr := p.parseExpression(0)
	p.spacedCondition, p.conditionStart = outerSpaced, outerStart
	if spacedErr != nil || p.index == end {
		p.index = end
		return condition, nil
	}
	return spaced, nil
}

// startsBranchValue applies the spacing rule of parseCondition to the current
// token. Brackets opened inside the condition suspend it.
func (p *parser) startsBranchValue() bool {
	if !p.spacedCondition || p.index == 0 || p.index+1 >= len(p.tokens) {
		return false
	}
	current, before, after := p.current(), p.previous(), p.tokens[p.index+1]
	if current.Kind != token.Dot && current.Kind != token.Minus && current.Kind != token.LParen {
		return false
	}
	spaceBefore := before.Pos.Line == current.Pos.Line && before.Pos.Column+len([]rune(before.Lexeme)) < current.Pos.Column
	attached := after.Pos.Line == current.Pos.Line && after.Pos.Column == current.Pos.Column+1
	if !spaceBefore || !attached {
		return false
	}
	depth := 0
	for _, tok := range p.tokens[p.conditionStart:p.index] {
		switch tok.Kind {
		case token.LParen, token.LBracket, token.LBrace:
			depth++
		case token.RParen, token.RBracket, token.RBrace:
			depth--
		}
	}
	return depth == 0
}

func (p *parser) parseExpression(minPrecedence int) (ast.Expr, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}
	for {
		// A block expression has consumed its line boundary already. Do not
		// interpret a following statement as a postfix call or index.
		if p.previous().Kind == token.Dedent {
			break
		}
		if p.startsBranchValue() {
			break
		}
		if p.at(token.Como) && minPrecedence <= 5 {
			op := p.advance()
			target, parseErr := p.parseTypeRef()
			if parseErr != nil {
				return nil, parseErr
			}
			left = &ast.AssertExpr{Pos: op.Pos, Value: left, Target: target}
			continue
		}
		if name, ok := qualifiedName(left); ok && p.at(token.Less) {
			saved := p.index
			args, parseErr := p.parseTypeArgs()
			if parseErr == nil && (p.at(token.LParen) || p.at(token.LBrace) || p.at(token.Dot)) {
				left = &ast.InstantiateExpr{Pos: qualifiedPosition(left), Name: name, Args: args}
				continue
			}
			p.index = saved
		}
		if minPrecedence == 0 && (p.at(token.Fallback) || p.at(token.Catch)) {
			op := p.advance()
			recovery := &ast.RecoverExpr{Pos: op.Pos, Value: left, Error: op.Kind == token.Catch}
			if recovery.Error {
				recovery.Binding, recovery.BindingPos, err = p.parseBinding()
				if err != nil {
					return nil, err
				}
			}
			if p.at(token.Newline) {
				recovery.Body, err = p.parseSuite()
			} else {
				var value ast.Expr
				value, err = p.parseExpression(0)
				if err == nil {
					recovery.Body = []ast.Stmt{&ast.ExprStmt{Pos: value.Position(), Expr: value}}
				}
			}
			if err != nil {
				return nil, err
			}
			left = recovery
			continue
		}
		if p.at(token.LBrace) {
			if instance, ok := left.(*ast.InstantiateExpr); ok {
				literal, parseErr := p.parseStructLiteral(instance.Pos, instance.Name)
				if parseErr != nil {
					return nil, parseErr
				}
				literal.(*ast.StructLiteralExpr).Type = &ast.TypeRef{Pos: instance.Pos, Name: instance.Name, Args: instance.Args}
				left = literal
				continue
			}
			name, ok := qualifiedName(left)
			if !ok {
				return nil, p.error(p.current(), "solo un nombre de tipo puede preceder a '{'")
			}
			left, err = p.parseStructLiteral(qualifiedPosition(left), name)
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.match(token.Dot) {
			dot := p.previous()
			// `texto.repetir(3)`: the loop keyword names a string method.
			if p.at(token.Repetir) && p.peekAt(1, token.LParen) {
				p.tokens[p.index].Kind = token.Ident
			}
			name, memberErr := p.expect(token.Ident, "se esperaba un miembro después de '.'")
			if memberErr != nil {
				return nil, memberErr
			}
			left = &ast.MemberExpr{Pos: dot.Pos, Object: left, NamePos: name.Pos, Name: name.Lexeme}
			continue
		}
		if p.at(token.LParen) {
			left, err = p.parseCall(left)
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.at(token.LBracket) {
			left, err = p.parseIndex(left)
			if err != nil {
				return nil, err
			}
			continue
		}
		prec, ok := precedence[p.current().Kind]
		if !ok || prec < minPrecedence {
			break
		}
		op := p.advance()
		right, parseErr := p.parseExpression(prec + 1)
		if parseErr != nil {
			return nil, parseErr
		}
		left = &ast.BinaryExpr{Pos: op.Pos, Left: left, Operator: op.Lexeme, Right: right}
	}
	return left, nil
}

func (p *parser) parsePrefix() (ast.Expr, error) {
	current := p.current()
	switch current.Kind {
	case token.Try:
		p.advance()
		value, err := p.parseExpression(7)
		return &ast.TryExpr{Pos: current.Pos, Value: value}, err
	case token.Return:
		p.advance()
		ret := &ast.ReturnExpr{Pos: current.Pos}
		if !p.at(token.Newline) && !p.at(token.Dedent) && !p.at(token.EOF) && !p.at(token.Sino) && !p.at(token.Osi) && !p.at(token.RParen) && !p.at(token.Comma) {
			var err error
			ret.Value, err = p.parseExpression(0)
			if err != nil {
				return nil, err
			}
		}
		return ret, nil
	case token.Dot:
		p.advance()
		name, err := p.expect(token.Ident, "se esperaba una variante después de '.'")
		if err != nil {
			return nil, err
		}
		return &ast.ContextualVariantExpr{Pos: current.Pos, NamePos: name.Pos, Name: name.Lexeme}, nil
	case token.Casos:
		return p.parseMatch()
	case token.Number:
		p.advance()
		kind := "entero"
		if strings.Contains(current.Lexeme, ".") {
			kind = "decimal"
		}
		if base := map[string]int{"0x": 16, "0b": 2}[current.Lexeme[:min(2, len(current.Lexeme))]]; base != 0 {
			// The rest of the compiler only sees the decimal spelling.
			value, err := strconv.ParseInt(current.Lexeme[2:], base, 64)
			if err != nil {
				return nil, p.error(current, "literal entero fuera del rango de entero (el máximo es 9223372036854775807)")
			}
			return &ast.LiteralExpr{Pos: current.Pos, Kind: kind, Value: strconv.FormatInt(value, 10)}, nil
		}
		return &ast.LiteralExpr{Pos: current.Pos, Kind: kind, Value: current.Lexeme}, nil
	case token.String:
		p.advance()
		return p.parseString(current)
	case token.True, token.False:
		p.advance()
		return &ast.LiteralExpr{Pos: current.Pos, Kind: "bool", Value: current.Lexeme}, nil
	case token.Cadena:
		// cadena(valor) converts like "${valor}", so it reuses interpolation.
		if !p.peekAt(1, token.LParen) {
			return nil, p.error(current, "se esperaba una expresión")
		}
		p.advance()
		p.advance()
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(token.RParen, "se esperaba ')' después del valor de cadena(...)"); err != nil {
			return nil, err
		}
		return &ast.InterpolatedStringExpr{Pos: current.Pos, Parts: []ast.InterpolatedStringPart{{Expr: value}}}, nil
	case token.Ident, token.Entero, token.Decimal:
		p.advance()
		return &ast.IdentExpr{Pos: current.Pos, Name: current.Lexeme}, nil
	case token.At:
		p.advance()
		if !p.at(token.Ident) {
			return &ast.ReceiverExpr{Pos: current.Pos, NamePos: current.Pos}, nil
		}
		name, err := p.expect(token.Ident, "se esperaba un miembro después de '@'")
		if err != nil {
			return nil, err
		}
		return &ast.ReceiverExpr{Pos: current.Pos, NamePos: name.Pos, Name: name.Lexeme}, nil
	case token.Minus, token.Bang:
		p.advance()
		value, err := p.parseExpression(7)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Pos: current.Pos, Operator: current.Lexeme, Value: value}, nil
	case token.LParen:
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(token.RParen, "se esperaba ')'"); err != nil {
			return nil, err
		}
		return expr, nil
	case token.LBrace:
		return p.parseStructLiteral(current.Pos, "")
	case token.LBracket:
		return p.parseListLiteral()
	case token.Si:
		return p.parseIfExpr()
	default:
		return nil, p.error(current, "se esperaba una expresión")
	}
}

func (p *parser) parseCall(callee ast.Expr) (ast.Expr, error) {
	start := p.advance()
	multiline := p.match(token.Newline)
	if multiline {
		if _, err := p.expect(token.Indent, "se esperaba indentación para los argumentos"); err != nil {
			return nil, err
		}
	}
	var args []ast.Expr
	var info []ast.ArgumentInfo
	if !p.at(token.RParen) {
		for {
			meta := ast.ArgumentInfo{Pos: p.current().Pos}
			if p.at(token.Ident) && p.peekAt(1, token.Assign) {
				meta.Name = p.advance().Lexeme
				p.advance()
			}
			arg, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			meta.Spread = p.match(token.Ellipsis)
			info = append(info, meta)
			if !p.match(token.Comma) {
				break
			}
			if multiline {
				p.match(token.Newline)
				if p.at(token.Dedent) {
					break
				}
			}
			if p.at(token.RParen) {
				break
			}
		}
	}
	if multiline {
		p.match(token.Newline)
		if _, err := p.expect(token.Dedent, "se esperaba el final de los argumentos"); err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(token.RParen, "se esperaba ')' después de los argumentos"); err != nil {
		return nil, err
	}
	return &ast.CallExpr{Pos: start.Pos, Callee: callee, Args: args, ArgInfo: info}, nil
}

func (p *parser) parseIndex(object ast.Expr) (ast.Expr, error) {
	start := p.advance()
	index, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(token.RBracket, "se esperaba ']' después del índice"); err != nil {
		return nil, err
	}
	return &ast.IndexExpr{Pos: start.Pos, Object: object, Index: index}, nil
}

func (p *parser) parseIfExpr() (ast.Expr, error) {
	startIndex := p.index
	start := p.advance()
	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	binding, bindingPos, err := p.parseBinding()
	if err != nil {
		return nil, err
	}
	if p.at(token.Newline) {
		p.index = startIndex
		stmt, err := p.parseIfStmt()
		if err != nil {
			return nil, err
		}
		conditional := stmt.(*ast.IfStmt)
		if len(conditional.Else) == 0 {
			return nil, p.error(p.current(), "una expresión 'si' requiere una rama 'sino'")
		}
		return &ast.BlockExpr{Pos: start.Pos, Body: []ast.Stmt{stmt}}, nil
	}
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, p.error(p.current(), "se esperaba el valor de la rama 'si'")
	}
	expr := &ast.IfExpr{Pos: start.Pos, Branches: []ast.IfExprBranch{{Pos: start.Pos, Condition: condition, Value: value, Binding: binding, BindingPos: bindingPos}}}
	for p.match(token.Osi) {
		branchStart := p.previous()
		branchCondition, parseErr := p.parseCondition()
		if parseErr != nil {
			return nil, parseErr
		}
		binding, bindingPos, parseErr := p.parseBinding()
		if parseErr != nil {
			return nil, parseErr
		}
		branchValue, parseErr := p.parseExpression(0)
		if parseErr != nil {
			return nil, p.error(p.current(), "se esperaba el valor de la rama 'osi'")
		}
		expr.Branches = append(expr.Branches, ast.IfExprBranch{Pos: branchStart.Pos, Condition: branchCondition, Value: branchValue, Binding: binding, BindingPos: bindingPos})
	}
	if _, err = p.expect(token.Sino, "una expresión 'si' requiere una rama 'sino'"); err != nil {
		return nil, err
	}
	expr.Else, err = p.parseExpression(0)
	if err != nil {
		return nil, p.error(p.current(), "se esperaba el valor de la rama 'sino'")
	}
	return expr, nil
}

func (p *parser) parseBinding() (string, ast.Pos, error) {
	if !p.match(token.Pipe) {
		return "", ast.Pos{}, nil
	}
	name, err := p.expect(token.Ident, "se esperaba una variable entre '|'")
	if err != nil {
		return "", ast.Pos{}, err
	}
	_, err = p.expect(token.Pipe, "se esperaba '|' después de la variable")
	return name.Lexeme, name.Pos, err
}
