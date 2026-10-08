package stdlib

import (
	"sort"
	"strings"
	"sync"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/lexer"
	"github.com/crsolver/cometa/internal/parser"
)

// reference is the navigable source of a module: what the editor opens at a
// cometa-std:/// URI, with the place of each declaration in it.
type reference struct {
	text string
	// places holds where each declaration starts, by its key: the name of a
	// type or function, or "Tipo.metodo".
	places map[string]place
}

type place struct{ line, column int }

var references sync.Map // *Module -> *reference

// Source exposes navigable declarations. Bodies are documentation stubs; the
// compiler links their private identities to the bundled native implementations.
func Source(path string) (string, bool) {
	m, ok := LookupModule(path)
	if !ok {
		return "", false
	}
	return m.reference().text, true
}

// URI is the address of the module's reference source.
func (m *Module) URI() string { return "cometa-std:///" + m.Path + ".cometa" }

func (m *Module) reference() *reference {
	if cached, ok := references.Load(m); ok {
		return cached.(*reference)
	}
	r := &reference{places: map[string]place{}}
	var b strings.Builder
	// declare writes a line that declares key, which starts at the given text.
	declare := func(key, line, start string) {
		r.places[key] = place{line: strings.Count(b.String(), "\n") + 1, column: strings.Index(line, start)}
		b.WriteString(line + "\n")
	}
	b.WriteString("// Biblioteca estándar de Cometa: " + m.Path + "\n")
	writeDoc(&b, "", m.Doc)
	for _, note := range m.Notes {
		b.WriteString("// " + note + "\n")
	}
	for i := range m.Types {
		t := &m.Types[i]
		writeDoc(&b, "", t.Doc)
		// The constants close the comment of the type, so the editor shows
		// them with it.
		if len(t.Constants) > 0 {
			var constants []string
			for constant := range t.Constants {
				constants = append(constants, "."+constant)
			}
			sort.Strings(constants)
			b.WriteString("// Valores: " + strings.Join(constants, ", ") + "\n")
		}
		if t.Interface {
			declare(t.Name, "pub interfaz "+t.Name, "pub interfaz")
			for _, method := range t.Methods {
				declare(t.Name+"."+method.Name, "\tfn "+method.Name+method.Signature, "fn ")
			}
			continue
		}
		declare(t.Name, "pub tipo "+t.Name, "pub tipo")
		for _, field := range t.Fields {
			b.WriteString("\tpub " + field + "\n")
		}
		for _, method := range t.Methods {
			writeDoc(&b, "\t", method.Doc)
			declare(t.Name+"."+method.Name, "\tpub fn "+method.Name+method.Signature, "fn ")
			b.WriteString("\t\timprimir(0)\n")
		}
	}
	for _, c := range m.Constants {
		declare(c.Name, "pub const "+c.Name+" "+c.Type+" = "+c.Value, "pub const")
	}
	for _, f := range m.Functions {
		writeDoc(&b, "", f.Doc)
		declare(f.Name, "pub fn "+f.Name+f.Signature, "fn ")
		b.WriteString("\timprimir(0)\n")
	}
	r.text = b.String()
	cached, _ := references.LoadOrStore(m, r)
	return cached.(*reference)
}

// writeDoc emits doc as one or more "// " comment lines at the given
// indentation, immediately before the declaration that follows — matching
// what documentationBefore expects in the LSP hover/completion machinery.
func writeDoc(b *strings.Builder, indent, doc string) {
	if doc == "" {
		return
	}
	for _, line := range strings.Split(doc, "\n") {
		b.WriteString(indent + "// " + line + "\n")
	}
}

// parseAt parses one declaration written in Cometa as if it were the text
// that key declares in the reference of m, so its positions lead there.
func (m *Module) parseAt(key, source string) ast.Decl {
	filename := m.URI()
	tokens, err := lexer.Lex(filename, source)
	if err != nil {
		panic(err)
	}
	at, ok := m.reference().places[key]
	if !ok {
		panic("stdlib: " + m.Name + " does not declare " + key)
	}
	for i := range tokens {
		pos := &tokens[i].Pos
		if pos.Line == 1 {
			pos.Column += at.column
		}
		pos.Line += at.line - 1
		pos.Filename = filename
	}
	program, err := parser.Parse(filename, tokens)
	if err != nil {
		panic(err)
	}
	return program.Decls[0]
}

// Declaration is the function as a declaration the checker understands, with
// the library types of its signature bound to their symbols. A function is
// named by its symbol and a method by its own name, with its receiver set
// (what an interface requires has no receiver).
// Each call returns a new tree, because the checker records what it learns
// about the default values on their nodes.
func (f Function) Declaration() *ast.FuncDecl {
	key := f.Name
	if f.owner != nil {
		key = f.owner.Name + "." + f.Name
	}
	d := f.module.parseAt(key, "fn "+f.Name+f.Signature+"\n\timprimir(0)\n").(*ast.FuncDecl)
	d.Public = true
	// A default may call another function of the module: ui.contenido().
	for _, param := range d.Params {
		if call, ok := param.Default.(*ast.CallExpr); ok {
			if id, ok := call.Callee.(*ast.IdentExpr); ok {
				if symbol := FunctionSymbol(f.module.Name, id.Name); functionsByName[symbol] != nil {
					id.Name = symbol
				}
			}
		}
	}
	bindTypes(d)
	switch {
	case f.owner == nil:
		d.Name = FunctionSymbol(f.Namespace, f.Name)
	case !f.owner.Interface:
		d.Receiver = f.owner.Symbol()
	}
	return d
}

// Declaration is the type as a declaration the checker understands. An opaque
// type has no fields to declare and no position.
func (t *Type) Declaration() *ast.TypeDecl {
	if len(t.Fields) == 0 {
		return &ast.TypeDecl{Name: t.Symbol(), Public: true}
	}
	source := "pub tipo " + t.Name + "\n\tpub " + strings.Join(t.Fields, "\n\tpub ") + "\n"
	d := t.module.parseAt(t.Name, source).(*ast.TypeDecl)
	d.Public = true
	for _, field := range d.Fields {
		field.Public = true
	}
	d.Name = t.Symbol()
	bindTypes(d)
	return d
}

// bindTypes translates the library types a declaration names to their symbols.
func bindTypes(d ast.Decl) {
	ast.Inspect(d, func(node any) bool {
		if ref, ok := node.(*ast.TypeRef); ok && typesBySymbol[Symbol(ref.Name)] != nil {
			ref.Name = Symbol(ref.Name)
		}
		return true
	})
}
