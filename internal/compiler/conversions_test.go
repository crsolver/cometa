package compiler

import (
	"strings"
	"testing"
)

func TestTextNumberConversions(t *testing.T) {
	source := `fn leer(texto cadena) cadena
	var n = texto.a_entero() o -1
	"${n}"

fn inicio()
	imprimir("42".a_entero())
	imprimir("-7".a_entero())
	imprimir("abc".a_entero())
	imprimir("".a_entero())
	imprimir(" 5".a_entero())
	imprimir("3.5".a_entero())
	imprimir("3.5".a_decimal())
	imprimir("10".a_decimal())
	imprimir("x".a_decimal())
	imprimir("inf".a_decimal())
	imprimir("nan".a_decimal())
	imprimir("1e400".a_decimal())
	imprimir("9223372036854775808".a_entero())
	imprimir(leer("12"))
	imprimir(leer("doce"))
	si "8".a_entero() |valor|
		imprimir(valor * 2)
	imprimir("Puntos: " + cadena(15))
	imprimir("Pi: " + cadena(3.5))
	imprimir("Bien: " + cadena(verdadero))
	imprimir(cadena("ya es texto"))
	imprimir(3.14159.formato(2))
	imprimir(2.5.formato(0))
	var siete = 7
	imprimir(siete.formato(2))
	var puntaje = 12.5
	imprimir("Puntaje ${puntaje.formato(3)}")
	imprimir(1.999.formato(1))
	imprimir(0.1.formato(25).longitud())
`
	got, err := Compile("conv.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"Alguno(42)", "Alguno(-7)", "Ninguno", "Ninguno", "Ninguno", "Ninguno",
		"Alguno(3.5)", "Alguno(10)", "Ninguno", "Ninguno", "Ninguno", "Ninguno", "Ninguno",
		"12", "-1", "16",
		"Puntos: 15", "Pi: 3.5", "Bien: verdadero", "ya es texto",
		"3.14", "2", "7.00", "Puntaje 12.500", "2.0", "22",
	}, "\n") + "\n"
	runGeneratedGo(t, got, want)
}

func TestConversionErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"a_entero args", "fn inicio()\n\timprimir(\"1\".a_entero(2))\n", "argumento"},
		{"formato needs arg", "fn inicio()\n\timprimir(1.5.formato())\n", "decimales"},
		{"formato typo", "fn inicio()\n\timprimir(1.5.formatos(1))\n", `¿quisiste decir "formato"?`},
		{"formato decimal arg", "fn inicio()\n\timprimir(1.5.formato(1.5))\n", "entero"},
		{"cadena of list", "fn inicio()\n\timprimir(cadena([1]))\n", "requiere"},
		{"cadena bare", "fn inicio()\n\tvar x = cadena\n", "se esperaba una expresión"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("se esperaba %q, se obtuvo %v", tc.want, err)
			}
		})
	}
}
