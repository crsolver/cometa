package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"strings"
	"testing"
)

func TestWrapperCompletion(t *testing.T) {
	for _, tc := range []struct {
		source string
		labels string
	}{
		{"fn f() entero?\n\t.^\n", "Ninguno,Alguno"},
		{"fn f() entero!\n\t.^\n", "Ok,Error"},
		{"fn f() !\n\t.^\n", "Ok,Error"},
		{"fn f(n entero?) entero\n\tcasos n\n\t\t.^\n", "Ninguno,Alguno"},
		{"fn f(n entero!) entero\n\tcasos n\n\t\t.^\n", "Ok,Error"},
		{"tipo U\n\tnombre cadena\nfn f(u U?)\n\tsi u |v|\n\t\timprimir(v.^)\n", "nombre"},
		{"tipo U\n\tnombre cadena\nfn f(u U?)\n\timprimir(u.^)\n", ""},
	} {
		mark := strings.Index(tc.source, "^")
		before := tc.source[:mark]
		line := strings.Count(before, "\n")
		column := utf16Length(before[strings.LastIndex(before, "\n")+1:])
		h := servertest.New(t, NewHandler())
		uri := lsp.DocumentURI("file:///wrappers.cometa")
		if err := h.DidOpen(uri, "cometa", strings.Replace(tc.source, "^", "", 1)); err != nil {
			t.Fatal(err)
		}
		_ = waitForDiagnostics(t, h, uri)
		list, err := h.Completion(uri, line, column)
		if err != nil {
			t.Fatal(err)
		}
		var labels []string
		for _, item := range list.Items {
			labels = append(labels, item.Label)
		}
		if got := strings.Join(labels, ","); got != tc.labels {
			t.Errorf("%s: got %q, want %q", tc.source, got, tc.labels)
		}
	}
}

func TestWrapperTypeDisplay(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"fn f(u entero?) entero! u o 1\n", "fn f(u entero?) entero!"},
		{"fn f() !bool .Ok\n", "fn f() !bool"},
		{"fn f() (entero!)? .Ninguno\n", "fn f() (entero!)?"},
	} {
		h := servertest.New(t, NewHandler())
		uri := lsp.DocumentURI("file:///wrapper_hover.cometa")
		if err := h.DidOpen(uri, "cometa", tc.source); err != nil {
			t.Fatal(err)
		}
		_ = waitForDiagnostics(t, h, uri)
		hover, err := h.Hover(uri, 0, 3)
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil || !strings.Contains(hover.Contents.Value(), tc.want) {
			t.Fatalf("hover = %+v, want %q", hover, tc.want)
		}
	}
}

func TestUnusedOptionalLocalsHaveNoDiagnostics(t *testing.T) {
	source := "tipo Usuario\n\tnombre cadena\n\tmascota cadena?\nfn nulable()\n\tvar usuario = Usuario {mascota: \"hola\"}\n\tvar mascota = usuario.mascota\n\tvar talvez_usuario Usuario? = .Alguno({})\n"
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///unused_optional.cometa")
	if err := h.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	diagnostics := waitForDiagnostics(t, h, uri)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
}
