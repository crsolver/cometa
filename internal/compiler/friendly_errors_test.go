package compiler

import (
	"strings"
	"testing"
)

// Messages for common beginner mistakes must say what was found and, where
// possible, what to write instead.
func TestFriendlyErrors(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{"typo local", "fn inicio()\n\tvar edad = 3\n\timprimir(edda)\n", `¿quisiste decir "edad"?`},
		{"typo function", "fn saludar() imprimir(1)\nfn inicio()\n\tsaludr()\n", `¿quisiste decir "saludar"?`},
		{"typo field", "tipo P\n\tnombre cadena\nfn inicio()\n\tvar p = P {}\n\timprimir(p.nombr)\n", `¿quisiste decir "nombre"?`},
		{"typo method", "tipo P\n\tfn hola() imprimir(1)\nfn inicio()\n\tvar p = P {}\n\tp.hoal()\n", `¿quisiste decir "hola"?`},
		{"typo list method", "fn inicio()\n\tvar l = [1]\n\tl.agregr(2)\n", `¿quisiste decir "agregar"?`},
		{"typo string method", "fn inicio()\n\timprimir(\"a\".mayuscula())\n", `¿quisiste decir "mayusculas"?`},
		{"typo variant", "enum E\n\tUno\nfn inicio()\n\tvar e = E.Uon\n", `¿quisiste decir "Uno"?`},
		{"while", "fn inicio()\n\tvar x = 0\n\twhile x < 3\n\t\tx = x + 1\n", "se escribe 'mientras'"},
		{"if", "fn inicio()\n\tif verdadero\n\t\timprimir(1)\n", "se escribe 'si'"},
		{"print", "fn inicio()\n\tprint(1)\n", "se escribe 'imprimir'"},
		{"increment", "fn inicio()\n\tvar x = 0\n\tx++\n", "Cometa no tiene '++'"},
		{"found token", "fn inicio()\n\timprimir(1\n", "pero se encontró el final de la línea"},
		{"bitwise", "fn inicio()\n\tvar x = 1 & 2\n", "operadores de bits"},
		{"semicolon", "fn inicio()\n\tvar x = 1;\n", "cada instrucción va en su propia línea"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil {
				t.Fatal("se esperaba un error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("falta %q en:\n%s", tc.want, err)
			}
		})
	}
}

func TestNoSuggestionForUnrelatedNames(t *testing.T) {
	_, err := Compile("caso.cometa", []byte("fn f() entero 1\nfn inicio()\n\timprimir(zzzzzz)\n"))
	if err == nil || strings.Contains(err.Error(), "quisiste decir") {
		t.Errorf("no debe sugerir nombres distintos: %v", err)
	}
}
