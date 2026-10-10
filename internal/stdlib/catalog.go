// Package stdlib is the compiler-owned catalog of the standard library: the
// modules a program can import, with their types, constants and functions, and
// the Go runtime that implements them.
package stdlib

import (
	"regexp"
	"sort"
	"strings"
)

const EbitenVersion = "v2.10.1"

// Module is one importable part of the library. Each one is declared in its
// mod_*.go file and listed in Modules; the type checker, the code generator,
// the editor and the reference all read it from there, so adding a function
// or a type means adding one entry (and its Go implementation to the runtime).
type Module struct {
	Name string // the namespace programs write: graficos.limpiar
	Path string // what `usar` imports: std/pincel/graficos
	Doc  string
	// Notes are extra lines of the reference header, such as limits and ranges.
	Notes     []string
	Types     []Type
	Constants []Constant
	Functions []Function
	// OnlyCalled makes the runtime carry just the functions a program calls
	// instead of the whole module. std/pincel/tiempo needs it: its clock and
	// timers work without a window, but fps and tps bring in Ebitengine.
	OnlyCalled bool

	phased bool // some function or type has a Phase
}

// Type is a library type. Its Go type is "_hg" followed by its name.
type Type struct {
	Name string
	Doc  string
	// Fields holds one "nombre tipo" per public field, with "= valor" after
	// it when a literal that omits the field should not get the zero value. A
	// type without fields is opaque: programs get its values from the
	// library, never from a literal.
	Fields []string
	// Value types are copied on assignment, like numbers. The rest are
	// references, like the types a program declares.
	Value bool
	// Interface types are implemented by programs; Methods are what they require.
	Interface bool
	// Constants maps each `.Nombre` of the type to its Go expression.
	Constants map[string]string
	Methods   []Function
	// Arithmetic gives the type the operators of a vector.
	Arithmetic *Arithmetic
	// Channels are members that can be read but never assigned, and that do
	// not make the type a structure: it stays opaque.
	Channels []Channel
	// ReadOnly is the error for an assignment to a channel.
	ReadOnly string
	// Wrapper is the Go type that carries Methods when the type itself cannot:
	// an alias of a type of another Go package has no methods of its own.
	Wrapper string
	// Phase names the only game callback where a value of the type may open a
	// `con` block, when it has that restriction.
	Phase string

	module *Module
}

// Arithmetic names the runtime functions behind the operators of a vector
// type, without their "_hg" prefix: a + b, a - b, and a * n, n * a, a / n and
// -a, which all scale.
type Arithmetic struct{ Add, Sub, Scale string }

// Channel is a read-only member of a type, held in a field of its Go type.
// Channels are integers.
type Channel struct{ Name, GoField string }

// Overload says how a function accepts more than the types of its signature.
type Overload int

const (
	// Numeric functions are written for decimals and also work on integers:
	// when every argument is an integer, so are the parameters and the result.
	// The runtime holds that variant as GoName followed by "Entero".
	Numeric Overload = iota + 1
	// Scalar functions take, as their first two parameters, two values of the
	// same built-in type. The runtime holds one variant for each: GoName
	// followed by "Entero", "Decimal", "Cadena" or "Bool".
	Scalar
)

// Constant is a named value of a module, such as mate.pi.
type Constant struct {
	Name     string
	Type     string // a built-in type
	Value    string // as written in Cometa, for the reference
	GoValue  string
	GoImport string // the Go package GoValue needs, if any

	module *Module
}

// Function is a function of a module or a method of a library type.
type Function struct {
	// Namespace is the module name of a function and the symbol of the
	// receiver type of a method. The catalog fills it in.
	Namespace string
	Name      string
	// Signature is the parameter list and result in Cometa syntax:
	// "(x decimal, y decimal = 0) decimal".
	Signature string
	// GoName is the runtime function without its "_hg" prefix, or the Go
	// method of the receiver.
	GoName string
	// Draw functions paint, so they are only valid while the game paints.
	Draw bool
	// Resource functions embed a file: they take one literal path and are
	// only valid as the initializer of a global.
	Resource bool
	// Visual functions step or paint a game, so they need Ebitengine even
	// when the rest of their module does not.
	Visual bool
	// Phase names the only game callback the function may run from
	// ("actualizar" or "pintar"), when it has that restriction.
	Phase    string
	Overload Overload
	Doc      string

	module *Module
	owner  *Type // the receiver type of a method
}

