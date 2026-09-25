package sema

import (
	"strings"
	"testing"

	"hacha/internal/lexer"
	"hacha/internal/parser"
)

func checkSource(source string) error {
	tokens, err := lexer.Lex("prueba.hacha", source)
	if err != nil {
		return err
	}
	program, err := parser.Parse("prueba.hacha", tokens)
	if err != nil {
		return err
	}
	_, err = Check("prueba.hacha", program)
	return err
}

func TestSemanticErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"unknown name", "fn f() entero desconocido\n", `el nombre "desconocido" no existe`},
		{"missing receiver marker", "tipo T\n\tx entero\n\tfn f() entero x\n", `el nombre "x" no existe; los campos requieren '@'`},
		{"incompatible assignment", "tipo T\n\tx entero\n\tfn f()\n\t\t@x = verdadero\n", "no se puede asignar bool a entero"},
		{"invalid operator", "fn f(a decimal) decimal a % 2\n", `el operador % requiere enteros`},
		{"missing return", "fn f(a entero) entero\n\tsi verdadero a\n", `debe producir entero en todos los caminos`},
		{"duplicate declaration", "fn f()\n\timprimir(1)\nfn f()\n\timprimir(2)\n", `la función "f" ya fue declarada`},
		{"invalid entrypoint", "fn inicio(valor entero)\n\timprimir(valor)\n", "inicio debe declararse"},
		{"branch mismatch", "tipo T\n\tx bool\n\tfn f()\n\t\t@x = si verdadero verdadero sino 1\n", "las ramas producen bool y entero"},
		{"empty inferred list", "fn inicio()\n\tvar lista = []\n", "no se puede inferir el tipo de una lista vacía"},
		{"wrong contextual field", "tipo Usuario\n\tedad entero\nfn inicio()\n\tvar usuario Usuario = {edad: falso}\n", `el campo "edad" debe ser entero, no bool`},
		{"mixed inferred list", "fn inicio()\n\tvar lista = [1, falso]\n", "el elemento debe ser entero, no bool"},
		{"index requires list", "fn inicio()\n\tvar x = 1[0]\n", "solo se pueden indexar listas, no entero"},
		{"index requires number", "fn inicio()\n\tvar lista = [1]\n\tvar x = lista[falso]\n", "el índice de una lista debe ser entero, no bool"},
		{"unknown member", "tipo Usuario\n\tnombre cadena\nfn inicio()\n\tvar usuario = Usuario {}\n\tusuario.activar()\n", `el método "activar" no existe en Usuario`},
		{"branch local does not escape", "fn inicio()\n\tsi verdadero\n\t\tvar local = 1\n\timprimir(local)\n", `el nombre "local" no existe`},
		{"repeat requires list", "fn inicio()\n\trepetir (1) |valor| imprimir(valor)\n", "repetir requiere una lista, no entero"},
		{"range start type", "fn inicio()\n\trepetir (falso..5) |i| imprimir(i)\n", "el inicio del rango debe ser entero"},
		{"range end type", "fn inicio()\n\trepetir (0..\"fin\") |i| imprimir(i)\n", "el final del rango debe ser entero"},
		{"range binding scope", "fn inicio()\n\trepetir (0..5) |i| imprimir(i)\n\timprimir(i)\n", `el nombre "i" no existe`},
		{"range duplicate binding", "fn inicio()\n\trepetir (0..5) |i, i| imprimir(i)\n", "nombres distintos"},
		{"range shadow binding", "fn inicio()\n\tvar i = 0\n\trepetir (0..5) |i| imprimir(i)\n", "ya fue declarada"},
		{"range end before binding", "fn inicio()\n\trepetir (0..i) |i| imprimir(i)\n", `el nombre "i" no existe`},
		{"continue outside loop", "fn inicio()\n\tcontinuar\n", "'continuar' solo puede usarse dentro de un ciclo"},
		{"break outside loop", "fn inicio()\n\tromper\n", "'romper' solo puede usarse dentro de un ciclo"},
		{"repeat binding does not escape", "fn inicio()\n\tvar lista = [1]\n\trepetir (lista) |valor| imprimir(valor)\n\timprimir(valor)\n", `el nombre "valor" no existe`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkSource(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestChecksRepeatElementAndIndexTypes(t *testing.T) {
	source := "tipo Usuario\n\tnombre cadena\nfn inicio()\n\tvar usuarios = [Usuario {nombre: \"Ana\"}]\n\trepetir (usuarios) |usuario, indice|\n\t\tsi (indice == 0) continuar\n\t\timprimir(usuario.nombre)\n\t\tromper\n\trepetir imprimir(\"hola\")\n"
	if err := checkSource(source); err != nil {
		t.Fatal(err)
	}
}

func TestVariablesCompositeLiteralsListsAndMemberCalls(t *testing.T) {
	source := "tipo Usuario\n\tnombre cadena\n\tedad entero\n\tamigos [Usuario]\n\tfn activar(valor bool)\n\t\timprimir(valor)\nfn procesar_usuario(usuario Usuario)\n\timprimir(usuario)\nfn inicio()\n\tvar usuario1 = Usuario {nombre: \"andres\", edad: 29}\n\tvar usuario2 Usuario = {nombre: \"andres\", edad: 29, amigos: [usuario1]}\n\tvar lista2 [Usuario] = []\n\tvar usuarios = [usuario1, usuario2]\n\tvar primero = usuarios[0]\n\tprocesar_usuario(usuarios[1])\n\tusuario2.activar(verdadero)\n"
	if err := checkSource(source); err != nil {
		t.Fatal(err)
	}
}

func TestAllPathsProduceDeclaredType(t *testing.T) {
	source := "fn f(n entero) entero\n\tsi (n < 0) 0\n\tosi (n == 0) 1\n\tsino 2\n"
	if err := checkSource(source); err != nil {
		t.Fatal(err)
	}
}
