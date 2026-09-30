package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		err := newProject("demo", game, true)
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
	if err := newProject("../fuera", false, true); err == nil {
		t.Error("un nombre con '/' debe rechazarse")
	}
	os.MkdirAll("ocupada", 0o755)
	os.WriteFile(filepath.Join("ocupada", "x"), []byte("x"), 0o644)
	if err := newProject("ocupada", false, true); err == nil {
		t.Error("una carpeta con contenido no debe sobrescribirse")
	}
	if err := runNew(nil); err == nil {
		t.Error("falta el nombre")
	}
}

func newInTemp(t *testing.T, game, agents bool) string {
	t.Helper()
	dir := t.TempDir()
	previous, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previous)
	if err := newProject("demo", game, agents); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "demo")
}

func TestNewProjectWritesAgentGuide(t *testing.T) {
	for _, game := range []bool{false, true} {
		dir := newInTemp(t, game, true)
		guide, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(guide)
		if !strings.Contains(text, "profesor") || !strings.Contains(text, "aprender") {
			t.Error("la guía debe fijar el papel de profesor y el propósito del lenguaje")
		}
		if strings.Contains(text, "Juegos con Pincel") != game {
			t.Errorf("la sección de Pincel debe aparecer solo en juegos (juego=%v)", game)
		}
		claude, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
		if err != nil || !strings.Contains(string(claude), "@AGENTS.md") {
			t.Error("CLAUDE.md debe importar AGENTS.md")
		}
	}
}

func TestNewProjectWithoutAgents(t *testing.T) {
	dir := newInTemp(t, false, false)
	for _, file := range []string{"AGENTS.md", "CLAUDE.md"} {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			t.Errorf("%s no debe crearse con --sin-agentes", file)
		}
	}
}

// The guide teaches syntax to agents, so its complete examples must really compile.
func TestAgentGuideExamplesCompile(t *testing.T) {
	block := regexp.MustCompile("(?s)```cometa\\n(.*?)```")
	for name, guide := range map[string]string{"base": agentsBase, "pincel": agentsPincel} {
		for i, match := range block.FindAllStringSubmatch(guide, -1) {
			file := filepath.Join(t.TempDir(), "ejemplo.cometa")
			if err := os.WriteFile(file, []byte(match[1]), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := compiler.CompileProject(file, nil); err != nil {
				t.Errorf("ejemplo %d de la guía %s no compila: %v", i+1, name, err)
			}
		}
	}
}
