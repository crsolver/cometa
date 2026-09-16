package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"hacha/internal/ast"
	"hacha/internal/codegen"
	"hacha/internal/lexer"
	"hacha/internal/parser"
	"hacha/internal/sema"
)

// SourceLoader reads canonical absolute paths. Editors can overlay unsaved buffers.
type SourceLoader func(string) ([]byte, error)

type Declaration struct {
	Module *Module
	Node   ast.Decl
	Name   string
	Symbol string
	Pos    ast.Pos
}

type Module struct {
	Path         string
	Source       []byte
	Program      *ast.Program // Original names for editor presentation.
	Bound        *ast.Program // Names resolved to graph-wide declaration identities.
	Imports      map[string]*Module
	Declarations map[string]*Declaration
	Model        *sema.Model
}

type Project struct {
	Root       *Module
	Modules    map[string]*Module
	Order      []*Module // Dependencies precede their consumers.
	Reverse    map[string][]string
	References map[ast.Pos]*Declaration
	Program    *ast.Program
	Model      *sema.Model
}

// CanonicalPath deduplicates relative paths and symlinks. Nonexistent paths remain
// usable as overlay keys and as dependencies to watch for later creation.
func CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if real, e := filepath.EvalSymlinks(abs); e == nil {
		abs = real
	}
	if runtime.GOOS == "windows" {
		abs = strings.ToLower(abs)
	}
	return abs, nil
}

func parseModule(path string, source []byte) (*ast.Program, error) {
	tokens, err := lexer.Lex(path, string(source))
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		tokens[i].Pos.Filename = path
	}
	return parser.Parse(path, tokens)
}

// AnalyzeProject returns the partial graph even on failure for dependency
// tracking and editor navigation. No writes or network access are performed.
func AnalyzeProject(entry string, loader SourceLoader) (*Project, error) {
	if loader == nil {
		loader = os.ReadFile
	}
	p := &Project{Modules: map[string]*Module{}, Reverse: map[string][]string{}, References: map[ast.Pos]*Declaration{}, Program: &ast.Program{}}
	root, err := CanonicalPath(entry)
	if err != nil {
		return p, err
	}
	active := map[string]bool{}
	var stack []string
	var load func(string) (*Module, error)
	load = func(path string) (*Module, error) {
		if m := p.Modules[path]; m != nil {
			return m, nil
		}
		source, err := loader(path)
		if err != nil {
			return nil, err
		}
		m := &Module{Path: path, Source: source, Imports: map[string]*Module{}, Declarations: map[string]*Declaration{}}
		p.Modules[path] = m
		if path == root {
			p.Root = m
		}
		m.Program, err = parseModule(path, source)
		if err != nil {
			return m, err
		}
		m.Bound, err = parseModule(path, source)
		if err != nil {
			return m, err
		}
		for _, node := range m.Program.Decls {
			name, pos := declarationName(node)
			if m.Declarations[name] != nil {
				return m, projectError(pos, "declaración duplicada %q", name)
			}
			m.Declarations[name] = &Declaration{Module: m, Node: node, Name: name, Pos: pos}
		}
		active[path] = true
		stack = append(stack, path)
		defer func() { delete(active, path); stack = stack[:len(stack)-1] }()
		seen := map[string]bool{}
		for _, imp := range m.Program.Imports {
			if m.Imports[imp.Alias] != nil || m.Declarations[imp.Alias] != nil {
				return m, projectError(imp.AliasPos, "alias duplicado o en conflicto %q", imp.Alias)
			}
			target, err := CanonicalPath(filepath.Join(filepath.Dir(path), filepath.FromSlash(imp.Path)+".hacha"))
			if err != nil {
				return m, projectError(imp.PathPos, "%s", err)
			}
			p.Reverse[target] = append(p.Reverse[target], path)
			chain := strings.Join(append(append([]string{}, stack...), target), " -> ")
			if active[target] {
				return m, projectError(imp.PathPos, "ciclo de importaciones: %s", chain)
			}
			if seen[target] {
				return m, projectError(imp.PathPos, "importación duplicada: %s", chain)
			}
			seen[target] = true
			dependency, err := load(target)
			if err != nil {
				if dependency == nil {
					return m, projectError(imp.PathPos, "no se pudo importar %s: %v (cadena: %s)", imp.Path, err, chain)
				}
				return m, err
			}
			m.Imports[imp.Alias] = dependency
			if dependency.Declarations["inicio"] != nil {
				return m, projectError(imp.PathPos, "un módulo importado no puede declarar inicio (cadena: %s)", chain)
			}
		}
		p.Order = append(p.Order, m)
		return m, nil
	}
	if _, err = load(root); err != nil {
		return p, err
	}
	// Prefixes depend on deterministic traversal, never on absolute machine paths.
	prefix := "HachaModulo"
	for {
		conflict := false
		for name := range p.Root.Declarations {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				conflict = true
			}
		}
		if !conflict {
			break
		}
		prefix += "X"
	}
	for i, m := range p.Order {
		for name, d := range m.Declarations {
			d.Symbol = name
			if m != p.Root {
				d.Symbol = fmt.Sprintf("%s%d_%s", prefix, i, name)
			}
			p.References[d.Pos] = d
		}
	}
	for _, m := range p.Order {
		b := binder{project: p, module: m}
		if err := b.bind(); err != nil {
			return p, err
		}
		p.Program.Decls = append(p.Program.Decls, m.Bound.Decls...)
	}
	p.Model, err = sema.Check(root, p.Program)
	if err != nil {
		p.Model, _ = sema.CheckForTooling(root, p.Program)
	}
	for _, m := range p.Order {
		m.Model = p.Model
	}
	if err != nil {
		return p, p.displayError(err)
	}
	return p, nil
}

func CompileProject(entry string, loader SourceLoader) ([]byte, error) {
	p, err := AnalyzeProject(entry, loader)
	if err != nil {
		return nil, err
	}
	return codegen.Generate(p.Root.Path, p.Program, p.Model)
}

func declarationName(d ast.Decl) (string, ast.Pos) {
	switch d := d.(type) {
	case *ast.TypeDecl:
		return d.Name, d.NamePos
	case *ast.EnumDecl:
		return d.Name, d.NamePos
	case *ast.InterfaceDecl:
		return d.Name, d.NamePos
	case *ast.FuncDecl:
		return d.Name, d.NamePos
	}
	panic("unknown declaration")
}

func projectError(pos ast.Pos, format string, args ...any) error {
	return &sema.Error{Filename: pos.Filename, Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// Display removes private linkage symbols from diagnostics and editor details.
func (p *Project) Display(s string) string {
	var declarations []*Declaration
	for _, m := range p.Order {
		for _, d := range m.Declarations {
			if d.Symbol != d.Name {
				declarations = append(declarations, d)
			}
		}
	}
	sort.Slice(declarations, func(i, j int) bool { return len(declarations[i].Symbol) > len(declarations[j].Symbol) })
	for _, d := range declarations {
		s = strings.ReplaceAll(s, d.Symbol, strings.TrimSuffix(filepath.Base(d.Module.Path), ".hacha")+"."+d.Name)
	}
	return s
}

func (p *Project) displayError(err error) error {
	if e, ok := err.(*sema.Error); ok {
		copy := *e
		copy.Message = p.Display(e.Message)
		return &copy
	}
	return err
}
