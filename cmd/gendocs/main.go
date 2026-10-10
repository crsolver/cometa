// Command gendocs writes the standard library reference as JSON for the
// website (webpage/src/data/biblioteca.json), straight from internal/stdlib.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/crsolver/cometa/internal/stdlib"
	"os"
	"path/filepath"
	"sort"
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

// Build assembles every importable module from the stdlib catalog, sorted by
// name, with the types of each module sorted too.
func Build() []Module {
	function := func(f stdlib.Function) Func {
		return Func{
			Nombre:  f.Name,
			Firma:   f.Name + stdlib.QualifiedSignature(f.Signature),
			Doc:     f.Doc,
			Dibujo:  f.Draw,
			Recurso: f.Resource,
		}
	}
	modules := make([]Module, 0, len(stdlib.Modules))
	for _, m := range stdlib.Modules {
		module := Module{Nombre: m.Name, Ruta: m.Path, Doc: m.Doc, Tipos: []Type{}}
		for _, t := range m.Types {
			out := Type{Nombre: t.Name, Doc: t.Doc, Interfaz: t.Interface, Campos: append([]string(nil), t.Fields...)}
			for c := range t.Constants {
				out.Constantes = append(out.Constantes, "."+c)
			}
			sort.Strings(out.Constantes)
			for _, method := range t.Methods {
				if t.Interface {
					// What a program must write is listed with the fields.
					out.Campos = append(out.Campos, "fn "+method.Name+method.Signature)
					continue
				}
				out.Metodos = append(out.Metodos, function(method))
			}
			module.Tipos = append(module.Tipos, out)
		}
		sort.Slice(module.Tipos, func(i, j int) bool { return module.Tipos[i].Nombre < module.Tipos[j].Nombre })
		for _, c := range m.Constants {
			module.Constantes = append(module.Constantes, c.Name+" "+c.Type+" = "+c.Value)
		}
		for _, f := range m.Functions {
			module.Funciones = append(module.Funciones, function(f))
		}
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Nombre < modules[j].Nombre })
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
