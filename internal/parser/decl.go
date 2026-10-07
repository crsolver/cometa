package parser

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/token"
)

func (p *parser) parseGlobalDecl() (*ast.GlobalDecl, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre de la global")
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
	if _, err = p.expect(token.Assign, "se esperaba '=' en la declaración global"); err != nil {
		return nil, err
	}
	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Newline, "la declaración global debe terminar al final de la línea"); err != nil {
		return nil, err
	}
	return &ast.GlobalDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, Type: declaredType, Value: value, Constant: start.Kind == token.Const}, nil
}

func (p *parser) parseTypeDecl() (*ast.TypeDecl, error) {
	start := p.advance()
	name, err := p.expect(token.Ident, "se esperaba el nombre del tipo")
	if err != nil {
		return nil, err
	}
	typeParams, err := p.parseTypeParams()
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Newline, "la declaración de tipo debe terminar en una línea nueva"); err != nil {
		return nil, err
	}
	if _, err = p.expect(token.Indent, "se esperaba un bloque indentado para el tipo"); err != nil {
		return nil, err
	}
	decl := &ast.TypeDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, TypeParams: typeParams}
	for !p.at(token.Dedent) && !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		start := p.index
		err := func() error {
			public := p.match(token.Pub)
			if p.at(token.Fn) {
				method, parseErr := p.parseFuncDecl(decl.Name)
				if parseErr != nil {
					return parseErr
				}
				method.Public = public
				decl.Methods = append(decl.Methods, method)
				return nil
			}
			// A name followed by another type starts an ordinary field. Otherwise
			// parse the complete bare type and let semantic analysis validate embedding.
			// Preserve named success-only result fields such as "error !".
			attachedResult := p.peekAt(1, token.Bang) && p.tokens[p.index+1].Pos.Column == p.current().Pos.Column+len([]rune(p.current().Lexeme))
			if !p.at(token.Ident) || p.peekAt(1, token.Newline) || p.peekAt(1, token.Less) || p.peekAt(1, token.Question) || p.peekAt(1, token.Dot) || attachedResult {
				fieldType, parseErr := p.parseTypeRef()
				if parseErr != nil {
					return parseErr
				}
				if p.at(token.Assign) {
					return p.error(p.current(), "los campos embebidos no admiten valores predeterminados")
				}
				if _, parseErr = p.expect(token.Newline, "se esperaba el final del tipo embebido"); parseErr != nil {
					return parseErr
				}
				decl.Fields = append(decl.Fields, &ast.Field{Public: public, Pos: fieldType.Pos, Name: baseName(fieldType.Name), Type: fieldType, Embedded: true})
				return nil
			}
			fieldName, parseErr := p.expect(token.Ident, "se esperaba un campo o método")
			if parseErr != nil {
				return parseErr
			}
			fieldType, parseErr := p.parseTypeRef()
			if parseErr != nil {
				return parseErr
			}
			field := &ast.Field{Public: public, Pos: fieldName.Pos, Name: fieldName.Lexeme, Type: fieldType}
			if p.match(token.Assign) {
				field.Default, parseErr = p.parseExpression(0)
				if parseErr != nil {
					return parseErr
				}
			}
			if _, parseErr = p.expect(token.Newline, "se esperaba el final de la declaración del campo"); parseErr != nil {
				return parseErr
			}
			decl.Fields = append(decl.Fields, field)
			return nil
		}()
		if err != nil {
			p.report(start, err)
			p.invalidNames[decl.Name] = decl.Pos
			p.synchronize(start, false)
		}
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
	typeParams, err := p.parseTypeParams()
	if err != nil {
		return nil, err
	}
	if receiver != "" && len(typeParams) > 0 {
		return nil, p.error(name, "los métodos no pueden declarar parámetros de tipo")
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
	if p.startsDefiniteType() || p.startsNamedResult() {
		p.headerResult = true
		parsed, parseErr := p.parseTypeRef()
		p.headerResult = false
		if parseErr != nil {
			return nil, parseErr
		}
		returnType = &parsed
	}
	if p.interfaceSignature {
		for _, param := range params {
			if param.Default != nil {
				return nil, p.error(name, "una interfaz no admite valores predeterminados")
			}
		}
		if len(typeParams) > 0 {
			return nil, p.error(name, "los métodos no pueden declarar parámetros de tipo")
		}
		_, err = p.expect(token.Newline, "una firma de interfaz no admite cuerpo")
		return &ast.FuncDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, Params: params, ReturnType: returnType}, err
	}
	body, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	return &ast.FuncDecl{Pos: start.Pos, NamePos: name.Pos, Name: name.Lexeme, Params: params, ReturnType: returnType, Body: body, Receiver: receiver, TypeParams: typeParams}, nil
}
