package parser

import (
	"strings"
	"unicode"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/token"
)

type parser struct {
	invalidNames       map[string]ast.Pos
	errors             diagnostic.List
	aliases            map[string]bool
	interfaceSignature bool
	filename           string
	tokens             []token.Token
	index              int
	// spacedCondition is set while a `si` condition is read again with the
	// spacing rule (see parseCondition); conditionStart is its first token.
	spacedCondition bool
	conditionStart  int
	// headerResult marks the result type of a function header, where a spaced
	// `!` attached to an expression starts the body (see parseTypeRef).
	headerResult bool
	// interpolation is set while parsing the expression inside "${...}".
	interpolation bool
}

func Parse(filename string, tokens []token.Token) (*ast.Program, error) {
	p := &parser{filename: filename, tokens: tokens, aliases: map[string]bool{}}
	program := &ast.Program{InvalidNames: map[string]ast.Pos{}}
	p.invalidNames = program.InvalidNames
	for !p.at(token.EOF) {
		if p.match(token.Newline) {
			continue
		}
		var decl ast.Decl
		start := p.index
		if p.match(token.Invalid) {
			if name := p.tokens[start].Lexeme; name != "" {
				program.InvalidNames[name] = p.tokens[start].Pos
			}
			p.synchronize(start, true)
			continue
		}
		if p.at(token.Usar) {
			if len(program.Decls) != 0 {
				p.errors.Add(p.error(p.current(), "'usar' debe preceder a las declaraciones"))
			}
			imp, err := p.parseImport()
			if err != nil {
				p.report(start, err)
				p.synchronize(start, true)
				continue
			}
			program.Imports = append(program.Imports, imp)
			p.aliases[imp.Alias] = true
			continue
		}
		public := p.match(token.Pub)
		var err error
		switch p.current().Kind {
		case token.Interfaz:
			decl, err = p.parseInterfaceDecl()
		case token.Enum:
			decl, err = p.parseEnumDecl()
		case token.Tipo:
			decl, err = p.parseTypeDecl()
		case token.Fn:
			decl, err = p.parseFuncDecl("")
		case token.Var, token.Const:
			decl, err = p.parseGlobalDecl()
		default:
			err = p.error(p.current(), "se esperaba una declaración 'tipo', 'enum', 'fn', 'var' o 'const'")
		}
		if err != nil {
			p.report(start, err)
			nameIndex := start + 1
			if public {
				nameIndex++
			}
			if nameIndex < len(tokens) && tokens[nameIndex].Kind == token.Ident {
				program.InvalidNames[tokens[nameIndex].Lexeme] = tokens[nameIndex].Pos
			}
			p.synchronize(start, true)
			continue
		}
		ast.SetPublic(decl, public)
		if f, ok := decl.(*ast.FuncDecl); ok && public && f.Name == "inicio" {
			p.errors.Add(p.error(tokens[start], "inicio no puede ser pub"))
		}
		program.Decls = append(program.Decls, decl)
	}
	return program, p.errors.Err()
}

func (p *parser) report(start int, err error) {
	if p.at(token.Invalid) || p.index > start && p.previous().Kind == token.Invalid {
		return
	}
	p.errors.Add(err)
}

// synchronize restarts at a sibling line, skipping the failed statement's suite.
// Scanning from start also handles parsers which consumed a boundary before failing.
func (p *parser) synchronize(start int, top bool) {
	depth := 0
	for i := 0; i < start; i++ {
		if p.tokens[i].Kind == token.Indent {
			depth++
		}
		if p.tokens[i].Kind == token.Dedent {
			depth--
		}
	}
	base := depth
	p.index = start
	for !p.at(token.EOF) {
		t := p.advance()
		if t.Kind == token.Indent {
			depth++
		}
		if t.Kind == token.Dedent {
			depth--
		}
		if p.index <= start {
			continue
		}
		if top {
			if depth == 0 && (t.Kind == token.Newline || t.Kind == token.Dedent) && !p.at(token.Indent) {
				return
			}
		} else if depth < base {
			p.index--
			return
		} else if depth == base && (t.Kind == token.Newline || t.Kind == token.Dedent) && !p.at(token.Indent) {
			return
		}
	}
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
	// Numeric type names are also compiler-owned callable/member names.
	if kind == token.Ident && (p.at(token.Entero) || p.at(token.Decimal)) {
		return p.advance(), nil
	}
	if !p.at(kind) {
		return token.Token{}, p.error(p.current(), message)
	}
	return p.advance(), nil
}

func (p *parser) error(at token.Token, message string) error {
	if strings.HasPrefix(message, "se esperaba") {
		message += p.foundSuffix(at, message)
	}
	return &diagnostic.Error{Stage: diagnostic.Parser, Filename: p.filename, Pos: at.Pos, Message: message}
}

// foundSuffix says what the parser saw instead, with a hint for mistakes
// borrowed from other languages ('+=', '++', 'while', ...).
func (p *parser) foundSuffix(at token.Token, message string) string {
	var found string
	switch at.Kind {
	case token.EOF:
		found = "el final del archivo"
		if p.interpolation {
			found = "el final de la interpolación"
		}
	case token.Newline:
		found = "el final de la línea"
	case token.Indent:
		found = "una línea indentada"
	case token.Dedent:
		found = "el fin del bloque"
	default:
		found = "'" + at.Lexeme + "'"
	}
	out := ", pero se encontró " + found
	if hint := diagnostic.Foreign(at.Lexeme); hint != "" && at.Kind == token.Ident {
		return out + "; " + hint
	}
	if at.Kind != token.Ident && isWord(at.Lexeme) && strings.Contains(message, "nombre") {
		return ", pero '" + at.Lexeme + "' es una palabra reservada de Cometa y no puede usarse como nombre"
	}
	previous := func(back int) token.Kind {
		if p.index-back < 0 || p.index-back >= len(p.tokens) {
			return ""
		}
		return p.tokens[p.index-back].Kind
	}
	switch {
	case (at.Kind == token.Plus || at.Kind == token.Minus) && previous(1) == at.Kind:
		return out + "; Cometa no tiene '++' ni '--'; escribe x = x + 1"
	}
	return out
}

func isWord(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if !unicode.IsLetter(r) && r != '_' {
			return false
		}
	}
	return true
}
