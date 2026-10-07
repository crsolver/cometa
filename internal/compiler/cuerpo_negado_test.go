package compiler

import "testing"

func TestOneLineBodyStartingWithNegation(t *testing.T) {
	t.Parallel()
	source := `tipo Pieza
	y entero

	fn cabe(y entero) bool y < 3
	fn en_suelo() bool !@cabe(@y + 1)
	fn libre() bool? !@en_suelo()

fn impar(n entero) bool !(n % 2 == 0)

fn mitad(n entero) entero!
	si n % 2 != 0
		retornar .Error("impar")
	n / 2

fn revisar(n entero) !
	si n < 0
		retornar .Error("negativo")
	.Ok

fn inicio()
	var pieza = Pieza {y: 2}
	imprimir(pieza.en_suelo())
	imprimir(pieza.libre() o verdadero)
	imprimir(impar(3))
	imprimir(mitad(8) o -1)
	imprimir(mitad(7) o -1)
	revisar(-1) atrapar |error|
		imprimir(error)
`
	got, err := Compile("negado.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "verdadero\nfalso\nverdadero\n4\n-1\nnegativo\n")
}
