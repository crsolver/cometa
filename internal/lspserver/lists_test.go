package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestListMethodCompletionForVariablesFieldsAndIndexes(t *testing.T) {
	for _, tc := range []struct {
		name, source    string
		line, character int
	}{
		{"field", "tipo Caja\n\tvalores [num]\nfn inicio()\n\tvar c = Caja {valores: [1]}\n\tc.valores.\n", 4, 11},
		{"index", "fn inicio()\n\tvar matrices = [[1]]\n\tmatrices[0].\n", 2, 13},
	} {
		t.Run(tc.name, func(t *testing.T) {
			harness := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///listas-completar-" + tc.name + ".hacha")
			if err := harness.DidOpen(uri, "hacha", tc.source); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, harness, uri)
			completion, err := harness.Completion(uri, tc.line, tc.character)
			if err != nil {
				t.Fatal(err)
			}
			if len(completion.Items) != 13 {
				t.Fatalf("completion at %d:%d = %+v", tc.line, tc.character, completion.Items)
			}
			labels := make([]string, len(completion.Items))
			for i, item := range completion.Items {
				labels[i] = item.Label
			}
			joined := strings.Join(labels, ",")
			if !strings.Contains(joined, "agregar") || !strings.Contains(joined, "buscar_indice") || !strings.Contains(joined, "invertir") {
				t.Fatalf("missing list methods: %s", joined)
			}
		})
	}
}

func TestListMethodHoverShowsConcreteSignatureAndMutation(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///listas-hover.hacha")
	source := "fn inicio()\n\tvar valores = [1]\n\tvalores.agregar(2)\n"
	if err := harness.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	hover, err := harness.Hover(uri, 2, 11)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil {
		t.Fatal("hover is nil")
	}
	value := hover.Contents.Value()
	if !strings.Contains(value, "fn agregar(valor num)") || !strings.Contains(value, "actualiza esta variable de lista") {
		t.Fatalf("unexpected hover: %q", value)
	}
}
