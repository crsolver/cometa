package stdlib

import (
	"bytes"
	"embed"
	"encoding/base64"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// The runtime is the Go code of package rt, embedded as text.
//
//go:embed rt/*.go
var runtimeFS embed.FS

// runtimeFiles lists the files of package rt that hold the runtime, in the
// order their declarations are written to a generated program. doc.go is left
// out: it documents the package and stands in for what programs define.
var runtimeFiles = []string{
	"pincel.go", "extra.go", "lienzo.go", "crt.go", "curvas.go", "ruido.go",
	"pruebas.go", "consola.go", "ui_core.go", "ui.go", "retro.go",
}

//go:embed assets/dungeon-mode.png
var dungeonAtlas []byte

//go:embed assets/dungeon-437.png
var asciiAtlas []byte

// parsedRuntime is the runtime as syntax, ready to pick declarations from.
type parsedRuntime struct {
	fset  *token.FileSet
	decls []ast.Decl
	// byName finds the declarations of a name; the methods of a type are
	// found under the name of the type.
	byName map[string][]ast.Decl
	// imports gives, for each declaration, the packages its file imports by
	// the name the file uses for them.
	imports map[ast.Decl]map[string]string
}

// loadRuntime parses package rt once: every program picks from the same
// read-only syntax.
var loadRuntime = sync.OnceValues(func() (*parsedRuntime, error) {
	rt := &parsedRuntime{fset: token.NewFileSet(), byName: map[string][]ast.Decl{}, imports: map[ast.Decl]map[string]string{}}
	for _, name := range runtimeFiles {
		source, err := runtimeFS.ReadFile("rt/" + name)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(rt.fset, name, source, 0)
		if err != nil {
			return nil, err
		}
		imports := map[string]string{}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			imports[importName(spec, imported)] = imported
		}
		for _, d := range file.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				name := d.Name.Name
				if d.Recv != nil {
					t := d.Recv.List[0].Type
					if p, ok := t.(*ast.StarExpr); ok {
						t = p.X
					}
					name = t.(*ast.Ident).Name
				}
				rt.byName[name] = append(rt.byName[name], d)
			case *ast.GenDecl:
				if d.Tok == token.IMPORT {
					continue
				}
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						rt.byName[s.Name.Name] = append(rt.byName[s.Name.Name], d)
					case *ast.ValueSpec:
						for _, n := range s.Names {
							rt.byName[n.Name] = append(rt.byName[n.Name], d)
						}
					}
				}
			}
			rt.decls = append(rt.decls, d)
			rt.imports[d] = imports
		}
	}
	return rt, nil
})

// importName is the name a file refers to an imported package by: the one it
// gives it, or the last element of its path that is not a version.
func importName(spec *ast.ImportSpec, imported string) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	name := path.Base(imported)
	if len(name) > 1 && name[0] == 'v' && strings.Trim(name[1:], "0123456789") == "" {
		name = path.Base(path.Dir(imported))
	}
	return name
}

// Runtime selects the declarations reachable from what the imported modules
// declare, and the Go imports those declarations need. Go syntax supplies the
// dependencies, so helpers and imports cannot drift apart.
func Runtime(modules []string) (string, []string, error) {
	return RuntimeFor(modules, nil)
}

// RuntimeFor is Runtime for a program whose library calls are known: called
// holds "namespace.name" for each one. A module marked OnlyCalled contributes
// just those functions; the others contribute everything they declare.
func RuntimeFor(modules []string, called map[string]bool) (string, []string, error) {
	if len(modules) == 0 {
		return "", nil, nil
	}
	rt, err := loadRuntime()
	if err != nil {
		return "", nil, err
	}
	selected := map[ast.Decl]bool{}
	imports := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		for _, d := range rt.byName[name] {
			if selected[d] {
				continue
			}
			selected[d] = true
			ast.Inspect(d, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					visit(id.Name)
				}
				if s, ok := n.(*ast.SelectorExpr); ok {
					// A name the file declares, such as a variable, is not a package.
					if id, ok := s.X.(*ast.Ident); ok && id.Obj == nil {
						if imported := rt.imports[d][id.Name]; imported != "" {
							imports[strconv.Quote(imported)] = true
						}
					}
				}
				return true
			})
		}
	}
	for _, module := range modules {
		m, ok := LookupModule(module)
		if !ok {
			continue
		}
		for _, f := range m.Functions {
			if m.OnlyCalled && called != nil && !called[m.Name+"."+f.Name] {
				continue
			}
			if f.Visual {
				if !UsesPincel(modules) {
					continue
				}
				visit("_hgpruebasBucle")
				visit("_hgpruebasReiniciar")
			}
			visit("_hg" + f.GoName)
			switch f.Overload {
			case Numeric:
				visit("_hg" + f.Integer().GoName)
			case Scalar:
				for _, kind := range ScalarVariants {
					visit("_hg" + f.Variant(kind).GoName)
				}
			}
		}
		for _, t := range m.Types {
			visit("_hg" + t.Name)
			if t.Wrapper != "" {
				visit("_hg" + t.Wrapper)
			}
			if a := t.Arithmetic; a != nil {
				visit("_hg" + a.Add)
				visit("_hg" + a.Sub)
				visit("_hg" + a.Scale)
			}
		}
		for _, c := range m.Constants {
			if c.GoImport != "" {
				imports[strconv.Quote(c.GoImport)] = true
			}
		}
	}
	if imports[strconv.Quote("image")] {
		if !imports[strconv.Quote("image/png")] {
			imports["_ "+strconv.Quote("image/png")] = true
		}
		imports["_ "+strconv.Quote("image/jpeg")] = true
	}
	var out bytes.Buffer
	for _, d := range rt.decls {
		if selected[d] {
			if err := format.Node(&out, rt.fset, d); err != nil {
				return "", nil, err
			}
			out.WriteString("\n")
		}
	}
	source := out.String()
	// The atlases travel inside the program that draws with them.
	if strings.Contains(source, "DUNGEON_ATLAS_DATA") || strings.Contains(source, "ASCII_ATLAS_DATA") {
		source = strings.NewReplacer(
			"DUNGEON_ATLAS_DATA", base64.StdEncoding.EncodeToString(dungeonAtlas),
			"ASCII_ATLAS_DATA", base64.StdEncoding.EncodeToString(asciiAtlas),
		).Replace(source)
	}
	var paths []string
	for imported := range imports {
		paths = append(paths, imported)
	}
	sort.Strings(paths)
	return source, paths, nil
}
