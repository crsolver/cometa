package stdlib

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

//go:embed runtime.txt
var runtimeSource string

//go:embed curvas_runtime.txt
var curvasRuntimeSource string

//go:embed ruido_runtime.txt
var ruidoRuntimeSource string

//go:embed pruebas_runtime.txt
var pruebasRuntimeSource string

//go:embed consola_runtime.txt
var consolaRuntimeSource string

//go:embed crt_runtime.txt
var crtRuntimeSource string

//go:embed extra_runtime.txt
var extraRuntimeSource string

//go:embed lienzo_runtime.txt
var lienzoRuntimeSource string

//go:embed retro_runtime.txt
var retroRuntimeSource string

//go:embed ui_core.txt
var uiCoreSource string

//go:embed ui_runtime.txt
var uiRuntimeSource string

//go:embed assets/dungeon-mode.png
var dungeonAtlas []byte

//go:embed assets/dungeon-437.png
var asciiAtlas []byte

var runtimeImports = map[string]string{
	"bufio": "bufio", "bytes": "bytes", "base64": "encoding/base64", "fmt": "fmt", "image": "image",
	"color": "image/color", "io": "io", "log": "log", "math": "math", "rand": "math/rand/v2", "strings": "strings",
	"ebiten": "github.com/hajimehoshi/ebiten/v2", "audio": "github.com/hajimehoshi/ebiten/v2/audio",
	"mp3": "github.com/hajimehoshi/ebiten/v2/audio/mp3", "vorbis": "github.com/hajimehoshi/ebiten/v2/audio/vorbis",
	"wav": "github.com/hajimehoshi/ebiten/v2/audio/wav", "ebitenutil": "github.com/hajimehoshi/ebiten/v2/ebitenutil",
	"inpututil": "github.com/hajimehoshi/ebiten/v2/inpututil", "text": "github.com/hajimehoshi/ebiten/v2/text/v2",
	"vector": "github.com/hajimehoshi/ebiten/v2/vector", "colorm": "github.com/hajimehoshi/ebiten/v2/colorm", "goregular": "golang.org/x/image/font/gofont/goregular",
	"strconv": "strconv", "os": "os", "png": "image/png", "json": "encoding/json", "filepath": "path/filepath", "sort": "sort", "unicode": "unicode", "time": "time",
}

// Runtime selects the declarations reachable from imported library exports.
// Go syntax supplies dependencies, so helpers and imports cannot drift apart.
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
	fs := token.NewFileSet()
	source := runtimeSource + "\n" + extraRuntimeSource + "\n" + lienzoRuntimeSource + "\n" + crtRuntimeSource + "\n" + curvasRuntimeSource + "\n" + ruidoRuntimeSource + "\n" + pruebasRuntimeSource + "\n" + consolaRuntimeSource
	for _, module := range modules {
		if module == "std/pincel/ui" {
			source += "\n" + uiCoreSource + "\n" + uiRuntimeSource
			break
		}
	}
	for _, module := range modules {
		if module == "std/pincel/retro" || module == "std/pincel/ui" {
			source += "\n" + strings.NewReplacer("DUNGEON_ATLAS_DATA", base64.StdEncoding.EncodeToString(dungeonAtlas), "ASCII_ATLAS_DATA", base64.StdEncoding.EncodeToString(asciiAtlas)).Replace(retroRuntimeSource)
			break
		}
	}
	file, err := parser.ParseFile(fs, "stdlib.go", "package main\n"+source, 0)
	if err != nil {
		return "", nil, err
	}
	byName := map[string][]ast.Decl{}
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
			byName[name] = append(byName[name], d)
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					byName[s.Name.Name] = append(byName[s.Name.Name], d)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						byName[n.Name] = append(byName[n.Name], d)
					}
				}
			}
		}
	}
	selected := map[ast.Decl]bool{}
	imports := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		for _, d := range byName[name] {
			if selected[d] {
				continue
			}
			selected[d] = true
			ast.Inspect(d, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					visit(id.Name)
				}
				if s, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := s.X.(*ast.Ident); ok {
						if path := runtimeImports[id.Name]; path != "" {
							imports[strconv.Quote(path)] = true
						}
					}
				}
				return true
			})
		}
	}
	for _, path := range modules {
		m, ok := LookupModule(path)
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
	for _, d := range file.Decls {
		if selected[d] {
			if err := format.Node(&out, fs, d); err != nil {
				return "", nil, err
			}
			out.WriteString("\n")
		}
	}
	var paths []string
	for path := range imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return out.String(), paths, nil
}
