package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"cometa/internal/stdlib"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func goExecutable() (string, error) {
	if path := os.Getenv("COMETA_GO"); path != "" {
		return path, nil
	}
	if runtime.GOOS == "windows" {
		path := `C:\Program Files\Go\bin\go.exe`
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	path, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("no se encontró Go. Cometa necesita Go 1.25 o superior para ejecutar y construir programas.\nInstálalo desde https://go.dev/dl/ y vuelve a abrir la terminal (o define COMETA_GO con la ruta de go)")
	}
	return path, nil
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
	goPath, err := goExecutable()
	if err != nil {
		return err
	}
	cmd := exec.Command(goPath, "build", "-mod=mod", "-o", output, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stdout
	var buildErrors bytes.Buffer
	cmd.Stderr = &buildErrors
	// The first Ebitengine build downloads and compiles dependencies; say so
	// when it is taking a while instead of looking frozen.
	slow := time.AfterFunc(4*time.Second, func() {
		fmt.Fprintln(os.Stderr, "compilando… la primera vez se descargan y compilan las dependencias del juego (puede tardar unos minutos)")
	})
	err = cmd.Run()
	slow.Stop()
	if err != nil {
		return buildFailure(buildErrors.String(), err)
	}
	if !execute {
		return nil
	}
	game := exec.Command(output)
	game.Stdin = os.Stdin
	game.Stdout = os.Stdout
	crash := &crashFilter{out: os.Stderr}
	game.Stderr = crash
	err = game.Run()
	crash.Flush()
	if err != nil {
		if crash.Report() {
			return errReported
		}
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
