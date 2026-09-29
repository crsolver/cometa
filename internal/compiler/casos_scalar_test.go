package compiler

import (
	"strings"
	"testing"
)

func TestCasosOverScalars(t *testing.T) {
	source := `fn nombre(n entero) cadena
	casos n
		1 => "uno"
		2, 3 => "dos o tres"
		-1 => "menos uno"
		_ => "otro"

fn puntos(comando cadena) entero
	casos comando
		"a" => 10
		"b", "c" => 20
		_ => 0

fn si_no(b bool) cadena
	casos b
		verdadero => "sí"
		falso => "no"

fn inicio()
	imprimir(nombre(1))
	imprimir(nombre(3))
	imprimir(nombre(-1))
	imprimir(nombre(9))
	imprimir(puntos("c"))
	imprimir(puntos("z"))
	imprimir(si_no(1 < 2))
	imprimir(si_no(falso))
	var total = 0
	var i = 0
	mientras i < 6
		casos i
			0 => total += 1
			1, 2 =>
				total += 10
				si i == 2
					i += 1
					continuar
			_ => total += 100
		i += 1
	imprimir(total)
	repetir (0..5) |k|
		casos k
			3 => romper
			_ => imprimir("k=${k}")
`
	got, err := Compile("casos.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "uno\ndos o tres\nmenos uno\notro\n20\n0\nsí\nno\n321\nk=0\nk=1\nk=2\n")
}

func TestCasosOverScalarsErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"missing wildcard", "fn inicio()\n\tcasos 1\n\t\t1 => imprimir(1)\n", "requiere una rama final '_'"},
		{"wrong type", "fn inicio()\n\tvar n = 1\n\tcasos n\n\t\t\"a\" => imprimir(1)\n\t\t_ => imprimir(2)\n", "el valor debe ser entero, no cadena"},
		{"duplicate", "fn inicio()\n\tvar n = 1\n\tcasos n\n\t\t1, 1 => imprimir(1)\n\t\t_ => imprimir(2)\n", "duplicado"},
		{"after wildcard", "fn inicio()\n\tvar n = 1\n\tcasos n\n\t\t_ => imprimir(1)\n\t\t2 => imprimir(2)\n", "después de '_'"},
		{"decimal", "fn inicio()\n\tvar n = 1.5\n\tcasos n\n\t\t1 => imprimir(1)\n\t\t_ => imprimir(2)\n", "casos requiere"},
		{"bool incomplete", "fn inicio()\n\tvar b = verdadero\n\tcasos b\n\t\tverdadero => imprimir(1)\n", "requiere una rama final '_'"},
		{"binding", "fn inicio()\n\tvar n = 1\n\tcasos n |x|\n\t\t_ => imprimir(1)\n", "no admite un nombre"},
		{"variant label on int", "fn inicio()\n\tvar n = 1\n\tcasos n\n\t\t.Uno => imprimir(1)\n\t\t_ => imprimir(2)\n", "valores literales"},
		{"literal on enum", "enum E\n\tA\nfn inicio()\n\tvar e = E.A\n\tcasos e\n\t\t1 => imprimir(1)\n\t\t_ => imprimir(2)\n", "solo sirven para entero, cadena o bool"},
		{"expression label", "fn inicio()\n\tvar n = 1\n\tvar m = 2\n\tcasos n\n\t\t1 + 1 => imprimir(1)\n\t\t_ => imprimir(2)\n", "se esperaba '=>'"},
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

func TestCasosWithConstantLabels(t *testing.T) {
	source := `const aire entero = 0
const ladrillo entero = 1
const moneda entero = 2
const pinchos entero = 3
const alto = 9
const saludo cadena = "hola"

fn nombre(n entero) cadena
	casos n
		aire => "aire"
		ladrillo, moneda => "sólido o moneda"
		pinchos => "pinchos"
		-1 => "menos uno"
		_ => "otro"

fn palabra(s cadena) entero
	casos s
		saludo => 1
		"adiós" => 2
		_ => 0

fn inicio()
	imprimir(nombre(0))
	imprimir(nombre(2))
	imprimir(nombre(3))
	imprimir(nombre(-1))
	imprimir(nombre(7))
	imprimir(palabra("hola"))
	imprimir(palabra("adiós"))
	var contador = 0
	casos contador
		aire => contador += 5
		_ => contador += 1
	imprimir(contador)
`
	got, err := Compile("constantes.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, got, "aire\nsólido o moneda\npinchos\nmenos uno\notro\n1\n2\n5\n")
}

func TestCasosWithConstantLabelErrors(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"duplicate value", "const a entero = 1\nconst b entero = 1\nfn inicio()\n\tvar n = 1\n\tcasos n\n\t\ta, b => imprimir(1)\n\t\t_ => imprimir(2)\n", "duplicado"},
		{"duplicate with literal", "const a entero = 1\nfn inicio()\n\tvar n = 1\n\tcasos n\n\t\ta => imprimir(1)\n\t\t1 => imprimir(2)\n\t\t_ => imprimir(3)\n", "duplicado"},
		{"variable", "var a entero = 1\nfn inicio()\n\tvar n = 1\n\tcasos n\n\t\ta, 2 => imprimir(1)\n\t\t_ => imprimir(2)\n", "no es una constante"},
		{"computed constant", "const a entero = 1 + 1\nfn inicio()\n\tvar n = 1\n\tcasos n\n\t\ta, 2 => imprimir(1)\n\t\t_ => imprimir(2)\n", "no es una constante"},
		{"wrong type", "const a cadena = \"x\"\nfn inicio()\n\tvar n = 1\n\tcasos n\n\t\ta, 2 => imprimir(1)\n\t\t_ => imprimir(2)\n", "el valor debe ser entero, no cadena"},
		{"unknown name", "fn inicio()\n\tvar n = 1\n\tcasos n\n\t\tuno, 2 => imprimir(1)\n\t\t_ => imprimir(2)\n", "no es una constante"},
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
