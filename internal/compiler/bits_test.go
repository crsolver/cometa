package compiler

import (
	"strings"
	"testing"
)

func TestBitFunctionsAndRadixLiterals(t *testing.T) {
	t.Parallel()
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/mate

const enroque_corto = 0b0001
const enroque_largo = 0b0010

fn jugada(desde entero, hasta entero, corona entero) entero
	mate.bit_o(desde, mate.bit_o(mate.desplazar_izquierda(hasta, 6), mate.desplazar_izquierda(corona, 12)))

fn nombre(n entero) cadena
	casos n
		0x10 => "dieciséis"
		0b11, 0xff => "tres o 255"
		_ => "otro"

fn inicio()
	imprimir(0xFF)
	imprimir(0x7fffffffffffffff)
	imprimir(0b1010)
	imprimir(-0x10)
	imprimir(0x1F.formato(1))
	var m = jugada(12, 28, 3)
	imprimir(m)
	imprimir(mate.bit_y(m, 0x3F))
	imprimir(mate.bit_y(mate.desplazar_derecha(m, 6), 0x3F))
	imprimir(mate.desplazar_derecha(m, 12))
	var derechos = mate.bit_o(enroque_corto, enroque_largo)
	derechos = mate.bit_y(derechos, mate.bit_no(enroque_corto))
	imprimir(derechos)
	imprimir(mate.bit_y(derechos, enroque_largo) != 0)
	imprimir(mate.bit_xor(0b1100, 0b1010))
	imprimir(mate.bit_no(0))
	imprimir(mate.desplazar_derecha(-8, 1))
	imprimir(mate.desplazar_izquierda(1, 64))
	imprimir(nombre(16))
	imprimir(nombre(255))
	imprimir(nombre(4))
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"255", "9223372036854775807", "10", "-16", "31.0",
		"14092", "12", "28", "3", "2", "verdadero", "6", "-1", "-4", "0",
		"dieciséis", "tres o 255", "otro",
	}, "\n") + "\n"
	runGeneratedGo(t, generated, want)

	for _, tc := range []struct{ name, source, want string }{
		{"hex vacío", "fn inicio()\n\timprimir(0x)\n", "se esperaba un dígito hexadecimal después de '0x'"},
		{"hex inválido", "fn inicio()\n\timprimir(0xFG)\n", "se esperaba un dígito hexadecimal"},
		{"binario inválido", "fn inicio()\n\timprimir(0b102)\n", "se esperaba un dígito binario después de '0b'"},
		{"fuera de rango", "fn inicio()\n\timprimir(0x8000000000000000)\n", "fuera del rango de entero"},
		{"decimal en bits", "usar std/mate\nfn inicio()\n\timprimir(mate.bit_y(1.5, 1))\n", "entero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}
