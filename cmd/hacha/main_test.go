package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesDefaultOutput(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "hola.hacha")
	if err := os.WriteFile(input, []byte("fn inicio()\n\timprimir(\"hola\")\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"compilar", input}); err != nil {
		t.Fatal(err)
	}
	output := strings.TrimSuffix(input, ".hacha") + ".go"
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "func main()") || !strings.Contains(string(generated), `fmt.Println("hola")`) {
		t.Fatalf("unexpected output:\n%s", generated)
	}
}

func TestRunAcceptsOutputAfterInput(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "hola.hacha")
	output := filepath.Join(directory, "salida.go")
	if err := os.WriteFile(input, []byte("fn inicio()\n\timprimir(\"hola\")\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"compilar", input, "-o", output}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
}

func TestUsageMentionsLSPMode(t *testing.T) {
	err := run(nil)
	if err == nil || !strings.Contains(err.Error(), "lsp") {
		t.Fatalf("usage error = %v", err)
	}
}

func TestRecognizesVSCodeLSPArguments(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"lsp"}, true},
		{[]string{"lsp", "--stdio"}, true},
		{[]string{"lsp", "--socket=9000"}, false},
		{[]string{"--stdio"}, false},
	}
	for _, test := range tests {
		if got := isLSPCommand(test.args); got != test.want {
			t.Errorf("isLSPCommand(%q) = %v, want %v", test.args, got, test.want)
		}
	}
}

func TestRunResolvesFileModules(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "inicio.hacha")
	if err := os.WriteFile(input, []byte("usar biblioteca como b\nfn inicio() imprimir(b.valor())\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "biblioteca.hacha"), []byte("fn valor() num 42\n"), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "salida.go")
	if err := run([]string{"compilar", input, "-o", output}); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(source), "package main") != 1 || !strings.Contains(string(source), "HachaModulo") {
		t.Fatal(string(source))
	}
}
