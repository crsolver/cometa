package compiler

import (
	"errors"
	"fmt"
	"io/fs"
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
	Path         string // Identity: canonical, and lower-cased on Windows.
	Display      string // The same file as people see it, for messages and editors.
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

	loader SourceLoader
}

// CanonicalPath deduplicates relative paths and symlinks. Nonexistent paths remain
// usable as overlay keys and as dependencies to watch for later creation.
func CanonicalPath(path string) (string, error) {
	key, _, err := resolvePath(path)
	return key, err
}

// resolvePath returns the identity of a file and the spelling to show for it.
// Windows file names ignore case, so the identity is lower-cased there; the
// display path keeps the case of the file on disk (or as written, if absent).
func resolvePath(path string) (key, display string, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	abs = filepath.Clean(abs)
	if real, e := filepath.EvalSymlinks(abs); e == nil {
		abs = real
	}
	key = abs
	if runtime.GOOS == "windows" {
		key = strings.ToLower(abs)
	}
	return key, abs, nil
}

func parseModule(path string, source []byte) (*ast.Program, error) {
	var errors diagnostic.List
	tokens, err := lexer.Lex(path, string(source))
	errors.Add(err)
	program, err := parser.Parse(path, tokens)
	errors.Add(err)
	return program, errors.Err()
}

