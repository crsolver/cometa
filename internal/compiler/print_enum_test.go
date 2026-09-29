package compiler

import "testing"

// imprimir must spell enum variants and results the Cometa way, not as Go structs.
func TestPrintEnumsAndResults(t *testing.T) {
	source := `enum Evento
	Cargar
	Texto cadena
	Num entero

fn dividir(a entero, b entero) entero!
	si b == 0
		.Error("cero")
	sino
		a / b

fn bien() !
	.Ok

fn inicio()
	imprimir(Evento.Cargar)
	imprimir(Evento.Texto("hola"))
	imprimir([Evento.Num(2), Evento.Cargar])
	var opc Evento? = .Alguno(.Num(5))
	imprimir(opc)
	imprimir(dividir(4, 2))
	imprimir(dividir(1, 0))
	imprimir(bien())
`
	want := "Evento.Cargar\nEvento.Texto(\"hola\")\n[Evento.Num(2), Evento.Cargar]\nAlguno(Evento.Num(5))\nOk(2)\nError(\"cero\")\nOk\n"
	got, err := Compile("enums.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, want)
}
