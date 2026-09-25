package compiler

import (
	"regexp"
	"strings"
	"testing"
)

func TestUnusedBindingsPreserveEffects(t *testing.T) {
	for _, flow := range []bool{false, true} {
		source := "fn efecto() entero\n\timprimir(7)\n\t3\nfn inicio()\n\tvar descartado = efecto()\n\tvar muerto = 42\n\tvar copia = muerto\n\tvar usado = 5\n\timprimir(usado)\n"
		if flow {
			source += "\tvar opcional entero? = .Ninguno\n"
		}
		got, err := Compile("unused.cometa", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"descartado", "muerto", "copia", "opcional"} {
			if strings.Contains(string(got), name) {
				t.Fatalf("unused binding %s remains:\n%s", name, got)
			}
		}
		if regexp.MustCompile(`(?m)^\s*_ = [a-zA-Z_][a-zA-Z_0-9]*\s*$`).Match(got) {
			t.Fatalf("dummy use remains:\n%s", got)
		}
		runGeneratedGo(t, got, "7\n5\n")
	}
}
