package compiler

import (
	"strings"
	"testing"
)

// Run builds carry //line directives so Go panics and build errors point at
// Cometa lines; plain compilation must stay free of them.
func TestLineDirectivesOnlyForRun(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		// The interpolation forces the multi-line lowering used by games.
		"main.cometa": "fn inicio()\n\tvar lista = [1, 2, 3]\n\tvar x = lista[5]\n\timprimir(\"${x}\")\n",
	})
	plain, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "//line") {
		t.Error("compilar no debe emitir directivas //line")
	}
	run, err := CompileProjectForRun(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	// Go numbers each generated line after a directive consecutively, so a
	// statement lowered to several lines needs the directive on every one.
	if count := strings.Count(string(run), "main.cometa:3\n"); count < 2 {
		t.Errorf("la línea 3 debe repetirse en cada línea generada; apariciones: %d\n%s", count, run)
	}
	if !strings.Contains(string(run), "main.cometa:4\n") {
		t.Error("falta la directiva de la línea 4")
	}
}
