package compiler

import (
	"strings"
	"testing"
)

func TestEnumEqualityWithUnitVariants(t *testing.T) {
	t.Parallel()
	source := `enum Estado
	Menu
	Jugando
	Fin cadena

tipo Juego
	estado Estado
	fn esta_jugando() bool
		@estado == .Jugando

fn describir(e Estado) cadena
	si e == .Menu
		"menu"
	osi Estado.Jugando == e
		"jugando"
	sino
		"otro"

fn inicio()
	var e = Estado.Menu
	imprimir(e == .Menu)
	imprimir(e != .Menu)
	imprimir(e == Estado.Jugando)
	imprimir(.Jugando != e)
	imprimir(describir(.Jugando))
	imprimir(describir(.Fin("x")))
	var j = Juego {estado: .Jugando}
	imprimir(j.esta_jugando())
	var f = Estado.Fin("z")
	imprimir(f == .Fin2)
`
	// The last line is invalid on purpose; check it separately below.
	valid := strings.Replace(source, "\timprimir(f == .Fin2)\n", "", 1)
	got, err := Compile("igual.cometa", []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "verdadero\nfalso\nfalso\nverdadero\njugando\notro\nverdadero\n")
	if _, err := Compile("igual.cometa", []byte(source)); err == nil || !strings.Contains(err.Error(), `la variante "Fin2" no existe`) {
		t.Errorf("se esperaba error de variante desconocida, se obtuvo %v", err)
	}
}

func TestEnumEqualityErrors(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, source, want string }{
		{"two values", "enum E\n\tA\n\tB\nfn inicio()\n\tvar x = E.A\n\tvar y = E.B\n\timprimir(x == y)\n", "solo se comparan con una variante sin payload"},
		{"payload variant", "enum E\n\tA\n\tB cadena\nfn inicio()\n\tvar x = E.A\n\timprimir(x == .B)\n", "requiere un payload"},
		{"payload call", "enum E\n\tA\n\tB cadena\nfn inicio()\n\tvar x = E.A\n\timprimir(x == E.B(\"s\"))\n", "solo se comparan con una variante sin payload"},
		{"not an enum", "fn inicio()\n\tvar x = 1\n\timprimir(x == .A)\n", "no se puede inferir el enum"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("se esperaba %q, se obtuvo %v", tc.want, err)
			}
		})
	}
}

func TestOptionalAndResultEquality(t *testing.T) {
	t.Parallel()
	runCometa(t, `tipo Caja
	n entero
fn buscar(nombres [cadena], inicial cadena) cadena?
	repetir (nombres) |nombre|
		si nombre.empieza_con(inicial) retornar nombre
	.Ninguno
fn dividir(a entero, b entero) entero!
	si b == 0 retornar .Error("entre cero")
	a / b
fn guardar(ok bool) !
	si !ok retornar .Error("lleno")
	.Ok
fn avisar(texto cadena) entero
	imprimir(texto)
	2
fn inicio()
	var nombres = ["Ana", "Luis"]
	imprimir(buscar(nombres, "Z") == .Ninguno)
	imprimir(buscar(nombres, "A") != .Ninguno)
	imprimir(.Ninguno == buscar(nombres, "L"))
	imprimir(buscar(nombres, "A") == .Alguno("Ana"))
	imprimir(buscar(nombres, "A") != .Alguno("Ana"))
	imprimir(buscar(nombres, "Z") == .Alguno("Ana"))
	var medio decimal? = 2
	imprimir(medio == .Alguno(2))
	imprimir(.Alguno(avisar("primero")) == nombres.buscar_indice("${avisar("segundo")}"))
	var caja Caja? = .Ninguno
	si caja == .Ninguno imprimir("sin caja")
	imprimir(dividir(6, 3) == .Ok(2))
	imprimir(dividir(1, 0) == .Error("entre cero"))
	imprimir(dividir(1, 0) != .Ok(0))
	imprimir(guardar(verdadero) == .Ok)
	imprimir(guardar(falso) == .Ok)
`, "verdadero\nverdadero\nfalso\nverdadero\nfalso\nfalso\nverdadero\nprimero\nsegundo\nfalso\nsin caja\nverdadero\nverdadero\nverdadero\nverdadero\nfalso\n")
}

func TestOptionalEqualityErrors(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, source, want string }{
		{"compound payload", "tipo Caja\n\tn entero\nfn inicio()\n\tvar c Caja? = .Ninguno\n\timprimir(c == .Alguno(Caja {}))\n", "Caja? solo se compara con .Alguno(valor) cuando el valor es entero, decimal, cadena o bool, no Caja"},
		{"wrong payload", "fn inicio()\n\tvar x entero? = 1\n\timprimir(x == .Alguno(\"a\"))\n", "el payload debe ser entero, no cadena"},
		{"unknown variant", "fn inicio()\n\tvar x entero? = 1\n\timprimir(x == .Nada)\n", `la variante "Nada" no existe en entero?`},
		{"two optionals", "fn inicio()\n\tvar x entero? = 1\n\timprimir(x == x)\n", `el operador "==" no acepta entero? y entero?`},
		{"enum payload", "enum E\n\tA\n\tB cadena\nfn inicio()\n\tvar x = E.A\n\timprimir(x == .B(\"s\"))\n", "solo se comparan con una variante sin payload"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want containing %q", err, tc.want)
			}
		})
	}
}
