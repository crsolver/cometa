package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/codegen"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/lexer"
	"github.com/crsolver/cometa/internal/parser"
	"github.com/crsolver/cometa/internal/sema"
	"github.com/crsolver/cometa/internal/stdlib"
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
	Native       bool
	Unavailable  bool
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
	var errors diagnostic.List
	tokens, err := lexer.Lex(path, string(source))
	errors.Add(err)
	for i := range tokens {
		tokens[i].Pos.Filename = path
	}
	program, err := parser.Parse(path, tokens)
	errors.Add(err)
	return program, errors.Err()
}

// AnalyzeProject returns the partial graph even on failure for dependency
// tracking and editor navigation. No writes or network access are performed.
func AnalyzeProject(entry string, loader SourceLoader) (*Project, error) {
	var errors diagnostic.List
	if loader == nil {
		loader = os.ReadFile
	}
	p := &Project{Modules: map[string]*Module{}, Reverse: map[string][]string{}, References: map[ast.Pos]*Declaration{}, Program: &ast.Program{InvalidNames: map[string]ast.Pos{}}}
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
		errors.Add(err)
		m.Bound, _ = parseModule(path, source)
		for name, pos := range m.Program.InvalidNames {
			m.Declarations[name] = &Declaration{Module: m, Name: name, Pos: pos}
		}
		for _, node := range m.Program.Decls {
			name, pos := declarationName(node)
			if previous := m.Declarations[name]; previous != nil && previous.Node != nil {
				errors.Add(projectError(pos, "declaración duplicada %q", name))
				continue
			}
			m.Declarations[name] = &Declaration{Module: m, Node: node, Name: name, Pos: pos}
		}
		active[path] = true
		stack = append(stack, path)
		defer func() { delete(active, path); stack = stack[:len(stack)-1] }()
		seen := map[string]bool{}
		for _, imp := range m.Program.Imports {
			if m.Imports[imp.Alias] != nil || m.Declarations[imp.Alias] != nil {
				errors.Add(projectError(imp.AliasPos, "alias duplicado o en conflicto %q", imp.Alias))
				continue
			}
			if strings.HasPrefix(imp.Path, "std/") {
				if seen[imp.Path] {
					errors.Add(projectError(imp.PathPos, "importación duplicada: %s", imp.Path))
					continue
				}
				seen[imp.Path] = true
				source, ok := stdlib.Source(imp.Path)
				if !ok {
					errors.Add(projectError(imp.PathPos, "módulo estándar desconocido %q", imp.Path))
					continue
				}
				target := "cometa-std:///" + imp.Path + ".cometa"
				dependency := p.Modules[target]
				if dependency == nil {
					dependency = &Module{Native: true, Path: target, Source: []byte(source), Imports: map[string]*Module{}, Declarations: map[string]*Declaration{}}
					dependency.Program, _ = parseModule(target, []byte(source))
					ns, _ := stdlib.Namespace(imp.Path)
					// Opaque native types have no Cometa fields or body to parse.
					for name, owner := range stdlib.TypeModules {
						if owner != ns || name == "Juego" || stdlib.Fields[stdlib.Symbol(name)] != "" {
							continue
						}
						for i, line := range strings.Split(source, "\n") {
							if line == "pub tipo "+name {
								pos := ast.Pos{Filename: target, Line: i + 1, Column: 10}
								dependency.Program.Decls = append(dependency.Program.Decls, &ast.TypeDecl{Public: true, Name: name, NamePos: pos, Pos: ast.Pos{Filename: target, Line: i + 1, Column: 5}})
								break
							}
						}
					}
					for _, node := range dependency.Program.Decls {
						name, pos := declarationName(node)
						symbol := stdlib.Symbol(name)
						if _, ok := node.(*ast.FuncDecl); ok {
							symbol = stdlib.FunctionSymbol(ns, name)
						}
						if name == "pi" {
							symbol = stdlib.FunctionSymbol(ns, name)
						}
						dependency.Declarations[name] = &Declaration{Module: dependency, Node: node, Name: name, Symbol: symbol, Pos: pos}
					}
					p.Modules[target] = dependency
					p.Program.NativeModules = append(p.Program.NativeModules, imp.Path)
				}
				m.Imports[imp.Alias] = dependency
				continue
			}
			target, err := CanonicalPath(filepath.Join(filepath.Dir(path), filepath.FromSlash(imp.Path)+".cometa"))
			if err != nil {
				errors.Add(projectError(imp.PathPos, "%s", err))
				continue
			}
			p.Reverse[target] = append(p.Reverse[target], path)
			chain := strings.Join(append(append([]string{}, stack...), target), " -> ")
			if active[target] {
				errors.Add(projectError(imp.PathPos, "ciclo de importaciones: %s", chain))
				m.Imports[imp.Alias] = p.Modules[target]
				continue
			}
			if seen[target] {
				errors.Add(projectError(imp.PathPos, "importación duplicada: %s", chain))
				m.Imports[imp.Alias] = p.Modules[target]
				continue
			}
			seen[target] = true
			dependency, err := load(target)
			if err != nil {
				if dependency == nil {
					errors.Add(projectError(imp.PathPos, "no se pudo importar %s: %v (cadena: %s)", imp.Path, err, chain))
					m.Imports[imp.Alias] = &Module{Path: target, Unavailable: true, Declarations: map[string]*Declaration{}}
					continue
				}
				errors.Add(err)
			}
			m.Imports[imp.Alias] = dependency
			if dependency.Declarations["inicio"] != nil {
				errors.Add(projectError(imp.PathPos, "un módulo importado no puede declarar inicio (cadena: %s)", chain))
			}
		}
		p.Order = append(p.Order, m)
		return m, nil
	}
	if _, err = load(root); err != nil {
		return p, err
	}
	// Prefixes depend on deterministic traversal, never on absolute machine paths.
	prefix := "CometaModulo"
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
			if _, invalid := m.Program.InvalidNames[name]; invalid {
				p.Program.InvalidNames[d.Symbol] = d.Pos
			}
		}
	}
	for _, m := range p.Order {
		// Keep the first declaration identity; duplicates were diagnosed while loading.
		decls := m.Bound.Decls[:0]
		for _, node := range m.Bound.Decls {
			name, pos := declarationName(node)
			if d := m.Declarations[name]; d != nil && d.Pos == pos {
				decls = append(decls, node)
			}
		}
		m.Bound.Decls = decls
		b := binder{project: p, module: m}
		if err := b.bind(); err != nil {
			errors.Add(err)
		}
		p.Program.Decls = append(p.Program.Decls, m.Bound.Decls...)
	}
	p.Model, err = sema.Check(root, p.Program)
	errors.Add(err)
	for _, m := range p.Order {
		m.Model = p.Model
	}
	if err := errors.Err(); err != nil {
		return p, p.displayError(err)
	}
	if err = loadAssets(root, p.Model, loader); err != nil {
		return p, err
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

// CompileProjectForRun is CompileProject with //line directives, so build
// errors and runtime panics of the generated Go point at Cometa source lines.
func CompileProjectForRun(entry string, loader SourceLoader) ([]byte, error) {
	p, err := AnalyzeProject(entry, loader)
	if err != nil {
		return nil, err
	}
	return codegen.GenerateWithOptions(p.Root.Path, p.Program, p.Model, codegen.Options{LineDirectives: true})
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
	case *ast.GlobalDecl:
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
	for _, m := range p.Modules {
		for _, d := range m.Declarations {
			if d.Symbol != d.Name {
				declarations = append(declarations, d)
			}
		}
	}
	sort.Slice(declarations, func(i, j int) bool { return len(declarations[i].Symbol) > len(declarations[j].Symbol) })
	for _, d := range declarations {
		s = strings.ReplaceAll(s, d.Symbol, strings.TrimSuffix(filepath.Base(d.Module.Path), ".cometa")+"."+d.Name)
	}
	return stdlib.Display(s)
}

func (p *Project) displayError(err error) error {
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var out diagnostic.List
		for _, item := range many.Unwrap() {
			out.Add(p.displayError(item))
		}
		return out.Err()
	}
	if e, ok := err.(*sema.Error); ok {
		copy := *e
		copy.Message = p.Display(e.Message)
		return &copy
	}
	return err
}
