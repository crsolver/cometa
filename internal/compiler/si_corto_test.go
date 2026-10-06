package compiler

import (
	"strings"
	"testing"
)

func TestShortConditionalSpacing(t *testing.T) {
	source := `enum Tono
	Claro
	Oscuro
	Gris

tipo Luz
	alta bool
	nivel entero

	fn tono() Tono si @alta .Claro osi @nivel > 0 .Gris sino .Oscuro

fn direccion(izquierda bool, derecha bool) entero
	si izquierda -1 osi derecha 1 sino 0

fn nombre(tono Tono) cadena
	casos tono
		.Claro => "claro"
		.Oscuro => "oscuro"
		.Gris => "gris"

fn doble(n entero) entero n * 2

fn inicio()
	var grande = verdadero
	var tono Tono = si grande .Claro sino .Oscuro
	imprimir(nombre(tono))
	var tenue = Luz {nivel: 2}
	imprimir(nombre(tenue.tono()))
	imprimir(direccion(verdadero, falso))
	imprimir(direccion(falso, verdadero))
	imprimir(direccion(falso, falso))
	var izquierda = falso
	var paso = si izquierda (-1) sino 1
	imprimir(paso)
	var vidas = 3
	imprimir(si vidas -1 > 1 "sigue" sino "fin")
	imprimir(si vidas - 3 == 0 -vidas sino vidas)
	imprimir(si doble(vidas -1) == 4 -1 sino 0)
	imprimir(si vidas == -3 -1 sino 1)
	var luz = Luz {alta: verdadero}
	imprimir(si luz.alta "alta" sino "baja")
	si vidas -1 > 1
		imprimir("bloque")
`
	got, err := Compile("corto.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{"claro", "gris", "-1", "1", "0", "1", "sigue", "-3", "-1", "1", "alta", "bloque"}, "\n") + "\n"
	runGeneratedGo(t, got, want)
}
