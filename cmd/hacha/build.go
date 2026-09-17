package main

import (
	"fmt"
	"hacha/internal/gameapi"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func goExecutable() string {
	if path := os.Getenv("HACHA_GO"); path != "" {
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
	dir, err := os.MkdirTemp("", "hacha-build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	module := "module hacha-game\n\ngo 1.25.0\n\nrequire github.com/hajimehoshi/ebiten/v2 " + gameapi.EbitenVersion + "\n"
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
