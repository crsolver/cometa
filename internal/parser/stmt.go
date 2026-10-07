package parser

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/token"
)

func (p *parser) recoverStatement() ast.Stmt {
	start := p.index
	bad := &ast.BadStmt{Pos: p.current().Pos}
	if p.at(token.Invalid) {
		bad.Name = p.current().Lexeme
	} else {
		stmt, err := p.parseStatement()
		if err == nil {
			return stmt
		}
		p.report(start, err)
		if p.tokens[start].Kind == token.Var && start+1 < len(p.tokens) && p.tokens[start+1].Kind == token.Ident {
			bad.Name = p.tokens[start+1].Lexeme
		}
	}
	p.synchronize(start, false)
	return bad
}

func (p *parser) parseSuite() ([]ast.Stmt, error) {
	if p.match(token.Newline) {
		if _, err := p.expect(token.Indent, "se esperaba un bloque indentado"); err != nil {
			return nil, err
		}
		var body []ast.Stmt
		for !p.at(token.Dedent) && !p.at(token.EOF) {
			if p.match(token.Newline) {
				continue
			}
			body = append(body, p.recoverStatement())
		}
		if _, err := p.expect(token.Dedent, "se esperaba el final del bloque"); err != nil {
			return nil, err
		}
		if len(body) == 0 {
			return nil, p.error(p.previous(), "un bloque no puede estar vacío")
		}
		return body, nil
	}
	stmt := p.recoverStatement()
	if _, bad := stmt.(*ast.BadStmt); bad {
		return []ast.Stmt{stmt}, nil
	}
	if p.previous().Kind == token.Newline || p.previous().Kind == token.Dedent {
		return []ast.Stmt{stmt}, nil
	}
	if _, err := p.expect(token.Newline, "el cuerpo en línea debe terminar al final de la línea"); err != nil {
		return nil, err
	}
	return []ast.Stmt{stmt}, nil
}

func (p *parser) parseStatement() (ast.Stmt, error) {
	if p.at(token.Con) {
		start := p.advance()
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if !p.at(token.Newline) {
			return nil, p.error(p.current(), "con requiere un bloque indentado")
		}
		body, err := p.parseSuite()
		return &ast.ScopeStmt{Pos: start.Pos, Value: value, Body: body}, err
	}
	if p.at(token.Casos) {
		match, err := p.parseMatch()
		if err != nil {
			return nil, err
		}
		return &ast.MatchStmt{Match: match}, nil
	}
	if p.at(token.Si) {
		return p.parseIfStmt()
	}
	if p.at(token.Repetir) {
		return p.parseRepeatStmt()
	}
	if p.at(token.Mientras) {
		return p.parseWhileStmt()
	}
	if p.at(token.Var) {
		return p.parseVarDecl()
	}
	if p.at(token.Continuar) {
		start := p.advance()
		return &ast.ContinueStmt{Pos: start.Pos}, nil
	}
	if p.at(token.Romper) {
		start := p.advance()
		return &ast.BreakStmt{Pos: start.Pos}, nil
	}
	return p.parseSimpleStatement()
}

// parseWhileStmt reads `mientras condición` and lowers it to an infinite
// repetir that leaves when the condition is false:
//
//	repetir
//		si !condición
//			romper
//		...cuerpo
//
// so `continuar` re-tests the condition and later phases need no new loop kind.
func (p *parser) parseWhileStmt() (ast.Stmt, error) {
	start := p.advance()
	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	body, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	exit := &ast.IfStmt{Pos: start.Pos, Branches: []ast.IfBranch{{
		Pos:       start.Pos,
		Condition: &ast.UnaryExpr{Pos: start.Pos, Operator: "!", Value: condition},
		Body:      []ast.Stmt{&ast.BreakStmt{Pos: start.Pos}},
	}}}
	return &ast.RepeatStmt{While: true, Pos: start.Pos, Body: append([]ast.Stmt{exit}, body...)}, nil
}

func (p *parser) parseRepeatStmt() (ast.Stmt, error) {
	start := p.advance()
	stmt := &ast.RepeatStmt{Pos: start.Pos}

	// Parentheses select list or range iteration. Without them, repetir
	// introduces an infinite loop whose suite starts immediately.
	if p.match(token.LParen) {
		iterable, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if p.match(token.Range) {
			stmt.RangeEnd, err = p.parseExpression(0)
			if err != nil {
				return nil, err
			}
		}
		if _, err = p.expect(token.RParen, "se esperaba ')' después de la lista o rango del ciclo"); err != nil {
			return nil, err
		}
		if stmt.RangeEnd != nil && !p.at(token.Pipe) {
			// A range loop may omit its variables to just repeat N times.
			stmt.Iterable = iterable
			body, bodyErr := p.parseSuite()
			if bodyErr != nil {
				return nil, bodyErr
			}
			stmt.Body = body
			return stmt, nil
		}
		if _, err = p.expect(token.Pipe, "se esperaba '|' antes de las variables del ciclo"); err != nil {
			return nil, err
		}
		element, err := p.expect(token.Ident, "se esperaba el nombre del elemento del ciclo")
		if err != nil {
			return nil, err
		}
		stmt.Iterable = iterable
		stmt.ElementPos = element.Pos
		stmt.Element = element.Lexeme
		if p.match(token.Comma) {
			index, indexErr := p.expect(token.Ident, "se esperaba el nombre del índice del ciclo")
			if indexErr != nil {
				return nil, indexErr
			}
			stmt.IndexPos = index.Pos
			stmt.Index = index.Lexeme
		}
		if _, err = p.expect(token.Pipe, "se esperaba '|' después de las variables del ciclo"); err != nil {
			return nil, err
		}
	}

	body, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	stmt.Body = body
	return stmt, nil
}

