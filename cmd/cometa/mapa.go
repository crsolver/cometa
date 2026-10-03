package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	mapaeditor "github.com/crsolver/cometa/tools/mapa-editor"
)

// openBrowser opens a local file with the system's default program; tests replace it.
var openBrowser = func(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}

// runMapa writes the bundled map editor to the user's cache folder and opens
// it in the browser. The path never changes, so the browser keeps the map
// being edited between runs.
func runMapa(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("uso: cometa mapa")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("no se encontró la carpeta de caché del usuario: %w", err)
	}
	dir := filepath.Join(cache, "cometa", "mapa-editor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("no se pudo crear %s: %w", dir, err)
	}
	path := filepath.Join(dir, "index.html")
	if err := os.WriteFile(path, mapaeditor.HTML, 0o644); err != nil {
		return fmt.Errorf("no se pudo escribir %s: %w", path, err)
	}
	if err := openBrowser(path); err != nil {
		fmt.Printf("No se pudo abrir el navegador. Abre este archivo a mano:\n%s\n", path)
		return nil
	}
	fmt.Printf("Editor de mapas abierto en el navegador:\n%s\n", path)
	return nil
}
