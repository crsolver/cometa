package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestUsuarioGolden(t *testing.T) {
	source, err := os.ReadFile("testdata/usuario.cometa")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Compile("usuario.cometa", source)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/usuario.go.golden")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("generated Go differs from golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestDiagnosticContainsSourcePosition(t *testing.T) {
	_, err := Compile("roto.cometa", []byte("fn f() entero\n\tno_existe\n"))
	if err == nil || !strings.Contains(err.Error(), "roto.cometa:2:2") {
		t.Fatalf("expected positioned diagnostic, got %v", err)
	}
}

func TestCompilesVariablesCompositeLiteralsAndMemberCall(t *testing.T) {
	source := "tipo Usuario\n\tnombre cadena\n\tamigos [Usuario]\n\tfn activar(valor bool)\n\t\timprimir(valor)\nfn procesar_usuario(usuario Usuario)\n\timprimir(usuario)\nfn inicio()\n\tvar usuario1 = Usuario {nombre: \"andres\"}\n\tvar usuario2 Usuario = {nombre: \"andres\", amigos: [usuario1]}\n\tvar vacios [Usuario] = []\n\tvar usuarios = [usuario1, usuario2]\n\tvar primero = usuarios[0]\n\tprocesar_usuario(usuarios[1])\n\tusuario2.activar(verdadero)\n"
	got, err := Compile("variables.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"usuario1 := &Usuario{Nombre: \"andres\"}", "usuarios := []*Usuario{usuario1, usuario2}", "_ = usuarios[0]", "Procesar_usuario(usuarios[1])", "usuario2.Activar(true)"} {
		if !strings.Contains(string(got), expected) {
			t.Errorf("generated Go does not contain %q:\n%s", expected, got)
		}
	}
}

func TestCompilesListAndInfiniteRepeats(t *testing.T) {
	source := "fn inicio()\n\tvar lista = [1, 2]\n\trepetir (lista) |valor| imprimir(valor)\n\trepetir (lista) |valor, indice|\n\t\tsi (indice == 0) continuar\n\t\timprimir(valor)\n\t\tromper\n\trepetir imprimir(\"hola\")\n"
	got, err := Compile("ciclos.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"for _, valor := range lista {",
		"for indice, valor := range lista {",
		"indice := int64(indice)",
		"continue",
		"break",
		"for {\n\t\t_hsimprimir(\"hola\")",
	} {
		if !strings.Contains(string(got), expected) {
			t.Errorf("generated Go does not contain %q:\n%s", expected, got)
		}
	}
}
