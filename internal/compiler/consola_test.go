package compiler

import (
	"strings"
	"testing"
)

const consolaSuma = `usar std/consola

fn inicio()
	consola.escribir("¿Nombre? ")
	var nombre = consola.leer_linea() o "nadie"
	imprimir("Hola, ${nombre}")
	var suma = 0
	mientras verdadero
		si consola.leer_linea() |linea|
			suma += linea.recortar().a_entero() o 0
		sino
			romper
	imprimir(suma)
	imprimir(consola.leer_linea())
`

func TestConsolaLeeLineas(t *testing.T) {
	t.Parallel()
	generated, err := Compile("consola.cometa", []byte(consolaSuma))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("std/consola no debe depender de Ebitengine")
	}
	// A leading BOM, CRLF endings, a blank line, and a last line without a newline.
	runGeneratedGoWithInput(t, generated, "\ufeffAna María\r\n10\n\n 20 \n12", "¿Nombre? Hola, Ana María\n42\nNinguno\n")
}

func TestConsolaSinEntrada(t *testing.T) {
	t.Parallel()
	generated, err := Compile("consola.cometa", []byte(consolaSuma))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGoWithInput(t, generated, "", "¿Nombre? Hola, nadie\n0\nNinguno\n")
}

func TestConsolaRequiereImportar(t *testing.T) {
	t.Parallel()
	if _, err := Compile("consola.cometa", []byte("fn inicio()\n\timprimir(consola.leer_linea())\n")); err == nil {
		t.Fatal("se esperaba un error sin usar std/consola")
	}
}
