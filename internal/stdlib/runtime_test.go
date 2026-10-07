package stdlib

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func goTool() string {
	exe := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	return exe
}

// The runtime lives in .txt files that Go tooling never sees. Vetting it with
// every module selected catches a type error here instead of in the first
// game that happens to use the broken function.
func TestRuntimeTypeChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles Ebitengine")
	}
	source, imports, err := Runtime(ModulePaths())
	if err != nil {
		t.Fatal(err)
	}
	var file strings.Builder
	file.WriteString("package main\n\nimport (\n")
	for _, path := range imports {
		file.WriteString("\t" + path + "\n")
	}
	// A generated program defines the data folder, the numeric conversion
	// helper (internal/codegen/numeric.go) and its own entry point.
	file.WriteString(")\n\nvar _hgdatosCarpeta = \"prueba\"\n\nfunc _hentero(v float64) int64 { return int64(v) }\n\nfunc main() {}\n\n")
	file.WriteString(source)
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, []byte(file.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	// The file is checked against this module's go.mod, like a generated game.
	// Colors are written as unkeyed literals throughout, which is a style
	// choice of the runtime rather than a defect.
	if out, err := exec.Command(goTool(), "vet", "-composites=false", path).CombinedOutput(); err != nil {
		t.Fatalf("el runtime no pasa go vet: %v\n%s", err, out)
	}
}

func TestRuntimeDependenciesAreRecorded(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "runtime_deps.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var recorded []string
	for _, spec := range file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		recorded = append(recorded, path)
	}
	var used []string
	for _, path := range runtimeImports {
		if strings.Contains(strings.SplitN(path, "/", 2)[0], ".") {
			used = append(used, path)
		}
	}
	sort.Strings(recorded)
	sort.Strings(used)
	if strings.Join(recorded, "\n") != strings.Join(used, "\n") {
		t.Fatalf("runtime_deps.go no coincide con runtimeImports\nregistradas:\n%s\nusadas:\n%s", strings.Join(recorded, "\n"), strings.Join(used, "\n"))
	}
}

// Generated games require EbitenVersion in their own go.mod, and the tests
// build them against this module's, so both must name the same release.
func TestEbitenVersionMatchesModule(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`github\.com/hajimehoshi/ebiten/v2 (\S+)`).FindSubmatch(data)
	if found == nil || string(found[1]) != EbitenVersion {
		t.Fatalf("go.mod requiere Ebitengine %s y EbitenVersion es %s", found, EbitenVersion)
	}
}
