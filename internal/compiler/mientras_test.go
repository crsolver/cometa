package compiler

import "testing"

func TestMientrasLoop(t *testing.T) {
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
