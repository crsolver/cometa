package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"cometa/internal/stdlib"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func goExecutable() string {
	if path := os.Getenv("COMETA_GO"); path != "" {
		return path
	}
	if runtime.GOOS == "windows" {
		path := `C:\Program Files\Go\bin\go.exe`
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "go"
}

// Each build is isolated from source directories and their existing go.mod.
func buildProgram(source []byte, output string, execute bool) error {
	dir, err := os.MkdirTemp("", "cometa-build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	module := "module cometa-programa\n\ngo 1.25.0\n"
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, parser.ImportsOnly)
	if err != nil {
		return err
	}
	for _, imp := range file.Imports {
		if strings.HasPrefix(imp.Path.Value, "\"github.com/hajimehoshi/ebiten/v2") {
			module += "\nrequire github.com/hajimehoshi/ebiten/v2 " + stdlib.EbitenVersion + "\n"
			break
		}
	}
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "main.go"), source, 0600); err != nil {
		return err
	}
	name := "juego"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if execute {
		output = filepath.Join(dir, name)
	} else {
		output, err = filepath.Abs(output)
		if err != nil {
			return err
		}
	}
	cmd := exec.Command(goExecutable(), "build", "-mod=mod", "-o", output, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("no se pudo construir el juego (Go y descarga de dependencias requeridos): %w", err)
	}
	if !execute {
		return nil
	}
	game := exec.Command(output)
	game.Stdin = os.Stdin
	game.Stdout = os.Stdout
	game.Stderr = os.Stderr
	if err = game.Run(); err != nil {
		return fmt.Errorf("el juego terminó con error: %w", err)
	}
	return nil
}

// Capture runs the game with a hidden window and saves its logical screen.
func captureProgram(source []byte, output string, frames, scale int) error {
	if !strings.Contains(string(source), "func _hgcapturar(") {
		return fmt.Errorf("captura requiere un juego iniciado con pincel.ejecutar")
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	os.Remove(output)
	source = append(source, fmt.Sprintf("\nfunc init() { _hgcapturaRuta = %q; _hgcapturaCuadros = %d; _hgcapturaEscala = %d }\n", output, frames, scale)...)
	if err = buildProgram(source, "", true); err != nil {
		return err
	}
	if _, err = os.Stat(output); err != nil {
		return fmt.Errorf("el juego terminó antes de capturar %d cuadros", frames)
	}
	fmt.Println("captura guardada en", output)
	return nil
}
