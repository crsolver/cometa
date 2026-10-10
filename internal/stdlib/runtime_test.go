package stdlib

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
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

// go vet already checks package rt as a whole. This checks what a program
// receives: with every module selected, the declarations and the imports
// picked for it must make a complete Go program.
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
	// The color constants of the catalog are unkeyed literals of an aliased
	// type, which vet reports and Go accepts.
	if out, err := exec.Command(goTool(), "vet", "-composites=false", path).CombinedOutput(); err != nil {
		t.Fatalf("el runtime no pasa go vet: %v\n%s", err, out)
	}
}

// Every file of package rt must be listed in runtimeFiles, or its
// declarations would never reach a program.
func TestRuntimeFilesAreListed(t *testing.T) {
	entries, err := runtimeFS.ReadDir("rt")
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, entry := range entries {
		if name := entry.Name(); name != "doc.go" && !strings.HasSuffix(name, "_test.go") {
			found = append(found, name)
		}
	}
	listed := append([]string(nil), runtimeFiles...)
	sort.Strings(listed)
	if strings.Join(found, " ") != strings.Join(listed, " ") {
		t.Fatalf("runtimeFiles = %v, package rt has %v", listed, found)
	}
}

// A module that needs nothing from Ebitengine must not bring it into a program.
func TestConsoleModulesStayWindowFree(t *testing.T) {
	for _, module := range []string{"std/mate", "std/azar", "std/consola", "std/mate/curvas", "std/mate/ruido", "std/pruebas", "std/pincel/datos"} {
		_, imports, err := Runtime([]string{module})
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range imports {
			if strings.Contains(imported, "ebiten") {
				t.Errorf("%s imports %s", module, imported)
			}
		}
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
