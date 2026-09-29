package compiler

import (
	"strings"
	"testing"
)

func TestEnumEqualityWithUnitVariants(t *testing.T) {
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
