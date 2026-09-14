package compiler

import "testing"

func TestRangeRuntime(t *testing.T) {
	runHacha(t, `enum E
	A
tipo Contador
	n num
fn limite(c Contador, n num) num
	c.n = c.n + 1
	imprimir(n)
	n
fn inicio()
	repetir (0..5) |i| imprimir(i)
	repetir (3..0) |i, indice| imprimir(i + indice)
	repetir (2..2) |i| imprimir("incorrecto")
	repetir (-2..1) |i| imprimir(i)
	repetir (0.5..2) |i| imprimir(i)
	repetir (2.5..0) |i| imprimir(i)
	var c = Contador {n: 0}
	var final = 3
	repetir (limite(c, 0)..limite(c, final)) |i|
		final = 0
		casos E.A
			_ =>
				si i == 0 continuar
				repetir (2..0) |j|
					casos E.A
						_ => romper
				imprimir(i)
				si i == 2 romper
	imprimir(c.n)
	var _hacha1 = 9
	repetir (0..2) |i|
		i = 100
		imprimir(i)
	imprimir(_hacha1)
`, "0\n1\n2\n3\n4\n3\n3\n3\n-2\n-1\n0\n0.5\n1.5\n2.5\n1.5\n0.5\n0\n3\n1\n2\n2\n100\n100\n9\n")
}

func TestRangesAreOnlyLoopSyntax(t *testing.T) {
	for _, body := range []string{
		"var rango = 0..5", "imprimir(0..5)", "var lista = [0..5]",
		"repetir (..5) |i| imprimir(i)", "repetir (0..) |i| imprimir(i)",
		"repetir (0..2..5) |i| imprimir(i)", "repetir (0...5) |i| imprimir(i)",
		"repetir (0..5) imprimir(0)",
	} {
		t.Run(body, func(t *testing.T) {
			if _, err := Compile("rango.hacha", []byte("fn inicio()\n\t"+body+"\n")); err == nil {
				t.Fatal("expected invalid range syntax to fail")
			}
		})
	}
}
