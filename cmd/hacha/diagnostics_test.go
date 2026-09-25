package main

import (
	"hacha/internal/diagnostic"
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidProgramPreservesOutput(t *testing.T) {
	for _, command := range []string{"compilar", "construir", "ejecutar"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			input, output := filepath.Join(dir, "errores.hacha"), filepath.Join(dir, "salida")
			if err := os.WriteFile(input, []byte("fn inicio()\n\tvar a bool = 1\n\tvar b entero = falso\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(output, []byte("conservar"), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{command, input}
			if command != "ejecutar" {
				args = append(args, "-o", output)
			}
			if err := run(args); len(diagnostic.Flatten(err)) != 2 {
				t.Fatalf("diagnostics: %v", err)
			}
			data, err := os.ReadFile(output)
			if err != nil || string(data) != "conservar" {
				t.Fatalf("output changed: %q %v", data, err)
			}
		})
	}
}
