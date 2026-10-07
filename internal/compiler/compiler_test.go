package compiler

import (
	"os"
	"strings"
	"testing"
)

func TestUsuarioGolden(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("testdata/usuario.cometa")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Compile("usuario.cometa", source)
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "testdata/usuario.go.golden", got)
}

func TestDiagnosticContainsSourcePosition(t *testing.T) {
	t.Parallel()
	_, err := Compile("roto.cometa", []byte("fn f() entero\n\tno_existe\n"))
	if err == nil || !strings.Contains(err.Error(), "roto.cometa:2:2") {
		t.Fatalf("expected positioned diagnostic, got %v", err)
	}
}

func TestCompilesVariablesCompositeLiteralsAndMemberCall(t *testing.T) {
	t.Parallel()
	source := "tipo Usuario\n\tnombre cadena\n\tamigos [Usuario]\n\tfn activar(valor bool)\n\t\timprimir(valor)\nfn procesar_usuario(usuario Usuario)\n\timprimir(usuario)\nfn inicio()\n\tvar usuario1 = Usuario {nombre: \"andres\"}\n\tvar usuario2 Usuario = {nombre: \"andres\", amigos: [usuario1]}\n\tvar vacios [Usuario] = []\n\tvar usuarios = [usuario1, usuario2]\n\tvar primero = usuarios[0]\n\tprocesar_usuario(usuarios[1])\n\tusuario2.activar(verdadero)\n"
	got, err := Compile("variables.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	// Operands are stored in temporaries in evaluation order, so the checks name
	// the shapes that matter and not whole statements.
	for _, expected := range []string{"var usuario1 *Usuario = ", "&Usuario{Nombre: ", "Amigos: ", "var usuarios []*Usuario = ", "[]*Usuario{", " := Procesar_usuario\n", ".Activar\n"} {
		if !strings.Contains(string(got), expected) {
			t.Errorf("generated Go does not contain %q:\n%s", expected, got)
		}
	}
}

func TestCompilesListAndInfiniteRepeats(t *testing.T) {
	t.Parallel()
	source := "fn inicio()\n\tvar lista = [1, 2]\n\trepetir (lista) |valor| imprimir(valor)\n\trepetir (lista) |valor, indice|\n\t\tsi (indice == 0) continuar\n\t\timprimir(valor)\n\t\tromper\n\trepetir imprimir(\"hola\")\n"
	got, err := Compile("ciclos.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"for _, valor := range ",
		", valor := range ",
		"indice := int64(",
		"continue",
		"break",
		"for {\n",
		"string = \"hola\"",
	} {
		if !strings.Contains(string(got), expected) {
			t.Errorf("generated Go does not contain %q:\n%s", expected, got)
		}
	}
}
