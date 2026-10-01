package compiler

import (
	"strings"
	"testing"
)

func TestPruebasAssertions(t *testing.T) {
	generated, err := Compile("p.cometa", []byte("usar std/pruebas\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("pruebas must not pull in Ebitengine")
	}
	runCometa(t, `usar std/pruebas
fn inicio()
	pruebas.afirmar(verdadero)
	pruebas.igual(1, 1)
	pruebas.igual(1, 1.0)
	pruebas.igual("a", "a", "cadenas")
	pruebas.igual(verdadero, verdadero)
	pruebas.casi_igual(0.1 + 0.2, 0.3)
	imprimir("ok")
`, "ok\n")
	visual, err := Compile("v.cometa", []byte("usar std/pruebas\nusar std/pincel\nfn paso(j pincel.Juego)\n\tpruebas.avanzar(j, 3)\n\tpruebas.pixel(0, 0, .Rojo, tolerancia = 2)\n\timprimir(pruebas.pantalla())\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(visual), "func _hgpruebasBucle(") {
		t.Fatal("visual tests need the hidden game loop")
	}
	for _, source := range []string{
		"usar std/pruebas\nfn inicio() pruebas.igual(1, \"a\")\n",
		"usar std/pruebas\nfn inicio() pruebas.igual(1)\n",
		"usar std/pruebas\nfn inicio() pruebas.afirmar(1)\n",
		"usar std/pincel/pruebas\n",
		// Visual assertions need a Pincel game somewhere in the project.
		"usar std/pruebas\nfn inicio() pruebas.pixel(0, 0, .Rojo)\n",
	} {
		if _, err := Compile("invalid.cometa", []byte(source)); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
}
