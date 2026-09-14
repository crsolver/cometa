package parser

import (
	"fmt"

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

type parser struct {
	filename string
	tokens   []token.Token
	index    int
}

func Parse(filename string, tokens []token.Token) (*ast.Program, error) {
	p := &parser{filename: filename, tokens: tokens}
	program := &ast.Program{}
	for !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		var decl ast.Decl
		var err error
		switch p.current().Kind {
		case token.Enum:
			decl, err = p.parseEnumDecl()
		case token.Tipo:
			decl, err = p.parseTypeDecl()
		case token.Fn:
			decl, err = p.parseFuncDecl("")
		default:
			err = p.error(p.current(), "se esperaba una declaración 'tipo', 'enum' o 'fn'")
		}
		if err != nil {
			return nil, err
		}
		program.Decls = append(program.Decls, decl)
	}
	return program, nil
}

func (p *parser) parseTypeDecl() (*ast.TypeDecl, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre del tipo")
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Newline, "la declaración de tipo debe terminar en una línea nueva"); err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Indent, "se esperaba un bloque indentado para el tipo"); err != nil {
		return nil, err
	}
	decl := &ast.TypeDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme}
	for !p.at(token.Dedent) && !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		if p.at(token.Fn) {
			method, parseErr := p.parseFuncDecl(decl.Name)
			if parseErr != nil {
				return nil, parseErr
			}
			decl.Methods = append(decl.Methods, method)
			continue
		}
		fieldName, parseErr := p.expect(token.Ident, "se esperaba un campo o método")
		if parseErr != nil {
			return nil, parseErr
		}
		fieldType, parseErr := p.parseTypeRef()
		if parseErr != nil {
			return nil, parseErr
		}
		if _, parseErr = p.expect(token.Newline, "se esperaba el final de la declaración del campo"); parseErr != nil {
			return nil, parseErr
		}
		decl.Fields = append(decl.Fields, &ast.Field{Pos: fieldName.Pos, Name: fieldName.Lexeme, Type: fieldType})
	}
	if _, err = p.expect(token.Dedent, "se esperaba el final del bloque de tipo"); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *parser) parseFuncDecl(receiver string) (*ast.FuncDecl, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre de la función")
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(token.LParen, "se esperaba '(' después del nombre de la función"); err != nil {
		return nil, err
	}
	var params []ast.Param
	if !p.at(token.RParen) {
		for {
			paramName, parseErr := p.expect(token.Ident, "se esperaba el nombre del parámetro")
			if parseErr != nil {
				return nil, parseErr
			}
			variadic := p.match(token.Ellipsis)
			paramType, parseErr := p.parseTypeRef()
			if parseErr != nil {
				return nil, parseErr
			}
			param := ast.Param{Pos: paramName.Pos, Name: paramName.Lexeme, Type: paramType, Variadic: variadic}
			if p.match(token.Assign) {
				param.Default, parseErr = p.parseExpression(0)
				if parseErr != nil {
					return nil, parseErr
				}
			}
			params = append(params, param)
			if !p.match(token.Comma) {
				break
			}
			if p.at(token.RParen) {
				break
			}
		}
	}
	if _, err = p.expect(token.RParen, "se esperaba ')' después de los parámetros"); err != nil {
		return nil, err
	}

	var returnType *ast.TypeRef
	if p.startsDefiniteType() || p.at(token.Ident) && !p.peekAt(1, token.LParen) {
		parsed, parseErr := p.parseTypeRef()
		if parseErr != nil {
			return nil, parseErr
		}
		returnType = &parsed
	}
	body, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	return &ast.FuncDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, Params: params, ReturnType: returnType, Body: body, Receiver: receiver}, nil
}

func (p *parser) parseTypeRef() (ast.TypeRef, error) {
	var base ast.TypeRef
	var err error
	if p.at(token.Bang) {
		base = ast.TypeRef{Pos: p.current().Pos, Name: "$unidad"}
	} else {
		base, err = p.parseTypeAtom()
		if err != nil {
			return base, err
		}
	}
	for p.at(token.Question) || p.at(token.Bang) {
		op := p.advance()
		payload := base
		base = ast.TypeRef{Pos: payload.Pos, Wrapper: op.Lexeme, Payload: &payload}
		if op.Kind == token.Bang {
			errorType := ast.TypeRef{Pos: op.Pos, Name: "cadena"}
			adjacent := p.current().Pos.Line == op.Pos.Line && p.current().Pos.Column == op.Pos.Column+1
			if adjacent && (p.at(token.Ident) || p.at(token.Num) || p.at(token.Cadena) || p.at(token.Bool) || p.at(token.LParen) || p.at(token.LBracket)) {
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
		if _, err = p.expect(token.RBracket, "se esperaba ']' después del tipo de elemento"); err != nil {
			return ast.TypeRef{}, err
		}
		return ast.TypeRef{Pos: start.Pos, Element: &element}, nil
	}
	current := p.current()
	if current.Kind != token.Num && current.Kind != token.Cadena && current.Kind != token.Bool && current.Kind != token.Ident {
		return ast.TypeRef{}, p.error(current, "se esperaba un tipo")
	}
	p.advance()
	return ast.TypeRef{Pos: current.Pos, Name: current.Lexeme}, nil
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
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			body = append(body, stmt)
		}
		if _, err := p.expect(token.Dedent, "se esperaba el final del bloque"); err != nil {
			return nil, err
		}
		if len(body) == 0 {
			return nil, p.error(p.previous(), "un bloque no puede estar vacío")
		}
		return body, nil
	}
	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	if p.previous().Kind == token.Newline || p.previous().Kind == token.Dedent {
		return []ast.Stmt{stmt}, nil
	}
	if _, err = p.expect(token.Newline, "el cuerpo en línea debe terminar al final de la línea"); err != nil {
		return nil, err
	}
	return []ast.Stmt{stmt}, nil
}

