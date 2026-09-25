package stdlib

import (
	"hacha/internal/ast"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Symbols are private linkage identities, never source-level names.
func Symbol(name string) string                    { return "__std_" + name }
func PublicName(name string) string                { return strings.TrimPrefix(name, "__std_") }
func FunctionSymbol(namespace, name string) string { return Symbol(namespace + "_" + name) }

var TypeModules = map[string]string{
	"Icono": "retro", "Atlas": "retro",
	"Vec2": "mate", "Rect": "mate", "Color": "color", "Camara2D": "graficos",
	"Imagen": "graficos", "Fuente": "graficos", "Tecla": "entrada", "BotonRaton": "entrada",
	"Sonido": "audio", "Reproduccion": "audio", "Juego": "juego",
}

var typeName = regexp.MustCompile(`\b(Vec2|Rect|Color|Camara2D|Imagen|Fuente|Tecla|BotonRaton|Sonido|Reproduccion|Juego|Icono|Atlas)\b`)

func QualifiedSignature(signature string) string {
	return typeName.ReplaceAllStringFunc(signature, func(name string) string { return TypeModules[name] + "." + name })
}

func Display(text string) string {
	// Longest first: function symbols can start with a type/namespace spelling.
	var names []string
	replacements := map[string]string{}
	for _, f := range Functions {
		name := FunctionSymbol(f.Namespace, f.Name)
		names = append(names, name)
		replacements[name] = f.Namespace + "." + f.Name
	}
	for name, owner := range TypeModules {
		symbol := Symbol(name)
		names = append(names, symbol)
		replacements[symbol] = owner + "." + name
	}
	names = append(names, FunctionSymbol("mate", "pi"))
	replacements[FunctionSymbol("mate", "pi")] = "mate.pi"
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		text = strings.ReplaceAll(text, name, replacements[name])
	}
	return text
}

// Relocate native signatures to the reference sources exposed to the editor.
func relocate(node any, namespace, declaration string) {
	if owner := TypeModules[PublicName(namespace)]; owner != "" {
		namespace = owner
	}
	path := ModulePath(namespace)
	source, ok := Source(path)
	if !ok {
		return
	}
	line, column := 0, 0
	for i, text := range strings.Split(source, "\n") {
		if at := strings.Index(text, declaration); at >= 0 {
			line, column = i+1, at
			break
		}
	}
	if line == 0 {
		return
	}
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if !v.IsNil() {
				walk(v.Elem())
			}
			return
		}
		if v.Kind() == reflect.Struct {
			if v.Type() == reflect.TypeOf(ast.Pos{}) && v.CanAddr() {
				pos := v.Addr().Interface().(*ast.Pos)
				if pos.Line > 0 {
					if pos.Line == 1 {
						pos.Column += column
					}
					pos.Line += line - 1
					pos.Filename = "hacha-std:///" + path + ".hacha"
				}
				return
			}
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		}
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(node))
}

func ModulePath(namespace string) string {
	if namespace == "curvas" || namespace == "ruido" {
		return "std/mate/" + namespace
	}
	if namespace == "mate" || namespace == "azar" {
		return "std/" + namespace
	}
	return "std/pincel/" + namespace
}

func Namespace(path string) (string, bool) {
	for _, f := range Functions {
		if ModulePath(f.Namespace) == path {
			return f.Namespace, true
		}
	}
	return "", false
}

func LookupSymbol(symbol string) (Function, bool) {
	for _, f := range Functions {
		if FunctionSymbol(f.Namespace, f.Name) == symbol {
			return f, true
		}
	}
	return Function{}, false
}

// BindTypes translates native declaration signatures to private type identities.
func BindTypes(node any) {
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return
			}
			if v.CanInterface() {
				if ref, ok := v.Interface().(*ast.TypeRef); ok {
					if TypeModules[ref.Name] != "" {
						ref.Name = Symbol(ref.Name)
					}
				}
			}
			walk(v.Elem())
			return
		}
		if v.Kind() == reflect.Struct {
			if v.Type() == reflect.TypeOf(ast.TypeRef{}) && v.CanAddr() {
				ref := v.Addr().Interface().(*ast.TypeRef)
				if TypeModules[ref.Name] != "" {
					ref.Name = Symbol(ref.Name)
				}
			}
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		}
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(node))
}

// Source exposes navigable declarations. Bodies are documentation stubs; the
// compiler links their private identities to the bundled native implementations.
func Source(path string) (string, bool) {
	ns, ok := Namespace(path)
	if !ok {
		return "", false
	}
	var b strings.Builder
	b.WriteString("// Biblioteca estándar de Hacha: " + path + "\n")
	if ns == "ruido" {
		b.WriteString("// Ruido 2D puro y reproducible por semilla; resultado en [0, 1].\n")
		b.WriteString("// Coordenadas finitas en (-2^52, 2^52), incluso en cada octava.\n")
		b.WriteString("// Octavas: 1..16; persistencia: [0, 1]; lacunaridad finita >= 1.\n")
	}
	if ns == "curvas" {
		b.WriteString("// Progreso limitado a [0, 1]; extremos exactos. El resultado puede sobrepasar [0, 1].\n")
		b.WriteString("// Funciones puras; el programa controla el tiempo y la interpolación.\n")
	}
	var names []string
	for name, owner := range TypeModules {
		if owner == ns {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if name == "Juego" {
			b.WriteString("interfaz Juego\n\tfn actualizar(dt decimal)\n\tfn pintar()\n")
			continue
		}
		b.WriteString("tipo " + name + "\n")
		if ns == "retro" {
			var constants []string
			for constant := range Constants[Symbol(name)] {
				constants = append(constants, "."+constant)
			}
			sort.Strings(constants)
			b.WriteString("// Valores: " + strings.Join(constants, ", ") + "\n")
		}
		for _, field := range strings.Split(Fields[Symbol(name)], "\n") {
			if field != "" {
				b.WriteString("\t" + field + "\n")
			}
		}
		for _, f := range Methods {
			if f.Namespace == Symbol(name) {
				b.WriteString("\tfn " + f.Name + "(" + f.Signature + "\n\t\timprimir(0)\n")
			}
		}
	}
	if ns == "mate" {
		b.WriteString("const pi decimal = 3.141592653589793\n")
	}
	for _, f := range Functions {
		if f.Namespace == ns {
			b.WriteString("fn " + f.Name + "(" + f.Signature + "\n\timprimir(0)\n")
		}
	}
	return b.String(), true
}
