package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestMapCompletion(t *testing.T) {
	for _, source := range []string{
		"fn inicio()\n\tvar m = [\"a\": 1]\n\tm.\n",
		"tipo Caja\n\tm [cadena: entero]\nfn inicio()\n\tvar c = Caja {}\n\tc.m.\n",
	} {
		h := servertest.New(t, NewHandler())
		uri := lsp.DocumentURI("file:///mapas-completar.cometa")
		if err := h.DidOpen(uri, "cometa", source); err != nil {
			t.Fatal(err)
		}
		_ = waitForDiagnostics(t, h, uri)
		lines := strings.Split(strings.TrimSuffix(source, "\n"), "\n")
		list, err := h.Completion(uri, len(lines)-1, len(lines[len(lines)-1]))
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Items) != 9 {
			t.Fatalf("completion: %+v", list.Items)
		}
		for _, item := range list.Items {
			if item.Label == "obtener" && !strings.Contains(item.Detail, "entero?") {
				t.Fatalf("signature: %s", item.Detail)
			}
		}
	}
}

func TestMapHover(t *testing.T) {
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///mapas-hover.cometa")
	source := "fn inicio()\n\tvar m = [\"a\": 1]\n\timprimir(m.obtener(\"a\") o 0)\n\trepetir (m) |valor, clave|\n\t\timprimir(clave)\n"
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	if ds := waitForDiagnostics(t, h, uri); len(ds) != 0 {
		t.Fatalf("diagnostics: %+v", ds)
	}
	for _, tc := range []struct {
		line, char int
		want       string
	}{{2, 14, "fn obtener(clave cadena) entero?"}, {3, 21, "cadena"}} {
		hover, err := h.Hover(uri, tc.line, tc.char)
		if err != nil || hover == nil {
			t.Fatalf("hover: %v %v", hover, err)
		}
		if !strings.Contains(hover.Contents.Value(), tc.want) {
			t.Fatalf("hover: %s, want %s", hover.Contents.Value(), tc.want)
		}
	}
}
