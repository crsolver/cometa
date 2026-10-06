package compiler

import (
	"strings"
	"testing"
)

func TestStringPaddingAndRepeat(t *testing.T) {
	source := `fn reloj(segundos entero) cadena
	var minutos = cadena(segundos / 60).rellenar_izquierda(2, "0")
	var resto = cadena(segundos % 60).rellenar_izquierda(2, "0")
	"${minutos}:${resto}"

fn inicio()
	imprimir(cadena(208).rellenar_izquierda(7, "0"))
	imprimir(reloj(187))
	imprimir("[" + "año".rellenar_izquierda(6) + "]")
	imprimir("[" + "año".rellenar_derecha(6, "·") + "]")
	imprimir("[" + "ab".rellenar_izquierda(7, "<>") + "]")
	imprimir("largo".rellenar_izquierda(3, "0"))
	imprimir("x".rellenar_derecha(4, ""))
	imprimir("x".rellenar_derecha(relleno = "-", ancho = 3))
	imprimir("ña".repetir(3))
	imprimir("[" + "ab".repetir(0) + "ab".repetir(-2) + "]")
	imprimir("=".repetir(5).longitud())
	repetir (0..2) |i|
		imprimir("-".repetir(i + 1))
`
	got, err := Compile("rellenar.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"0000208", "03:07", "[   año]", "[año···]", "[<><><ab]", "largo", "x", "x--",
		"ñañaña", "[]", "5", "-", "--",
	}, "\n") + "\n"
	runGeneratedGo(t, got, want)

	for _, tc := range []struct{ name, source, want string }{
		{"sin ancho", "fn inicio()\n\timprimir(\"a\".rellenar_izquierda())\n", `falta el parámetro "ancho"`},
		{"relleno entero", "fn inicio()\n\timprimir(\"a\".rellenar_derecha(3, 0))\n", "debe ser cadena"},
		{"veces decimal", "fn inicio()\n\timprimir(\"a\".repetir(1.5))\n", "debe ser entero"},
		{"repetir como miembro", "fn inicio()\n\timprimir(\"a\".repetir)\n", "se esperaba un miembro"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}
