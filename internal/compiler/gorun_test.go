package compiler

import (
	"bytes"
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// update rewrites the golden files instead of comparing against them:
//
//	go test ./internal/compiler -run Golden -update
var update = flag.Bool("update", false, "rewrite golden files")

func goTool() string {
	exe := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	return exe
}

// goRun writes generated as the only file of a program and runs it with
// input on stdin. Stdout and stderr are returned separately: graphics drivers
// may print warnings to stderr. Tests run in parallel, so the limit allows for
// several builds competing for the machine.
func goRun(t *testing.T, generated []byte, input string) (stdout, stderr string, err error) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(file, generated, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	// Built into the test's own folder rather than with `go run`, which keeps
	// every executable in the build cache: a few megabytes per program, for
	// each of the hundreds of programs here, each time the generator changes.
	program := filepath.Join(filepath.Dir(file), "programa.exe")
	if output, err := exec.CommandContext(ctx, goTool(), "build", "-o", program, file).CombinedOutput(); err != nil {
		return "", string(output), err
	}
	cmd := exec.CommandContext(ctx, program)
	cmd.Stdin = strings.NewReader(input)
	var out, errs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errs
	err = cmd.Run()
	return strings.ReplaceAll(out.String(), "\r\n", "\n"), errs.String(), err
}

func runCometa(t *testing.T, source, want string) []byte {
	t.Helper()
	generated, err := Compile("enum.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return runGeneratedGo(t, generated, want)
}

func runGeneratedGo(t *testing.T, generated []byte, want string) []byte {
	t.Helper()
	runGeneratedGoWithInput(t, generated, "", want)
	return generated
}

// runGeneratedGoWithInput is runGeneratedGo with the given text on stdin.
func runGeneratedGoWithInput(t *testing.T, generated []byte, input, want string) {
	t.Helper()
	out, stderr, err := goRun(t, generated, input)
	if err != nil {
		t.Fatalf("generated program failed: %v\n%s%s\n%s", err, out, stderr, generated)
	}
	if out != want {
		t.Fatalf("output = %q, want %q\n%s", out, want, generated)
	}
}

// runGoExpectingFailure runs a generated program that must fail and returns
// everything it printed.
func runGoExpectingFailure(t *testing.T, generated []byte) string {
	t.Helper()
	out, stderr, err := goRun(t, generated, "")
	if err == nil {
		t.Fatalf("el programa debía fallar\n%s%s", out, stderr)
	}
	return out + stderr
}

// checkGolden compares got with a golden file, or rewrites it with -update.
func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	got = bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n"))
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))) {
		t.Fatalf("generated Go differs from %s (rerun with -update to accept)\n--- got ---\n%s", path, got)
	}
}
