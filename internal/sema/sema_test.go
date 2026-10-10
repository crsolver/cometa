package sema

import (
	"errors"
	"strings"
	"testing"

	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/lexer"
	"github.com/crsolver/cometa/internal/parser"
)

func checkSource(source string) error {
	tokens, err := lexer.Lex("prueba.cometa", source)
	if err != nil {
		return err
	}
	program, err := parser.Parse("prueba.cometa", tokens)
	if err != nil {
		return err
	}
	_, err = Check("prueba.cometa", program)
	return err
}

func TestSemanticErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"unknown name", "fn f() entero desconocido\n", `el nombre "desconocido" no existe`},
		{"missing receiver marker", "tipo T\n\tx entero\n\tfn f() entero x\n", `el nombre "x" no existe; para usar un campo o método del objeto escribe '@x'`},
		{"incompatible assignment", "tipo T\n\tx entero\n\tfn f()\n\t\t@x = verdadero\n", "no se puede asignar bool a entero"},
		{"invalid operator", "fn f(a decimal) decimal a % 2\n", `el operador % requiere enteros`},
		{"missing return", "fn f(a entero) entero\n\tsi verdadero a\n", `debe producir entero en todos los caminos`},
		{"duplicate declaration", "fn f()\n\timprimir(1)\nfn f()\n\timprimir(2)\n", `la función "f" ya fue declarada`},
		{"invalid entrypoint", "fn inicio(valor entero)\n\timprimir(valor)\n", "inicio debe declararse"},
		{"branch mismatch", "tipo T\n\tx bool\n\tfn f()\n\t\t@x = si verdadero verdadero sino 1\n", "las ramas producen bool y entero"},
		{"empty inferred list", "fn inicio()\n\tvar lista = []\n", "no se puede inferir el tipo de una lista vacía"},
		{"wrong contextual field", "tipo Usuario\n\tedad entero\nfn inicio()\n\tvar usuario Usuario = {edad: falso}\n", `el campo "edad" debe ser entero, no bool`},
		{"mixed inferred list", "fn inicio()\n\tvar lista = [1, falso]\n", "el elemento debe ser entero, no bool"},
		{"index requires collection", "fn inicio()\n\tvar x = 1[0]\n", "solo se pueden indexar listas o mapas, no entero"},
		{"index requires number", "fn inicio()\n\tvar lista = [1]\n\tvar x = lista[falso]\n", "el índice de una lista debe ser entero, no bool"},
		{"unknown member", "tipo Usuario\n\tnombre cadena\nfn inicio()\n\tvar usuario = Usuario {}\n\tusuario.activar()\n", `el método "activar" no existe en Usuario`},
		{"branch local does not escape", "fn inicio()\n\tsi verdadero\n\t\tvar local = 1\n\timprimir(local)\n", `el nombre "local" no existe`},
		{"repeat requires collection", "fn inicio()\n\trepetir (1.5) |valor| imprimir(valor)\n", "repetir requiere una lista, un mapa, una cadena o una cantidad entera de vueltas, no decimal"},
		{"repeat over a list needs a variable", "fn inicio()\n\tvar xs = [1]\n\trepetir (xs) imprimir(1)\n", "para recorrer [entero] falta el nombre de cada elemento: repetir (...) |elemento|"},
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

func TestMissingResultPointsAtDeclaredType(t *testing.T) {
	err := checkSource("fn sx(x decimal) entero\n\timprimir(\"\")\n")
	var found *diagnostic.Error
	if !errors.As(err, &found) {
		t.Fatalf("se esperaba un diagnóstico; got %v", err)
	}
	if !strings.Contains(found.Message, "debe producir entero en todos los caminos") {
		t.Fatalf("mensaje inesperado: %v", err)
	}
	if found.Pos.Line != 1 || found.Pos.Column != 18 {
		t.Fatalf("el error debe marcar el tipo declarado (1:18); got %d:%d", found.Pos.Line, found.Pos.Column)
	}
}

func TestUnmetInterfaceSaysWhy(t *testing.T) {
	const criatura = "interfaz Criatura\n\tfn atacar() entero\n"
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"missing method", criatura + "tipo Lobo\n\tvida entero\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "el literal es Lobo, pero se esperaba Criatura: a Lobo le falta el método atacar() entero"},
		{"other result", criatura + "tipo Lobo\n\tpub fn atacar() decimal 1.5\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "Lobo.atacar devuelve decimal y Criatura pide entero"},
		{"no result", criatura + "tipo Lobo\n\tpub fn atacar()\n\t\timprimir(1)\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "Lobo.atacar no devuelve nada y Criatura pide entero"},
		{"other parameters", criatura + "tipo Lobo\n\tpub fn atacar(fuerza entero) entero fuerza\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "Lobo.atacar es atacar(fuerza entero) entero y Criatura pide atacar() entero"},
		{"private method", criatura + "tipo Lobo\n\tfn atacar() entero 1\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "Lobo.atacar existe pero no es pub"},
		{"private embedding", criatura + "tipo Animal\n\tpub fn atacar() entero 1\ntipo Lobo\n\tAnimal\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "Lobo.atacar viene del tipo embebido Animal, que no es pub"},
		{"field", criatura + "tipo Lobo\n\tpub atacar entero\nfn inicio()\n\tvar c Criatura = Lobo {}\n", "Lobo.atacar es un campo, no un método"},
		{"list element", criatura + "tipo Lobo\n\tvida entero\nfn inicio()\n\tvar lobo = Lobo {}\n\tvar cs [Criatura] = [lobo]\n", "el elemento debe ser Criatura, no Lobo: a Lobo le falta el método atacar() entero"},
		{"assignment", criatura + "tipo Golem\n\tvida entero\nfn inicio()\n\tvar golem = Golem {}\n\tvar c Criatura = golem\n", "no se puede asignar Golem a Criatura: a Golem le falta el método atacar() entero"},
		{"argument", criatura + "tipo Lobo\n\tvida entero\nfn ver(c Criatura)\n\timprimir(c.atacar())\nfn inicio()\n\tvar lobo = Lobo {}\n\tver(lobo)\n", "el argumento 1 debe ser Criatura, no Lobo: a Lobo le falta el método atacar() entero"},
		{"optional interface", criatura + "tipo Lobo\n\tvida entero\nfn inicio()\n\tvar lobo = Lobo {}\n\tvar c Criatura? = lobo\n", "a Lobo le falta el método atacar() entero"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkSource(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
	if err := checkSource("fn inicio()\n\tvar n entero = 1.5\n"); err == nil || strings.HasSuffix(err.Error(), ": ") || strings.Contains(err.Error(), "entero: ") {
		t.Fatalf("un error sin interfaces no lleva motivo: %v", err)
	}
}
