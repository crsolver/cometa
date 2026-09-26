package main

import (
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildGameExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop executable build")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "game.cometa")
	output := filepath.Join(dir, "game")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	if err := os.WriteFile(input, []byte("usar std/pincel\nusar std/pincel/graficos\ntipo Partida\n\tpub fn actualizar(dt decimal) imprimir(dt)\n\tpub fn pintar() graficos.limpiar(.Negro)\nfn inicio()\n\tpincel.ejecutar(Partida {}) capturar |e| imprimir(e)\n"), 0600); err != nil {
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
	for _, args := range [][]string{{"ejecutar"}, {"construir"}, {"ejecutar", "x.cometa", "-o", "x.exe"}, {"construir", "x.txt"}, {"construir", "x.cometa", "--unknown"}, {"compilar", "x.cometa", "--escala", "2"}, {"captura", "x.cometa", "--cuadros", "0"}, {"captura", "x.cometa", "--escala", "65"}, {"captura", "x.cometa", "--escala"}} {
		if err := run(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestCaptureGameScreen(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "game.cometa")
	output := filepath.Join(dir, "shot.png")
	game := "usar std/pincel\nusar std/pincel/graficos\nusar std/pincel/lienzo\nvar punto = lienzo.nuevo(1, 1, .Rojo)\ntipo Partida\n\tpub fn actualizar(dt decimal) retornar\n\tpub fn pintar()\n\t\tgraficos.limpiar(.Negro)\n\t\tgraficos.imagen(punto, 1, 0)\nfn inicio()\n\tpincel.ejecutar(Partida {}, 3, 2) capturar |e| imprimir(e)\n"
	if err := os.WriteFile(input, []byte(game), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"captura", input, "-o", output, "--escala", "2", "--cuadros", "3"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 6 || img.Bounds().Dy() != 4 {
		t.Fatalf("capture bounds = %v", img.Bounds())
	}
	for _, point := range [][3]int{{0, 0, 0}, {2, 0, 181}, {3, 1, 181}, {4, 3, 0}} {
		if r, _, _, _ := img.At(point[0], point[1]).RGBA(); int(r>>8) != point[2] {
			t.Fatalf("pixel %v red = %d", point, r>>8)
		}
	}
	plain := filepath.Join(dir, "plain.cometa")
	if err := os.WriteFile(plain, []byte("fn inicio() imprimir(1)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"captura", plain}); err == nil || !strings.Contains(err.Error(), "pincel.ejecutar") {
		t.Fatalf("captured a program without a game: %v", err)
	}
}
