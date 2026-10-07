package compiler

import (
	"strings"
	"testing"
)

// Run builds carry //line directives so Go panics and build errors point at
// Cometa lines; plain compilation must stay free of them.
func TestLineDirectivesOnlyForRun(t *testing.T) {
	t.Parallel()
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

// A panic trace has no columns, so indexes and divisions carry theirs in the
// file name of an inline directive and hand the position back afterwards.
func TestLineDirectivesCarryColumns(t *testing.T) {
	t.Parallel()
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa": "fn inicio()\n\tvar lista = [1, 2, 3]\n\tvar cero = 0\n\tlista[0] = lista[1] / cero + lista[2] % 2\n\timprimir(lista)\n",
	})
	run, err := CompileProjectForRun(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	code := string(run)
	for _, column := range []string{"#7:4", "#18:4", "#22:4", "#36:4", "#40:4"} {
		if !strings.Contains(code, "main.cometa"+column+"*/") {
			t.Errorf("falta la posición %s\n%s", column, code)
		}
	}
	if strings.ContainsAny(code, "\x00\x01") {
		t.Error("quedaron marcas de posición sin resolver")
	}
	// Each of the five marked expressions hands the position back to its
	// statement afterwards.
	if strings.Count(code, "main.cometa:4*/") != 5 {
		t.Errorf("las directivas no devuelven la posición\n%s", code)
	}
	plain, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "/*line") {
		t.Error("compilar no debe emitir directivas de columna")
	}
}
