package codegen

import (
	"fmt"
	"sort"
	"unicode"

	"github.com/crsolver/cometa/internal/sema"
)

// goNames holds the Go identifier of every top-level function and global.
//
// Cometa names are case-sensitive and every top-level name is capitalized in
// Go, so `fn punto` and `tipo Punto` would collide. Types, enums and
// interfaces always keep the plain capitalized spelling, because imprimir
// shows it; functions and globals move aside when it is taken.
type goNames struct {
	functions map[string]string
	globals   map[string]string
}

func newGoNames(model *sema.Model) *goNames {
	names := &goNames{functions: map[string]string{}, globals: map[string]string{}}
	occupied := map[string]bool{}
	for name := range model.Types {
		occupied[exported(name)] = true
	}
	for name := range model.Enums {
		occupied[exported(name)] = true
	}
	for name := range model.Interfaces {
		occupied[exported(name)] = true
	}
	// Sorted, so the choice never depends on map iteration order.
	for _, name := range sortedNames(model.Functions) {
		candidate := exported(name)
		if name == "inicio" {
			candidate = "main"
		}
		for occupied[candidate] {
			candidate += "_fn"
		}
		occupied[candidate] = true
		names.functions[name] = candidate
	}
	for _, name := range sortedNames(model.Globals) {
		candidate := exported(name)
		if occupied[candidate] {
			// Keeps the common jugador/Jugador pair of a global and its type.
			candidate = fmt.Sprintf("CometaGlobal_%x", []byte(name))
		}
		for occupied[candidate] {
			candidate += "_"
		}
		occupied[candidate] = true
		names.globals[name] = candidate
	}
	return names
}

func sortedNames[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// FunctionName returns the Go identifier of a top-level Cometa function, for
// tools that call generated functions from Go code of their own.
func FunctionName(model *sema.Model, name string) string {
	return newGoNames(model).function(name)
}

func (n *goNames) function(name string) string {
	if assigned, ok := n.functions[name]; ok {
		return assigned
	}
	return exported(name)
}

func (g *generator) funcName(name string) string { return g.names.function(name) }

func (g *generator) globalName(name string) string {
	if assigned, ok := g.names.globals[name]; ok {
		return assigned
	}
	return exported(name)
}

var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}

// goBodyNames are the predeclared identifiers and package names that code
// lowered into a function body refers to. A Cometa local with one of these
// names would shadow it. Other predeclared names (error, min, new...) are
// never emitted, so shadowing them is harmless and they keep their spelling.
// TestLocalsNamedLikeGoIdentifiers exercises every entry.
var goBodyNames = map[string]bool{
	"any": true, "append": true, "bool": true, "clear": true, "copy": true,
	"delete": true, "false": true, "float64": true, "int": true, "int64": true,
	"len": true, "make": true, "nil": true, "panic": true, "rune": true,
	"string": true, "true": true,
	"ebiten": true, "fmt": true, "math": true, "reflect": true, "sort": true,
	"strconv": true, "strings": true,
}

func exported(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return name
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func localName(name string) string {
	if goKeywords[name] || goBodyNames[name] || name == "_self" {
		return "h_" + name
	}
	return name
}
