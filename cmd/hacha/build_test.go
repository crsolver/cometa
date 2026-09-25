package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildGameExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop executable build")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "game.hacha")
	output := filepath.Join(dir, "game")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	if err := os.WriteFile(input, []byte("usar std/pincel/juego\nusar std/pincel/graficos\ntipo Partida\n\tfn actualizar(dt decimal) imprimir(dt)\n\tfn pintar() graficos.limpiar(.Negro)\nfn inicio()\n\tjuego.ejecutar(Partida {}) capturar |e| imprimir(e)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"construir", input, "-o", output}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		t.Fatalf("missing executable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); !os.IsNotExist(err) {
		t.Fatal("build modified source project")
	}
}

func TestGameCLIArguments(t *testing.T) {
	for _, args := range [][]string{{"ejecutar"}, {"construir"}, {"ejecutar", "x.hacha", "-o", "x.exe"}, {"construir", "x.txt"}, {"construir", "x.hacha", "--unknown"}} {
		if err := run(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}