// Phased reports whether the module of f restricts anything to one game
// callback, so a program that calls f needs its callbacks checked.
func (f Function) Phased() bool { return f.module != nil && f.module.phased }

// Integer is the variant of a Numeric function that works on integers.
func (f Function) Integer() Function {
	f.GoName += "Entero"
	f.Signature = strings.ReplaceAll(f.Signature, "decimal", "entero")
	return f
}

// Variant is the variant of a Scalar function for a built-in type, named as
// the runtime does: "Entero", "Decimal", "Cadena" or "Bool".
func (f Function) Variant(kind string) Function {
	f.GoName += kind
	return f
}

// ScalarVariants are the suffixes of the variants of a Scalar function.
var ScalarVariants = []string{"Entero", "Decimal", "Cadena", "Bool"}

// Modules lists every module of the library.
var Modules = []*Module{
	&azarModule, &consolaModule, &mateModule, &curvasModule, &ruidoModule, &pincelModule, &audioModule,
	&colorModule, &datosModule, &entradaModule, &graficosModule, &lienzoModule, &recursosModule,
	&rejillaModule, &retroModule, &tiempoModule, &uiModule, &ventanaModule, &pruebasModule,
}

// Functions lists the functions of every module, in catalog order.
var Functions []Function

// Constants maps the symbol of each type that has constants to them.
var Constants = map[string]map[string]string{}

// The lookups below are filled once, from Modules.
var (
	modulesByPath   = map[string]*Module{}
	modulesByName   = map[string]*Module{}
	typesBySymbol   = map[string]*Type{}
	functionsByName = map[string]*Function{} // by symbol
	constantsByName = map[string]*Constant{} // by symbol
	typeName        *regexp.Regexp
	display         *strings.Replacer
)

func init() {
	var types []string
	// Longest first: a function symbol can start with the spelling of a type.
	var symbols []string
	shown := map[string]string{}
	for _, m := range Modules {
		modulesByPath[m.Path] = m
		modulesByName[m.Name] = m
		for i := range m.Types {
			t := &m.Types[i]
			t.module = m
			m.phased = m.phased || t.Phase != ""
			typesBySymbol[Symbol(t.Name)] = t
			types = append(types, t.Name)
			symbols = append(symbols, Symbol(t.Name))
			shown[Symbol(t.Name)] = m.Name + "." + t.Name
			if len(t.Constants) > 0 {
				Constants[Symbol(t.Name)] = t.Constants
			}
			for j := range t.Methods {
				method := &t.Methods[j]
				method.Namespace, method.module, method.owner = Symbol(t.Name), m, t
			}
		}
		for i := range m.Constants {
			c := &m.Constants[i]
			c.module = m
			symbol := FunctionSymbol(m.Name, c.Name)
			constantsByName[symbol] = c
			symbols = append(symbols, symbol)
			shown[symbol] = m.Name + "." + c.Name
		}
		for i := range m.Functions {
			f := &m.Functions[i]
			f.Namespace, f.module = m.Name, m
			m.phased = m.phased || f.Phase != ""
			symbol := FunctionSymbol(m.Name, f.Name)
			functionsByName[symbol] = f
			symbols = append(symbols, symbol)
			shown[symbol] = m.Name + "." + f.Name
		}
	}
	for _, m := range Modules {
		Functions = append(Functions, m.Functions...)
	}
	typeName = regexp.MustCompile(`\b(` + strings.Join(types, "|") + `)\b`)
	sort.SliceStable(symbols, func(i, j int) bool { return len(symbols[i]) > len(symbols[j]) })
	var pairs []string
	for _, symbol := range symbols {
		pairs = append(pairs, symbol, shown[symbol])
	}
	display = strings.NewReplacer(pairs...)
}