// AnalyzeProject returns the partial graph even on failure for dependency
// tracking and editor navigation. No writes or network access are performed.
// Embedded resources are not read: see LoadAssets and CheckAssets.
func AnalyzeProject(entry string, loader SourceLoader) (*Project, error) {
	var errors diagnostic.List
	if loader == nil {
		loader = os.ReadFile
	}
	p := &Project{loader: loader, Modules: map[string]*Module{}, Reverse: map[string][]string{}, References: map[ast.Pos]*Declaration{}, Program: &ast.Program{InvalidNames: map[string]ast.Pos{}}}
	root, rootDisplay, err := resolvePath(entry)
	if err != nil {
		return p, err
	}
	active := map[string]bool{}
	var stack []string
	var load func(path, display string) (*Module, error)
	load = func(path, display string) (*Module, error) {
		if m := p.Modules[path]; m != nil {
			return m, nil
		}
		source, err := loader(path)
		if err != nil {
			return nil, err
		}
		m := &Module{Path: path, Display: display, Source: source, Imports: map[string]*Module{}, Declarations: map[string]*Declaration{}}
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
				errors.Add(projectError(pos, "%s", duplicateMessage(name, previous.Node, node)))
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
				native, ok := stdlib.LookupModule(imp.Path)
				if !ok {
					errors.Add(projectError(imp.PathPos, "módulo estándar desconocido %q", imp.Path))
					continue
				}
				target := native.URI()
				dependency := p.Modules[target]
				if dependency == nil {
					source, _ := stdlib.Source(imp.Path)
					dependency = &Module{Native: true, Path: target, Display: target, Source: []byte(source), Imports: map[string]*Module{}, Declarations: map[string]*Declaration{}}
					dependency.Program, _ = parseModule(target, []byte(source))
					// Opaque native types have no Cometa fields or body to parse.
					lines := strings.Split(source, "\n")
					for _, t := range native.Types {
						if t.Interface || len(t.Fields) > 0 {
							continue
						}
						for i, line := range lines {
							if line == "pub tipo "+t.Name {
								pos := ast.Pos{Filename: target, Line: i + 1, Column: 10}
								dependency.Program.Decls = append(dependency.Program.Decls, &ast.TypeDecl{Public: true, Name: t.Name, NamePos: pos, Pos: ast.Pos{Filename: target, Line: i + 1, Column: 5}})
								break
							}
						}
					}
					for _, node := range dependency.Program.Decls {
						name, pos := declarationName(node)
						symbol := stdlib.Symbol(name)
						switch node.(type) {
						case *ast.FuncDecl, *ast.GlobalDecl:
							symbol = stdlib.FunctionSymbol(native.Name, name)
						}
						dependency.Declarations[name] = &Declaration{Module: dependency, Node: node, Name: name, Symbol: symbol, Pos: pos}
					}
					p.Modules[target] = dependency
					p.Program.NativeModules = append(p.Program.NativeModules, imp.Path)
				}
				m.Imports[imp.Alias] = dependency
				continue
			}
			// Resolved from the display path of the importer, so a missing file
			// keeps the case the user wrote.
			target, targetDisplay, err := resolvePath(filepath.Join(filepath.Dir(m.Display), filepath.FromSlash(imp.Path)+".cometa"))
			if err != nil {
				errors.Add(projectError(imp.PathPos, "%s", err))
				continue
			}
			p.Reverse[target] = append(p.Reverse[target], path)
			chain := p.importChain(rootDisplay, stack, targetDisplay)
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
			dependency, err := load(target, targetDisplay)
			if err != nil {
				if dependency == nil {
					message := importFailure(imp.Path, filepath.Base(m.Display), err)
					if len(stack) > 1 {
						// With a single step the diagnostic already points at the only `usar` involved.
						message += fmt.Sprintf(" (cadena: %s)", chain)
					}
					errors.Add(projectError(imp.PathPos, "%s", message))
					m.Imports[imp.Alias] = &Module{Path: target, Display: targetDisplay, Unavailable: true, Declarations: map[string]*Declaration{}}
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
	if _, err = load(root, rootDisplay); err != nil {
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
	return p, nil
}

// LoadAssets reads and validates the resources the program embeds (images,
// fonts, sounds, maps). Generate needs them; analysis alone does not.
func (p *Project) LoadAssets() error {
	return loadAssets(p.Root.Path, p.Model, p.loader)
}

// Generate produces the Go source of a successfully analyzed project.
func (p *Project) Generate(options codegen.Options) ([]byte, error) {
	if err := p.LoadAssets(); err != nil {
		return nil, err
	}
	options.DisplayPath = p.DisplayPath
	return codegen.GenerateWithOptions(p.Root.Path, p.Program, p.Model, options)
}

// CompileProject analyzes and generates a project. Its diagnostics name files
// by their display paths; AnalyzeProject keeps the canonical ones for tooling.
func CompileProject(entry string, loader SourceLoader) ([]byte, error) {
	return compileProject(entry, loader, codegen.Options{})
}

func compileProject(entry string, loader SourceLoader, options codegen.Options) ([]byte, error) {
	p, err := AnalyzeProject(entry, loader)
	if err != nil {
		return nil, p.DisplayPaths(err)
	}
	generated, err := p.Generate(options)
	return generated, p.DisplayPaths(err)
}

// CompileProjectForRun is CompileProject with //line directives, so build
// errors and runtime panics of the generated Go point at Cometa source lines.
func CompileProjectForRun(entry string, loader SourceLoader) ([]byte, error) {
	return compileProject(entry, loader, codegen.Options{LineDirectives: true})
}

// duplicateMessage explains that name is taken, by what kind of declaration
// when it differs from the one being added.
func duplicateMessage(name string, previous, current ast.Decl) string {
	kind := func(d ast.Decl) string {
		switch d.(type) {
		case *ast.FuncDecl:
			return "función"
		case *ast.GlobalDecl:
			return "global"
		}
		return "tipo"
	}
	switch was, is := kind(previous), kind(current); {
	case was != is:
		return fmt.Sprintf("el nombre %q ya fue declarado como %s", name, was)
	case is == "tipo":
		return fmt.Sprintf("el tipo %q ya fue declarado", name)
	default:
		return fmt.Sprintf("la %s %q ya fue declarada", is, name)
	}
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
	return &diagnostic.Error{Stage: diagnostic.Sema, Filename: pos.Filename, Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// importFailure explains why the file of a `usar` could not be read, without
// the operating system's own wording or absolute paths.
func importFailure(path, importer string, err error) string {
	file := path + ".cometa"
	switch {
	case errors.Is(err, fs.ErrNotExist):
		where := "en"
		if strings.Contains(path, "/") {
			where = "desde"
		}
		return fmt.Sprintf("no existe el archivo %s (se buscó %s la carpeta de %s)", file, where, importer)
	case errors.Is(err, fs.ErrPermission):
		return fmt.Sprintf("no hay permiso para leer el archivo %s", file)
	}
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		err = pathError.Err
	}
	return fmt.Sprintf("no se pudo leer el archivo %s: %v", file, err)
}

// importChain spells the modules being loaded, and the one that failed, as
// paths relative to the folder of the entry file.
func (p *Project) importChain(rootDisplay string, stack []string, targetDisplay string) string {
	base := filepath.Dir(rootDisplay)
	var names []string
	for _, path := range stack {
		names = append(names, p.DisplayPath(path))
	}
	names = append(names, targetDisplay)
	for i, name := range names {
		if rel, err := filepath.Rel(base, name); err == nil {
			names[i] = filepath.ToSlash(rel)
		}
	}
	return strings.Join(names, " -> ")
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

// DisplayPath returns the spelling to show for the canonical path of a module.
func (p *Project) DisplayPath(path string) string {
	if m := p.Modules[path]; m != nil && m.Display != "" {
		return m.Display
	}
	return path
}

// DisplayPaths rewrites the file of every diagnostic in err to its display
// path, for output read by people. Tooling keys files by the canonical path
// and should keep the original error.
func (p *Project) DisplayPaths(err error) error {
	if err == nil {
		return nil
	}
	var out diagnostic.List
	for _, item := range diagnostic.Flatten(err) {
		if e, ok := item.(*diagnostic.Error); ok {
			copy := *e
			copy.Filename = p.DisplayPath(e.Filename)
			if e.Pos.Filename != "" {
				copy.Pos.Filename = p.DisplayPath(e.Pos.Filename)
			}
			item = &copy
		}
		out = append(out, item)
	}
	if len(out) == 1 {
		return out[0]
	}
	return out
}

func (p *Project) displayError(err error) error {
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		var out diagnostic.List
		for _, item := range many.Unwrap() {
			out.Add(p.displayError(item))
		}
		return out.Err()
	}
	if e, ok := err.(*diagnostic.Error); ok {
		copy := *e
		copy.Message = p.Display(e.Message)
		return &copy
	}
	return err
}
