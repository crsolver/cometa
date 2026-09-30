package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestBuiltinMethodCompletion(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         string
	}{
		{"entero", "fn inicio()\n\tvar n = 3\n\tn.\n", "formato(...)"},
		{"decimal", "fn inicio()\n\tvar precio = 3.5\n\tprecio.\n", "formato(...)"},
		{"tipo", "tipo Perro\n\tnombre cadena\nfn inicio()\n\tvar perro = Perro {}\n\tperro.\n", "copiar()"},
		{"copiar propio", "tipo Perro\n\tnombre cadena\n\tfn copiar() entero\n\t\t1\nfn inicio()\n\tvar perro = Perro {}\n\tperro.\n", "copiar("},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///incorporados-completar.cometa")
			if err := h.DidOpen(uri, "cometa", tc.source); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, h, uri)
			lines := strings.Split(strings.TrimSuffix(tc.source, "\n"), "\n")
			list, err := h.Completion(uri, len(lines)-1, len(lines[len(lines)-1]))
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, item := range list.Items {
				if strings.HasPrefix(item.Label, strings.TrimSuffix(tc.want, "...)")) || item.Label == tc.want {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("want exactly one %q, got %d in %+v", tc.want, count, list.Items)
			}
		})
	}
}
