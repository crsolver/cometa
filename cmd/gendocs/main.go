// Command gendocs writes the standard library reference as JSON for the
// website (webpage/src/data/biblioteca.json), straight from internal/stdlib.
package main

import (
	"github.com/crsolver/cometa/internal/stdlib"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Func struct {
	Nombre  string `json:"nombre"`
	Firma   string `json:"firma"`
	Doc     string `json:"doc,omitempty"`
	Dibujo  bool   `json:"dibujo,omitempty"`
	Recurso bool   `json:"recurso,omitempty"`
}

type Type struct {
	Nombre     string   `json:"nombre"`
	Doc        string   `json:"doc,omitempty"`
	Campos     []string `json:"campos,omitempty"`
	Constantes []string `json:"constantes,omitempty"`
	Metodos    []Func   `json:"metodos,omitempty"`
	Interfaz   bool     `json:"interfaz,omitempty"`
}

type Module struct {
	Nombre     string   `json:"nombre"`
	Ruta       string   `json:"ruta"`
	Doc        string   `json:"doc,omitempty"`
	Tipos      []Type   `json:"tipos"`
	Constantes []string `json:"constantes,omitempty"`
	Funciones  []Func   `json:"funciones"`
}

// Build assembles every importable module from the stdlib catalog.
func Build() []Module {
	byNamespace := map[string]*Module{}
	var names []string
	for _, f := range stdlib.Functions {
		if byNamespace[f.Namespace] == nil {
			byNamespace[f.Namespace] = &Module{
				Nombre: f.Namespace,
				Ruta:   stdlib.ModulePath(f.Namespace),
				Doc:    stdlib.NamespaceDocs[f.Namespace],
			}
			names = append(names, f.Namespace)
		}
		m := byNamespace[f.Namespace]
		m.Funciones = append(m.Funciones, Func{
			Nombre:  f.Name,
			Firma:   f.Name + "(" + stdlib.QualifiedSignature(f.Signature),
			Doc:     stdlib.FunctionDocs[f.Namespace+"."+f.Name],
			Dibujo:  f.Draw,
			Recurso: f.Resource,
		})
	}
	sort.Strings(names)

	var typeNames []string
	for name := range stdlib.TypeModules {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)
	for _, name := range typeNames {
		m := byNamespace[stdlib.TypeModules[name]]
		if m == nil {
			continue
		}
		t := Type{Nombre: name, Doc: stdlib.TypeDocs[name], Interfaz: name == "Juego"}
		if name == "Juego" {
			t.Campos = []string{"fn actualizar(dt decimal)", "fn pintar()"}
		}
		for _, field := range strings.Split(stdlib.Fields[stdlib.Symbol(name)], "\n") {
			if field != "" {
				t.Campos = append(t.Campos, field)
			}
		}
		for c := range stdlib.Constants[stdlib.Symbol(name)] {
			t.Constantes = append(t.Constantes, "."+c)
		}
		sort.Strings(t.Constantes)
		for _, method := range stdlib.Methods {
			if method.Namespace == stdlib.Symbol(name) {
				t.Metodos = append(t.Metodos, Func{
					Nombre: method.Name,
					Firma:  method.Name + "(" + stdlib.QualifiedSignature(method.Signature),
					Doc:    stdlib.MethodDocs[name+"."+method.Name],
				})
			}
		}
		m.Tipos = append(m.Tipos, t)
	}
	if m := byNamespace["mate"]; m != nil {
		m.Constantes = append(m.Constantes, "pi decimal = 3.141592653589793")
	}

	modules := make([]Module, 0, len(names))
	for _, n := range names {
		if byNamespace[n].Tipos == nil {
			byNamespace[n].Tipos = []Type{}
		}
		modules = append(modules, *byNamespace[n])
	}
	return modules
}

func main() {
	out := filepath.Join("webpage", "src", "data", "biblioteca.json")
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	data, err := json.MarshalIndent(Build(), "", "\t")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("escrito", out)
}