func (p *parser) parseStatement() (ast.Stmt, error) {
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

func (p *parser) parseSimpleStatement() (ast.Stmt, error) {
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
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
	condition, err := p.parseExpression(0)
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
		branchCondition, parseErr := p.parseExpression(0)
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

var precedence = map[token.Kind]int{
	token.Or: 1, token.And: 2,
	token.Equal: 3, token.NotEqual: 3,
	token.Less: 4, token.LessEq: 4, token.Greater: 4, token.GreaterEq: 4,
	token.Plus: 5, token.Minus: 5,
	token.Star: 6, token.Slash: 6, token.Percent: 6,
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
			ident, ok := left.(*ast.IdentExpr)
			if !ok {
				return nil, p.error(p.current(), "solo un nombre de tipo puede preceder a '{'")
			}
			left, err = p.parseStructLiteral(ident.Pos, ident.Name)
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.match(token.Dot) {
			dot := p.previous()
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
		return &ast.LiteralExpr{Pos: current.Pos, Kind: "num", Value: current.Lexeme}, nil
	case token.String:
		p.advance()
		return &ast.LiteralExpr{Pos: current.Pos, Kind: "cadena", Value: current.Lexeme}, nil
	case token.True, token.False:
		p.advance()
		return &ast.LiteralExpr{Pos: current.Pos, Kind: "bool", Value: current.Lexeme}, nil
	case token.Ident:
		p.advance()
		return &ast.IdentExpr{Pos: current.Pos, Name: current.Lexeme}, nil
	case token.At:
		p.advance()
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

func (p *parser) parseStructLiteral(pos ast.Pos, typeName string) (ast.Expr, error) {
	if _, err := p.expect(token.LBrace, "se esperaba '{'"); err != nil {
		return nil, err
	}
	literal := &ast.StructLiteralExpr{Pos: pos, TypeName: typeName}
	if p.match(token.Newline) {
		if p.match(token.Indent) {
			for !p.at(token.Dedent) && !p.at(token.EOF) {
				field, err := p.parseFieldValue()
				if err != nil {
					return nil, err
				}
				literal.Fields = append(literal.Fields, field)
				p.match(token.Comma)
				if _, err = p.expect(token.Newline, "se esperaba el final del campo"); err != nil {
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
		field, err := p.parseFieldValue()
		if err != nil {
			return nil, err
		}
		literal.Fields = append(literal.Fields, field)
		if !p.match(token.Comma) {
			break
		}
	}
	if _, err := p.expect(token.RBrace, "se esperaba '}' después del literal"); err != nil {
		return nil, err
	}
	return literal, nil
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
	if !p.at(token.RBracket) {
		for {
			value, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			literal.Elements = append(literal.Elements, value)
			if !p.match(token.Comma) {
				break
			}
		}
	}
	if _, err := p.expect(token.RBracket, "se esperaba ']' después de la lista"); err != nil {
		return nil, err
	}
	return literal, nil
}

func (p *parser) parseCall(callee ast.Expr) (ast.Expr, error) {
	start := p.advance()
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
			if p.at(token.RParen) {
				break
			}
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
	condition, err := p.parseExpression(0)
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
		branchCondition, parseErr := p.parseExpression(0)
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

func (p *parser) startsDefiniteType() bool {
	return p.at(token.Num) || p.at(token.Cadena) || p.at(token.Bool) || p.at(token.LBracket) || p.at(token.Bang) || p.at(token.LParen)
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

func (p *parser) current() token.Token {
	if p.index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.index]
}

func (p *parser) previous() token.Token { return p.tokens[p.index-1] }

func (p *parser) advance() token.Token {
	current := p.current()
	if p.index < len(p.tokens) {
		p.index++
	}
	return current
}

func (p *parser) at(kind token.Kind) bool { return p.current().Kind == kind }

func (p *parser) peekAt(offset int, kind token.Kind) bool {
	index := p.index + offset
	return index < len(p.tokens) && p.tokens[index].Kind == kind
}

func (p *parser) match(kind token.Kind) bool {
	if !p.at(kind) {
		return false
	}
	p.advance()
	return true
}

func (p *parser) expect(kind token.Kind, message string) (token.Token, error) {
	if !p.at(kind) {
		return token.Token{}, p.error(p.current(), message)
	}
	return p.advance(), nil
}

func (p *parser) error(at token.Token, message string) error {
	return &Error{Filename: p.filename, Pos: at.Pos, Message: message}
}
