package diagnostic

// foreign maps words from other languages to their Cometa equivalent, so a
// newcomer's first guess gets a useful hint instead of a bare "unknown name".
var foreign = map[string]string{
	"if":        "en Cometa se escribe 'si'",
	"else":      "en Cometa se escribe 'sino'",
	"elif":      "en Cometa se escribe 'osi'",
	"elsif":     "en Cometa se escribe 'osi'",
	"elseif":    "en Cometa se escribe 'osi'",
	"while":     "en Cometa se escribe 'mientras'",
	"for":       "para recorrer usa 'repetir (lista) |elemento|' o 'repetir (0..n) |i|'",
	"para":      "para recorrer usa 'repetir (lista) |elemento|' o 'repetir (0..n) |i|'",
	"foreach":   "para recorrer usa 'repetir (lista) |elemento|'",
	"return":    "en Cometa se escribe 'retornar'",
	"break":     "en Cometa se escribe 'romper'",
	"continue":  "en Cometa se escribe 'continuar'",
	"true":      "en Cometa se escribe 'verdadero'",
	"false":     "en Cometa se escribe 'falso'",
	"null":      "Cometa no tiene null; usa un opcional ('T?') con '.Ninguno'",
	"nil":       "Cometa no tiene nil; usa un opcional ('T?') con '.Ninguno'",
	"none":      "Cometa no tiene None; usa un opcional ('T?') con '.Ninguno'",
	"function":  "en Cometa las funciones se declaran con 'fn'",
	"func":      "en Cometa las funciones se declaran con 'fn'",
	"def":       "en Cometa las funciones se declaran con 'fn'",
	"let":       "en Cometa las variables se declaran con 'var'",
	"print":     "en Cometa se escribe 'imprimir'",
	"println":   "en Cometa se escribe 'imprimir'",
	"console":   "en Cometa se escribe 'imprimir(valor)'",
	"int":       "en Cometa el tipo se llama 'entero'",
	"integer":   "en Cometa el tipo se llama 'entero'",
	"float":     "en Cometa el tipo se llama 'decimal'",
	"double":    "en Cometa el tipo se llama 'decimal'",
	"string":    "en Cometa el tipo se llama 'cadena'",
	"str":       "en Cometa el tipo se llama 'cadena'",
	"boolean":   "en Cometa el tipo se llama 'bool'",
	"struct":    "en Cometa las estructuras se declaran con 'tipo'",
	"class":     "en Cometa las estructuras se declaran con 'tipo'",
	"switch":    "en Cometa se escribe 'casos'",
	"match":     "en Cometa se escribe 'casos'",
	"import":    "en Cometa se escribe 'usar'",
	"and":       "en Cometa el «y» lógico es '&&'",
	"not":       "en Cometa la negación es '!'",
	"self":      "dentro de un método, los campos se acceden con '@'",
	"this":      "dentro de un método, los campos se acceden con '@'",
	"undefined": "Cometa no tiene undefined; usa un opcional ('T?') con '.Ninguno'",
}

// Foreign returns a hint when name is a keyword from another language.
func Foreign(name string) string {
	return foreign[name]
}
