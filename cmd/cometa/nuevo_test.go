package main

import (
	"os"
	"path/filepath"
	"testing"

	"cometa/internal/compiler"
)

// Both starter templates must compile, or `cometa nuevo` would hand beginners a broken first file.
func TestNewProjectTemplatesCompile(t *testing.T) {
	for _, game := range []bool{false, true} {
		dir := t.TempDir()
		previous, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		err := newProject("demo", game)
		os.Chdir(previous)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compiler.CompileProject(filepath.Join(dir, "demo", "principal.cometa"), nil); err != nil {
			t.Errorf("la plantilla (juego=%v) no compila: %v", game, err)
		}
	}
}

func TestNewProjectRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	previous, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(previous)
	if err := newProject("../fuera", false); err == nil {
		t.Error("un nombre con '/' debe rechazarse")
	}
	os.MkdirAll("ocupada", 0o755)
	os.WriteFile(filepath.Join("ocupada", "x"), []byte("x"), 0o644)
	if err := newProject("ocupada", false); err == nil {
		t.Error("una carpeta con contenido no debe sobrescribirse")
	}
	if err := runNew(nil); err == nil {
		t.Error("falta el nombre")
	}
}
