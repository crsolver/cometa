package compiler

import (
	"strings"
	"testing"
)

// An interpolation writes any value the way imprimir does. The program never
// calls imprimir with a compound value, so the formatting runtime and the
// enum names must come from the interpolations alone.
func TestInterpolationOfCompoundValues(t *testing.T) {
	source := `enum Evento
	Cargar
	Texto cadena

tipo Punto
	x entero
	y entero

interfaz Nombrable
	fn nombre() cadena

tipo Gato
	pub fn nombre() cadena "gato"

fn mitad(n entero) entero!
	si n % 2 != 0
		retornar .Error("impar")
	n / 2

fn describir(notas [entero]) cadena
	"Notas: ${notas}"

fn inicio()
	var notas = [7, 9, 6]
	var linea = describir(notas)
	imprimir(linea)
	var nombres = ["Ana", "Luis"]
	var edades = ["Ana": 30]
	var evento Evento = .Texto("hola")
	var espera Evento = .Cargar
	var punto = Punto {x: 1, y: 2}
	var quizas entero? = 3
	var nada entero? = .Ninguno
	var gato = Gato {}
	var animal Nombrable = gato
	var texto = "${nombres} ${edades} ${evento} ${espera} ${punto}"
	imprimir(texto)
	var envueltos = "${quizas} ${nada} ${mitad(8)} ${mitad(7)} ${animal}"
	imprimir(envueltos)
	var mezcla = cadena([1.5, 2]) + cadena(punto) + "${verdadero} ${"sin comillas"} ${notas.longitud()}"
	imprimir(mezcla)
	var matriz = "${[[1], [2, 3]]}"
	imprimir(matriz)
`
	got, err := Compile("interpolar.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"Notas: [7, 9, 6]",
		`["Ana", "Luis"] ["Ana": 30] Evento.Texto("hola") Evento.Cargar Punto {x: 1, y: 2}`,
		`Alguno(3) Ninguno Ok(4) Error("impar") Gato {}`,
		"[1.5, 2]Punto {x: 1, y: 2}verdadero sin comillas 3",
		"[[1], [2, 3]]",
	}, "\n") + "\n"
	runGeneratedGo(t, got, want)

	_, err = Compile("vacio.cometa", []byte("fn nada()\n\tvar x = 1\n\nfn inicio()\n\tvar texto = \"${nada()}\"\n\timprimir(texto)\n"))
	if err == nil || !strings.Contains(err.Error(), "la interpolación requiere un valor") {
		t.Fatalf("error inesperado: %v", err)
	}
}
