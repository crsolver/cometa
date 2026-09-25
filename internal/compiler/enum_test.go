package compiler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func runCometa(t *testing.T, source, want string) []byte {
	t.Helper()
	generated, err := Compile("enum.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return runGeneratedGo(t, generated, want)
}

func runGeneratedGo(t *testing.T, generated []byte, want string) []byte {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, generated, 0600); err != nil {
		t.Fatal(err)
	}
	goName := "go"
	if runtime.GOOS == "windows" {
		goName += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", goName), "run", file)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated program failed: %v\n%s\n%s", err, out, generated)
	}
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
		t.Fatalf("output = %q, want %q\n%s", got, want, generated)
	}
	return generated
}

func TestEnumRuntimeAndReferenceSemantics(t *testing.T) {
	source := `tipo Boton
	caracter cadena
tipo Contador
	valor entero
enum E
	A
	Boton Boton
	Texto cadena
	Numero entero
	Activo bool
	Lista [E]
	Anidado E
fn obtener(c Contador) E
	c.valor = c.valor + 1
	E.A
fn lista(e E) [Boton]
	casos e
		.A => []
		_ =>
			si verdadero
				casos e
					_ => []
			sino []
fn inicio()
	var contador = Contador {}
	casos obtener(contador)
		.A => imprimir("A")
		_ => imprimir("incorrecto")
	var resultado = casos obtener(contador)
		.A =>
			var local = "valor"
			local
		_ => "incorrecto"
	imprimir(resultado)
	imprimir(contador.valor)
	var boton = Boton {caracter: "a"}
	var evento = E.Boton(boton)
	boton.caracter = "b"
	casos evento |e|
		.Boton =>
			imprimir(e.caracter)
			e.caracter = "c"
			e = Boton {caracter: "otro"}
		_ => imprimir("incorrecto")
	imprimir(boton.caracter)
	var _cometa1 = 7
	var n = 2
	var eventos = [E.Texto("texto"), E.Numero(n), E.Activo(verdadero), E.Lista([E.A]), E.Anidado(E.A)]
	repetir (eventos) |evento2|
		casos evento2 |p|
			.Texto => imprimir(p)
			.Numero => imprimir(p + 1)
			.Activo => imprimir(p)
			.Lista =>
				casos p[0]
					_ => imprimir("lista")
			.Anidado =>
				casos p
					_ => imprimir("anidado")
			_ => imprimir("incorrecto")
	var vacios = lista(E.A)
	imprimir(vacios)
	var vacios2 = lista(E.Texto("otro"))
	imprimir(vacios2)
	var contextual Boton = casos E.A
		_ => si verdadero Boton {caracter: "contextual"} sino {}
	imprimir(contextual.caracter)
	E.Numero(1)
`
	generated := runCometa(t, source, "A\nvalor\n2\nb\nc\ntexto\n3\ntrue\nlista\nanidado\n[]\n[]\ncontextual\n")
	for _, fragment := range []string{"payload2 *Boton", "payload6 []*E", "payload7 *E", "switch", "func() string", "func() *Boton"} {
		if !strings.Contains(string(generated), fragment) {
			t.Errorf("missing %q in generated storage/match code", fragment)
		}
	}
}

func TestMatchLoopControlAndExpressionContexts(t *testing.T) {
	source := `enum E
	A
	B
fn identidad(n entero) entero n
fn inicio()
	var suma = 0
	repetir ([1, 2, 3]) |i|
		casos E.A
			.A =>
				si i == 1 continuar
				repetir
					casos E.B
						_ => romper
				suma = suma + i
				si i == 2 romper
			.B => imprimir("incorrecto")
	imprimir(suma)
	var valor = casos E.A
		_ =>
			repetir
				casos E.A
					_ => romper
			4
	valor = casos E.B
		.A => 0
		.B => valor + 1
	imprimir(valor)
	imprimir(identidad(casos E.A
		.A => 8
		.B => 9
	))
	var lista [entero] = [casos E.B
		_ => 10
	]
	imprimir(lista[0])
`
	runCometa(t, source, "2\n5\n8\n10\n")
}

func TestEnumExample(t *testing.T) {
	source, err := os.ReadFile("../../examples/usuario.cometa")
	if err != nil {
		t.Fatal(err)
	}
	runCometa(t, string(source), "izquierda\nleyendo el boton:\nb\ncargando pagina\nbuton presionado:\nb\nv4\nv6\n")
}
