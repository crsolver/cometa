package compiler

import (
	"strings"
	"testing"
)

// Messages for common beginner mistakes must say what was found and, where
// possible, what to write instead.
func TestFriendlyErrors(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	_, err := Compile("caso.cometa", []byte("fn f() entero 1\nfn inicio()\n\timprimir(zzzzzz)\n"))
	if err == nil || strings.Contains(err.Error(), "quisiste decir") {
		t.Errorf("no debe sugerir nombres distintos: %v", err)
	}
}

// A member written without '@' is the mistake to expect from someone coming
// from another language; the hint must reach every program, with or without
// imports, and through embedded types of this or another module.
func TestMissingReceiverMarkerHint(t *testing.T) {
	t.Parallel()
	hint := func(name string) string {
		return `el nombre "` + name + `" no existe; para usar un campo o método del objeto escribe '@` + name + `'`
	}
	_, err := Compile("receptor.cometa", []byte("tipo Base\n\tvida entero\n\tfn curar()\n\t\timprimir(1)\ntipo Heroe\n\tBase\n\tnombre cadena\n\tfn saludar()\n\t\timprimir(nombre)\n\t\timprimir(vida)\n\t\tcurar()\n\t\timprimir(otro)\n"))
	if err == nil {
		t.Fatal("se esperaban errores")
	}
	for _, want := range []string{"receptor.cometa:9:12: " + hint("nombre"), "receptor.cometa:10:12: " + hint("vida"), "receptor.cometa:11:3: " + hint("curar")} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("falta %q en:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "'@otro'") {
		t.Errorf("otro no es un miembro:\n%v", err)
	}

	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":  "usar std/mate\nusar seres\ntipo Heroe\n\tseres.Base\n\tfn saludar()\n\t\timprimir(vida)\n\t\timprimir(mate.pi)\nfn inicio()\n\tHeroe {}.saludar()\n",
		"seres.cometa": "pub tipo Base\n\tpub vida entero\n",
	})
	if _, err := CompileProject(entry, loader); err == nil || !strings.Contains(err.Error(), "main.cometa:6:12: "+hint("vida")) {
		t.Fatalf("falta la pista con un tipo embebido de otro módulo: %v", err)
	}
}

func TestDuplicateDeclarationsNameTheEarlierKind(t *testing.T) {
	t.Parallel()
	for source, want := range map[string]string{
		"var dato = 1\nfn dato() entero 2\n":             `:2:4: el nombre "dato" ya fue declarado como global`,
		"fn dato() entero 2\nvar dato = 1\n":             `:2:5: el nombre "dato" ya fue declarado como función`,
		"tipo Dato\n\tx entero\nfn Dato() entero 2\n":    `:3:4: el nombre "Dato" ya fue declarado como tipo`,
		"fn f()\n\timprimir(1)\nfn f()\n\timprimir(2)\n": `:3:4: la función "f" ya fue declarada`,
		"var a = 1\nvar a = 2\n":                         `:2:5: la global "a" ya fue declarada`,
		"tipo A\n\tx entero\nenum A\n\tB\n":              `:3:6: el tipo "A" ya fue declarado`,
	} {
		_, err := Compile("duplicado.cometa", []byte(source+"fn inicio()\n\timprimir(1)\n"))
		if err == nil || !strings.Contains(err.Error(), "duplicado.cometa"+want) {
			t.Errorf("%q: se esperaba %q, se obtuvo %v", source, want, err)
		}
	}
}
