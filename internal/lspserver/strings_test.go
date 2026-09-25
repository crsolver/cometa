package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestStringMethodCompletionAndHover(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///cadenas.hacha")
	source := "fn inicio()\n\tvar texto = \"hola\"\n\ttexto.\n"
	if err := harness.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	completion, err := harness.Completion(uri, 2, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(completion.Items) != 13 {
		t.Fatalf("completion = %+v", completion.Items)
	}
	labels := make([]string, len(completion.Items))
	for i, item := range completion.Items {
		labels[i] = item.Label
	}
	if joined := strings.Join(labels, ","); !strings.Contains(joined, "subcadena") || !strings.Contains(joined, "mayusculas") {
		t.Fatalf("missing methods: %s", joined)
	}

	source = "fn inicio()\n\tvar texto = \"hola\"\n\timprimir(texto.longitud())\n"
	harness = servertest.New(t, NewHandler())
	uri = lsp.DocumentURI("file:///cadenas-hover.hacha")
	if err := harness.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	hover, err := harness.Hover(uri, 2, 18)
	if err != nil || hover == nil {
		t.Fatalf("hover = %+v, %v", hover, err)
	}
	if value := hover.Contents.Value(); !strings.Contains(value, "fn longitud() entero") || !strings.Contains(value, "puntos de código Unicode") {
		t.Fatalf("hover = %q", value)
	}
}

func TestStringMethodCompletionOnLiteral(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///cadena-literal.hacha")
	source := "fn inicio()\n\t\"hola\".\n"
	if err := harness.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	completion, err := harness.Completion(uri, 1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(completion.Items) != 13 {
		t.Fatalf("completion = %+v", completion.Items)
	}
}

func TestHoverInsideInterpolationUsesEmbeddedPositions(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///interpolacion.hacha")
	source := "fn inicio()\n\tvar nombre = \"Ana\"\n\timprimir(\"Hola ${nombre}\")\n"
	if err := harness.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	hover, err := harness.Hover(uri, 2, 20)
	if err != nil || hover == nil {
		t.Fatalf("hover = %+v, %v", hover, err)
	}
	if value := hover.Contents.Value(); !strings.Contains(value, "var nombre cadena") {
		t.Fatalf("hover = %q", value)
	}
}