// Symbols are private linkage identities, never source-level names.
func Symbol(name string) string                    { return "__std_" + name }
func PublicName(name string) string                { return strings.TrimPrefix(name, "__std_") }
func FunctionSymbol(namespace, name string) string { return Symbol(namespace + "_" + name) }

// Reserved reports whether a program may not declare name.
func Reserved(name string) bool {
	return name == "entero" || name == "decimal" || strings.HasPrefix(name, "_hg") || strings.HasPrefix(name, "__std_")
}

// LookupModule returns the module imported as path ("std/mate").
func LookupModule(path string) (*Module, bool) {
	m, ok := modulesByPath[path]
	return m, ok
}

// ModulePaths lists every importable "std/..." path, for editor completion.
func ModulePaths() []string {
	var paths []string
	for _, m := range Modules {
		paths = append(paths, m.Path)
	}
	sort.Strings(paths)
	return paths
}

// LookupSymbol returns the function a symbol names.
func LookupSymbol(symbol string) (Function, bool) {
	if f := functionsByName[symbol]; f != nil {
		return *f, true
	}
	return Function{}, false
}

// LookupConstant returns the module constant a symbol names, such as mate.pi.
func LookupConstant(symbol string) (Constant, bool) {
	if c := constantsByName[symbol]; c != nil {
		return *c, true
	}
	return Constant{}, false
}

// LookupType returns the library type a symbol names, interfaces included.
func LookupType(symbol string) (*Type, bool) {
	t, ok := typesBySymbol[symbol]
	return t, ok
}

// IsType reports whether symbol names a concrete library type. The interfaces
// programs implement (pincel.Juego) are not included.
func IsType(symbol string) bool {
	t := typesBySymbol[symbol]
	return t != nil && !t.Interface
}

// IsValue reports whether symbol names a library type that is copied on assignment.
func IsValue(symbol string) bool {
	t := typesBySymbol[symbol]
	return t != nil && t.Value
}

// IsOpaque reports whether symbol names a library type that has no fields, so
// only the library can make its values.
func IsOpaque(symbol string) bool {
	t := typesBySymbol[symbol]
	return t != nil && !t.Interface && len(t.Fields) == 0
}

// GoType returns the Go type of a concrete library type, or "". Native Go
// names are deliberately outside the exported user-name space.
func GoType(symbol string) string {
	if IsType(symbol) {
		return "_hg" + PublicName(symbol)
	}
	return ""
}

// GoInterface returns the Go type of a library interface, or "".
func GoInterface(symbol string) string {
	if t := typesBySymbol[symbol]; t != nil && t.Interface {
		return "_hg" + t.Name
	}
	return ""
}

// Vector returns the arithmetic of a library type that has operators, or nil.
func Vector(symbol string) *Arithmetic {
	if t := typesBySymbol[symbol]; t != nil {
		return t.Arithmetic
	}
	return nil
}

// Channel returns the Go field behind a read-only member of a library type,
// or "" when the type has no such member.
func (t *Type) Channel(name string) string {
	for _, channel := range t.Channels {
		if channel.Name == name {
			return channel.GoField
		}
	}
	return ""
}

// Symbol is the identity of the type in a checked program.
func (t *Type) Symbol() string { return Symbol(t.Name) }

// Module is the module that declares the type.
func (t *Type) Module() *Module { return t.module }

// UsesPincel reports whether any imported native module belongs to Pincel.
func UsesPincel(modules []string) bool {
	for _, module := range modules {
		if module == "std/pincel" || strings.HasPrefix(module, "std/pincel/") {
			return true
		}
	}
	return false
}

// QualifiedSignature writes the library types of a signature with their
// module: "(pos Vec2)" becomes "(pos mate.Vec2)".
func QualifiedSignature(signature string) string {
	return typeName.ReplaceAllStringFunc(signature, func(name string) string {
		return typesBySymbol[Symbol(name)].module.Name + "." + name
	})
}

// Display replaces the symbols in a message with the names programs write.
func Display(text string) string { return display.Replace(text) }
