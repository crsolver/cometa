package compiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// runGeneratedGoWithInput is runGeneratedGo with the given text on stdin.
func runGeneratedGoWithInput(t *testing.T, generated []byte, input, want string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "main.go")
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
	cmd.Stdin = strings.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("generated program failed: %v\n%s%s\n%s", err, out, stderr.Bytes(), generated)
	}
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
		t.Fatalf("output = %q, want %q\n%s", got, want, generated)
	}
}

const consolaSuma = `usar std/consola

fn inicio()
	consola.escribir("¿Nombre? ")
	var nombre = consola.leer_linea() o "nadie"
	imprimir("Hola, ${nombre}")
	var suma = 0
	mientras verdadero
		si consola.leer_linea() |linea|
			suma += linea.recortar().a_entero() o 0
		sino
			romper
	imprimir(suma)
	imprimir(consola.leer_linea())
`

func TestConsolaLeeLineas(t *testing.T) {
	generated, err := Compile("consola.cometa", []byte(consolaSuma))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("std/consola no debe depender de Ebitengine")
	}
	// A leading BOM, CRLF endings, a blank line, and a last line without a newline.
	runGeneratedGoWithInput(t, generated, "\ufeffAna María\r\n10\n\n 20 \n12", "¿Nombre? Hola, Ana María\n42\nNinguno\n")
}

func TestConsolaSinEntrada(t *testing.T) {
	generated, err := Compile("consola.cometa", []byte(consolaSuma))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGoWithInput(t, generated, "", "¿Nombre? Hola, nadie\n0\nNinguno\n")
}

func TestConsolaRequiereImportar(t *testing.T) {
	if _, err := Compile("consola.cometa", []byte("fn inicio()\n\timprimir(consola.leer_linea())\n")); err == nil {
		t.Fatal("se esperaba un error sin usar std/consola")
	}
}
