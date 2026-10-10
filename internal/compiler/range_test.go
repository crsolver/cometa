package compiler

import "testing"

func TestRangeRuntime(t *testing.T) {
	t.Parallel()
	runCometa(t, `enum E
	A
tipo Contador
	n entero
fn limite(c Contador, n entero) entero
	c.n = c.n + 1
	imprimir(n)
	n
fn inicio()
	repetir (0..5) |i| imprimir(i)
	repetir (3..0, paso = -1) |i, indice| imprimir(i + indice)
	repetir (2..2) |i| imprimir("incorrecto")
	repetir (-2..1) |i| imprimir(i)
	repetir (0..2) |i| imprimir(i)
	repetir (3..0, paso = -1) |i| imprimir(i)
	var c = Contador {n: 0}
	var final = 3
	repetir (limite(c, 0)..limite(c, final)) |i|
		final = 0
		casos E.A
			_ =>
				si i == 0 continuar
				repetir (2..0, paso = -1) |j|
					casos E.A
						_ => romper
				imprimir(i)
				si i == 2 romper
	imprimir(c.n)
	var _cometa1 = 9
	repetir (0..2) |i|
		i = 100
		imprimir(i)
	imprimir(_cometa1)
`, "0\n1\n2\n3\n4\n3\n3\n3\n-2\n-1\n0\n0\n1\n3\n2\n1\n0\n3\n1\n2\n2\n100\n100\n9\n")
}

func TestRangesAreOnlyLoopSyntax(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"var rango = 0..5", "imprimir(0..5)", "var lista = [0..5]",
		"repetir (..5) |i| imprimir(i)", "repetir (0..) |i| imprimir(i)",
		"repetir (0..2..5) |i| imprimir(i)", "repetir (0...5) |i| imprimir(i)",
		"repetir (3..0) |i| imprimir(i)", "repetir (0..3, paso = 0) |i| imprimir(i)",
		"repetir (0..3, paso = 1.5) |i| imprimir(i)", "repetir (0..3, salto = 1) |i| imprimir(i)",
		"repetir (0..3, paso) |i| imprimir(i)", "repetir (3, paso = 1) |i| imprimir(i)",
	} {
		t.Run(body, func(t *testing.T) {
			if _, err := Compile("rango.cometa", []byte("fn inicio()\n\t"+body+"\n")); err == nil {
				t.Fatal("expected invalid range syntax to fail")
			}
		})
	}
}

func TestRangeLoopWithoutVariablesCompiles(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"repetir (0..3) imprimir(0)",
		"repetir (3..0, paso = -1)\n\t\timprimir(0)",
	} {
		if _, err := Compile("rango.cometa", []byte("fn inicio()\n\t"+body+"\n")); err != nil {
			t.Fatalf("%q: %v", body, err)
		}
	}
}

func TestRepeatCount(t *testing.T) {
	t.Parallel()
	runCometa(t, `fn veces() entero
	imprimir("una vez")
	2
fn inicio()
	var n = 3
	repetir (n) imprimir("hola")
	repetir (veces()) |i| imprimir(i)
	repetir (n - 1) |valor, indice|
		si (valor == 1) continuar
		imprimir("${valor} ${indice}")
	repetir (0) imprimir("nunca")
	repetir (-2) |i| imprimir(i)
	repetir (n) |i|
		n = 0
		si (i == 1) romper
		imprimir("vuelta ${i}")
`, "hola\nhola\nhola\nuna vez\n0\n1\n0 0\nvuelta 0\n")
}

func TestRangeStep(t *testing.T) {
	t.Parallel()
	runCometa(t, `const alto = 3
fn paso_de(n entero) entero
	imprimir("paso")
	n
fn inicio()
	var vacia [entero] = []
	repetir (0..vacia.longitud() - 1) |i| imprimir("incorrecto")
	var n = -2
	repetir (0..n) |i| imprimir("incorrecto")
	repetir (alto..n) imprimir("incorrecto")
	repetir (0..7, paso = 3) |i, indice| imprimir("${i} ${indice}")
	repetir (5..0, paso = -2) |i| imprimir(i)
	repetir (alto..n, paso = paso_de(-1) * 2) |i| imprimir(i)
	repetir (0..3, paso = -1) |i| imprimir("incorrecto")
	repetir (3..0, paso = 1) |i| imprimir("incorrecto")
`, "0 0\n3 1\n6 2\n5\n3\n1\npaso\n3\n1\n-1\n")
}
