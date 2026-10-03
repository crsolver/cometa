package main

import (
	"os"
	"strings"
	"testing"
)

// `cometa mapa` writes the bundled editor to the cache folder and opens that file.
func TestRunMapaOpensEditor(t *testing.T) {
	cache := t.TempDir()
	for _, name := range []string{"LocalAppData", "XDG_CACHE_HOME", "HOME"} {
		t.Setenv(name, cache)
	}
	var opened string
	original := openBrowser
	openBrowser = func(path string) error { opened = path; return nil }
	defer func() { openBrowser = original }()

	if err := run([]string{"mapa"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(opened, cache) {
		t.Fatalf("abrió %q, fuera de %q", opened, cache)
	}
	page, err := os.ReadFile(opened)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "<title>Editor de mapas de Cometa</title>") {
		t.Fatal("el archivo abierto no es el editor de mapas")
	}
	if err := run([]string{"mapa", "sobra"}); err == nil {
		t.Fatal("aceptó un argumento de más")
	}
}
