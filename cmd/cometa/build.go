package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/crsolver/cometa/internal/stdlib"
)

// goExecutable finds the Go toolchain that builds programs: COMETA_GO, the
// toolchain bundled with a complete install, or the one installed on the
// machine.
func goExecutable() (string, error) {
	if path := os.Getenv("COMETA_GO"); path != "" {
		return path, nil
	}
	if b := findBundle(); b != nil {
		return b.goEx, nil
	}
	return installedGo()
}

// installedGo is looked up once: checking a candidate runs it.
var installedGo = sync.OnceValues(func() (string, error) {
	path, broken := pickGo(goCandidates(), func(path string) bool {
		out, err := exec.Command(path, "env", "GOROOT").Output()
		return err == nil && strings.TrimSpace(string(out)) != ""
	})
	if path != "" {
		return path, nil
	}
	message := "no se encontró Go. Cometa necesita Go 1.25 o superior para ejecutar y construir programas.\nInstálalo desde https://go.dev/dl/ y vuelve a abrir la terminal (o define COMETA_GO con la ruta de go)"
	if len(broken) > 0 {
		message = "se encontró Go en " + strings.Join(broken, ", ") + ", pero no funciona. Cometa necesita Go 1.25 o superior para ejecutar y construir programas.\nReinstálalo desde https://go.dev/dl/ y vuelve a abrir la terminal (o define COMETA_GO con la ruta de go)"
	}
	return "", errors.New(message)
})

// goCandidates lists every `go` on PATH in order, then the folder of the
// official Windows installer in case PATH was not updated yet.
func goCandidates() []string {
	name := "go"
	if runtime.GOOS == "windows" {
		name = "go.exe"
	}
	var candidates []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			candidates = append(candidates, path)
		}
	}
	if runtime.GOOS == "windows" {
		path := `C:\Program Files\Go\bin\go.exe`
		if _, err := os.Stat(path); err == nil {
			candidates = append(candidates, path)
		}
	}
	return candidates
}

// pickGo returns the first candidate that works. A leftover of another
// installation earlier on PATH (a go without its GOROOT, for instance) must
// not hide a good one, so the ones that fail are skipped and reported.
func pickGo(candidates []string, usable func(string) bool) (path string, broken []string) {
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if usable(candidate) {
			return candidate, broken
		}
		broken = append(broken, candidate)
	}
	return "", broken
}

// dataEnv keeps std/pincel/datos away from the player's saved data while a
// game is captured or tested: the program reads and writes folder, or an
// empty temporary folder removed by cleanup when folder is "".
func dataEnv(folder string) (env string, cleanup func(), err error) {
	cleanup = func() {}
	if folder == "" {
		if folder, err = os.MkdirTemp("", "cometa-datos-"); err != nil {
			return "", cleanup, err
		}
		temporary := folder
		cleanup = func() { os.RemoveAll(temporary) }
	} else {
		if folder, err = filepath.Abs(folder); err == nil {
			err = os.MkdirAll(folder, 0o755)
		}
		if err != nil {
			return "", cleanup, fmt.Errorf("no se pudo usar la carpeta de datos: %w", err)
		}
	}
	return "COMETA_DATOS=" + folder, cleanup, nil
}

// Each build is isolated from source directories and their existing go.mod.
// env adds variables to the environment of an executed program.
func buildProgram(source []byte, output string, execute bool, env ...string) error {
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
	game.Env = append(os.Environ(), env...)
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