func (p *parser) parseVarDecl() (ast.Stmt, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre de la variable")
	if err != nil {
		return nil, err
	}
	var declaredType *ast.TypeRef
	if !p.at(token.Assign) {
		parsed, parseErr := p.parseTypeRef()
		if parseErr != nil {
			return nil, parseErr
		}
		declaredType = &parsed
	}
	if _, err = p.expect(token.Assign, "se esperaba '=' en la declaración de variable"); err != nil {
		return nil, err
	}
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ast.VarDeclStmt{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, Type: declaredType, Value: value}, nil
}

var compoundOperators = map[token.Kind]string{
	token.PlusAssign: "+", token.MinusAssign: "-", token.StarAssign: "*",
	token.SlashAssign: "/", token.PercentAssign: "%",
}

// simpleTarget reports whether evaluating expr twice is harmless, which is
// what `target op= value` needs because it is rewritten as `target = target op value`.
func simpleTarget(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.IdentExpr, *ast.ReceiverExpr, *ast.LiteralExpr:
		return true
	case *ast.MemberExpr:
		return simpleTarget(e.Object)
	case *ast.IndexExpr:
		return simpleTarget(e.Object) && simpleTarget(e.Index)
	case *ast.UnaryExpr:
		return simpleTarget(e.Value)
	case *ast.BinaryExpr:
		return simpleTarget(e.Left) && simpleTarget(e.Right)
	}
	return false
}

func (p *parser) parseSimpleStatement() (ast.Stmt, error) {
	start := p.index
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if operator, ok := compoundOperators[p.current().Kind]; ok {
		op := p.advance()
		if !simpleTarget(expr) {
			return nil, p.error(op, "el lado izquierdo de '"+op.Lexeme+"' no puede contener llamadas a funciones; guarda ese valor en una variable antes")
		}
		value, parseErr := p.parseExpression(0)
		if parseErr != nil {
			return nil, parseErr
		}
		// Parse the target a second time so the rewritten expression owns an
		// independent tree (later passes annotate nodes in place).
		end := p.index
		p.index = start
		target, copyErr := p.parseExpression(0)
		p.index = end
		if copyErr != nil {
			return nil, copyErr
		}
		return &ast.AssignStmt{Pos: expr.Position(), Target: expr, Compound: operator,
			Value: &ast.BinaryExpr{Pos: op.Pos, Left: target, Operator: operator, Right: value}}, nil
	}
	if p.match(token.Assign) {
		value, parseErr := p.parseExpression(0)
		if parseErr != nil {
			return nil, parseErr
		}
		return &ast.AssignStmt{Pos: expr.Position(), Target: expr, Value: value}, nil
	}
	return &ast.ExprStmt{Pos: expr.Position(), Expr: expr}, nil
}

func (p *parser) parseIfStmt() (ast.Stmt, error) {
	start := p.advance()
	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	binding, bindingPos, err := p.parseBinding()
	if err != nil {
		return nil, err
	}
	body, err := p.parseConditionalSuite()
	if err != nil {
		return nil, err
	}
	stmt := &ast.IfStmt{Pos: start.Pos, Branches: []ast.IfBranch{{Pos: start.Pos, Condition: condition, Body: body, Binding: binding, BindingPos: bindingPos}}}
	for p.at(token.Osi) {
		branchStart := p.advance()
		branchCondition, parseErr := p.parseCondition()
		if parseErr != nil {
			return nil, parseErr
		}
		binding, bindingPos, parseErr := p.parseBinding()
		if parseErr != nil {
			return nil, parseErr
		}
		branchBody, parseErr := p.parseConditionalSuite()
		if parseErr != nil {
			return nil, parseErr
		}
		stmt.Branches = append(stmt.Branches, ast.IfBranch{Pos: branchStart.Pos, Condition: branchCondition, Body: branchBody, Binding: binding, BindingPos: bindingPos})
	}
	if p.match(token.Sino) {
		stmt.Else, err = p.parseConditionalSuite()
		if err != nil {
			return nil, err
		}
	}
	return stmt, nil
}

// parseConditionalSuite differs from a regular inline suite only by allowing an
// aligned osi/sino token to follow on the same physical line. This lets a final
// `si ... sino ...` act as the implicit result of a one-line function.
func (p *parser) parseConditionalSuite() ([]ast.Stmt, error) {
	if p.at(token.Newline) {
		return p.parseSuite()
	}
	var stmt ast.Stmt
	var err error
	if p.at(token.Continuar) || p.at(token.Romper) {
		stmt, err = p.parseStatement()
	} else {
		stmt, err = p.parseSimpleStatement()
	}
	if err != nil {
		return nil, err
	}
	if !p.at(token.Osi) && !p.at(token.Sino) {
		if _, err = p.expect(token.Newline, "el cuerpo en línea debe terminar al final de la línea"); err != nil {
			return nil, err
		}
	}
	return []ast.Stmt{stmt}, nil
}
