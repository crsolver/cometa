package compiler

import (
	"strings"
	"testing"
)

func TestRepeatOverString(t *testing.T) {
	source := `fn vocales(texto cadena) entero
	var total = 0
	repetir (texto) |letra|
		si "aeiouáéíóú".contiene(letra)
			total += 1
	total

fn primera_mayuscula(texto cadena) cadena?
	repetir (texto) |letra|
		si letra != letra.minusculas()
			retornar letra
	.Ninguno

fn inicio()
	repetir ("año") |letra, indice|
		imprimir("${indice}: ${letra}")
	repetir ("") |letra|
		imprimir("nunca " + letra)
	repetir ("ab") |_|
		imprimir("vuelta")
	repetir ("abc") |letra, i|
		si i == 0 continuar
		si letra == "c" romper
		imprimir(letra)
	imprimir(vocales("murciélago"))
	imprimir(primera_mayuscula("hola Mundo") o "?")
	var invertido = ""
	repetir ("día".mayusculas()) |letra|
		invertido = letra + invertido
	imprimir(invertido)
`
	got, err := Compile("letras.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{"0: a", "1: ñ", "2: o", "vuelta", "vuelta", "b", "5", "M", "AÍD"}, "\n") + "\n"
	runGeneratedGo(t, got, want)
}
