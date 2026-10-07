package compiler

import "testing"

func TestMemberAccessOnStructLiteral(t *testing.T) {
	t.Parallel()
	source := `tipo Luz
	nivel entero

	fn tono(extra entero = 0) entero @nivel * 2 + extra
	fn subir()
		@nivel += 1

tipo Caja<T>
	valor T

	fn obtener() T @valor

tipo Foco
	Luz
	nombre cadena

fn inicio()
	imprimir(Luz {nivel: 2}.tono())
	imprimir(Luz {nivel: 2}.tono(extra = 1))
	imprimir(Luz {nivel: 3}.nivel)
	var total = Luz {nivel: 4}.tono() + Luz {}.nivel
	imprimir(total)
	imprimir(Luz {nivel: 5}.copiar().nivel)
	imprimir(Caja<entero> {valor: 7}.obtener())
	imprimir(Foco {Luz: {nivel: 6}, nombre: "sala"}.tono())
	Luz {nivel: 1}.subir()
	si Luz {nivel: 1}.tono() == 2
		imprimir("dos")
`
	got, err := Compile("literal.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "4\n5\n3\n8\n5\n7\n12\ndos\n")
}
