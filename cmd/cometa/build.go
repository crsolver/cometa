package main

import (
	"bytes"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"github.com/crsolver/cometa/internal/stdlib"
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
	if b := findBundle(); b != nil {
		return b.goEx, nil
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
	if err = goBuild(dir, source, nil, output); err != nil {
		return err
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

// goBuild writes source (plus optional extra files of package main) into dir
// with a fresh go.mod and compiles them to output.
func goBuild(dir string, source []byte, extra map[string]string, output string) error {
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
	for name, content := range extra {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			return err
		}
	}
	goPath, err := goExecutable()
	if err != nil {
		return err
	}
	// -s -w skips symbol/DWARF generation: ~15% faster link. Panic traces use
	// the runtime's own tables, so crash reports keep their Cometa lines.
	args := []string{"build", "-mod=mod", "-ldflags=-s -w", "-o", output, "."}
	env := append(os.Environ(), "GOWORK=off")
	if b := findBundle(); b != nil && os.Getenv("COMETA_GO") == "" {
		// -trimpath makes cached packages independent of the install directory.
		args = append(args[:1], append([]string{"-trimpath"}, args[1:]...)...)
		bundleEnv, err := b.env()
		if err != nil {
			return err
		}
		env = append(env, bundleEnv...)
	}
	cmd := exec.Command(goPath, args...)
	cmd.Dir = dir
	cmd.Env = env
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
	return nil
}

// locateGoroot points Go's type importer (used to validate generated code) at
// the user's Go installation. A released cometa carries the GOROOT of the
// machine that built it, which does not exist anywhere else.
func locateGoroot() {
	if root := os.Getenv("GOROOT"); root != "" {
		build.Default.GOROOT = root
		return
	}
	if root := build.Default.GOROOT; root != "" {
		if _, err := os.Stat(filepath.Join(root, "src", "fmt")); err == nil {
			return
		}
	}
	goEx, err := goExecutable()
	if err != nil {
		return // reported later, when the program is built
	}
	out, err := exec.Command(goEx, "env", "GOROOT").Output()
	if err != nil {
		return
	}
	if root := strings.TrimSpace(string(out)); root != "" {
		os.Setenv("GOROOT", root)
		build.Default.GOROOT = root
	}
}
