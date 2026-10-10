package compiler

import (
	"strings"
	"testing"
)

func TestMientrasLoop(t *testing.T) {
	t.Parallel()
	source := `fn inicio()
	var i = 0
	mientras i < 3
		imprimir(i)
		i += 1
	var n = 0
	mientras verdadero
		n += 1
		si n < 3
			continuar
		si n == 5
			romper
		imprimir("n=${n}")
	var vacio = 5
	mientras vacio < 3
		imprimir("no debe imprimirse")
	var a = 0
	mientras a < 2
		var b = 0
		mientras b < 2
			imprimir("${a}${b}")
			b += 1
		a += 1
`
	got, err := Compile("mientras.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "0\n1\n2\nn=3\nn=4\n00\n01\n10\n11\n")
}

// mientras is lowered to an exit test with `!`; a wrong condition must be
// reported as the user wrote it, once.
func TestMientrasRequiresBooleanCondition(t *testing.T) {
	t.Parallel()
	_, err := Compile("mientras.cometa", []byte("fn inicio()\n\tvar veces = 3\n\tmientras veces\n\t\timprimir(veces)\n\tmientras falta\n\t\timprimir(1)\n"))
	if err == nil {
		t.Fatal("se esperaba un error")
	}
	text := err.Error()
	if !strings.Contains(text, "mientras.cometa:3:11: la condición de mientras debe ser bool, no entero") || strings.Contains(text, "operador") {
		t.Fatalf("mensaje inesperado:\n%s", text)
	}
	if strings.Count(text, "\n") != 1 || !strings.Contains(text, `mientras.cometa:5:11: el nombre "falta" no existe`) {
		t.Fatalf("se esperaban exactamente dos errores:\n%s", text)
	}
}
