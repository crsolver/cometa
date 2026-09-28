package lspserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/server"
	"github.com/owenrumney/go-lsp/servertest"

	"cometa/internal/ast"
	"cometa/internal/compiler"
	"cometa/internal/sema"
)

var (
	_ server.LifecycleHandler        = (*Handler)(nil)
	_ server.ClientHandler           = (*Handler)(nil)
	_ server.TextDocumentSyncHandler = (*Handler)(nil)
	_ server.TextDocumentSaveHandler = (*Handler)(nil)
	_ server.DocumentSymbolHandler   = (*Handler)(nil)
	_ server.CompletionHandler       = (*Handler)(nil)
	_ server.HoverHandler            = (*Handler)(nil)
)

func TestInitializeAdvertisesCometaServer(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	if harness.InitResult.ServerInfo == nil || harness.InitResult.ServerInfo.Name != "cometa" {
		t.Fatalf("unexpected server info: %+v", harness.InitResult.ServerInfo)
	}
	if harness.InitResult.Capabilities.TextDocumentSync == nil {
		t.Fatal("text document synchronization was not advertised")
	}
	if harness.InitResult.Capabilities.DocumentSymbolProvider == nil {
		t.Fatal("document symbols were not advertised")
	}
	if provider := harness.InitResult.Capabilities.CompletionProvider; provider == nil || len(provider.TriggerCharacters) != 2 || provider.TriggerCharacters[0] != "." || provider.TriggerCharacters[1] != "@" {
		t.Fatalf("member completion was not advertised: %+v", provider)
	}
	if provider := harness.InitResult.Capabilities.HoverProvider; provider == nil || !*provider {
		t.Fatal("hover was not advertised")
	}
}

func TestHoverShowsInferredVariablesAndFunctionDocumentation(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///hover.cometa")
	source := "tipo Usuario\n\tnombre cadena\n\n// Procesa una lista de usuarios.\n// Conserva el orden original.\nfn procesar_lista(usuarios [Usuario])\n\timprimir(usuarios)\n\nfn inicio()\n\tvar usuario1 = Usuario {nombre: \"uno\"}\n\tvar usuario2 = Usuario {nombre: \"dos\"}\n\tvar lista = [usuario1, usuario2]\n\tvar x = lista[0]\n\tprocesar_lista(lista)\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	tests := []struct {
		name   string
		line   int
		char   int
		value  string
		range_ lsp.Range
	}{
		{
			name:   "parameter",
			line:   5,
			char:   20,
			value:  "```cometa\nvar usuarios [Usuario]\n```",
			range_: lsp.Range{Start: lsp.Position{Line: 5, Character: 18}, End: lsp.Position{Line: 5, Character: 26}},
		},
		{
			name:   "inferred list use",
			line:   12,
			char:   10,
			value:  "```cometa\nvar lista [Usuario]\n```",
			range_: lsp.Range{Start: lsp.Position{Line: 12, Character: 9}, End: lsp.Position{Line: 12, Character: 14}},
		},
		{
			name:   "inferred indexed value declaration",
			line:   12,
			char:   5,
			value:  "```cometa\nvar x Usuario\n```",
			range_: lsp.Range{Start: lsp.Position{Line: 12, Character: 5}, End: lsp.Position{Line: 12, Character: 6}},
		},
		{
			name:   "function call with documentation",
			line:   13,
			char:   3,
			value:  "```cometa\nfn procesar_lista(\n\tusuarios [Usuario],\n)\n```\n\nProcesa una lista de usuarios.\nConserva el orden original.",
			range_: lsp.Range{Start: lsp.Position{Line: 13, Character: 1}, End: lsp.Position{Line: 13, Character: 15}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hover, err := harness.Hover(uri, test.line, test.char)
			if err != nil {
				t.Fatal(err)
			}
			if hover == nil {
				t.Fatal("hover is nil")
			}
			if got := hover.Contents.Value(); got != test.value {
				t.Fatalf("hover contents = %q, want %q", got, test.value)
			}
			if hover.Range == nil || *hover.Range != test.range_ {
				t.Fatalf("hover range = %+v, want %+v", hover.Range, test.range_)
			}
		})
	}
}

func TestHoverTraversesPositionalStructValues(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///hover-positional.cometa")
	source := "tipo Punto\n\tx entero\nfn inicio()\n\tvar n = 1\n\tvar punto = Punto {n}\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
	hover, err := harness.Hover(uri, 4, 20)
	if err != nil || hover == nil || !strings.Contains(hover.Contents.Value(), "var n entero") {
		t.Fatalf("hover = %+v, %v; want inferred positional value", hover, err)
	}
}

func TestHoverIgnoresWhitespaceAndDetachedComments(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///hover-sin-documentacion.cometa")
	source := "// Este comentario no está adjunto.\n\nfn saludo(nombre cadena)\n\timprimir(nombre)\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)

	hover, err := harness.Hover(uri, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil || hover.Contents.Value() != "```cometa\nfn saludo(nombre cadena)\n```" {
		t.Fatalf("unexpected function hover: %+v", hover)
	}
	hover, err = harness.Hover(uri, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if hover != nil {
		t.Fatalf("expected no hover on whitespace, got %+v", hover)
	}
}

func TestCompletesFieldsAndMethodsForVariable(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///completar.cometa")
	source := "tipo Usuario\n\tnombre cadena\n\tfn activar(valor bool)\n\t\timprimir(valor)\nfn inicio()\n\tvar usuario = Usuario {nombre: \"andres\"}\n\tusuario.\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	completion, err := harness.Completion(uri, 6, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(completion.Items) != 2 {
		t.Fatalf("completion items = %+v", completion.Items)
	}
	if completion.Items[0].Label != "nombre" || completion.Items[0].Kind == nil || *completion.Items[0].Kind != lsp.CompletionItemKindField {
		t.Fatalf("field completion = %+v", completion.Items[0])
	}
	if completion.Items[1].Label != "activar(...)" || completion.Items[1].Kind == nil || *completion.Items[1].Kind != lsp.CompletionItemKindMethod || completion.Items[1].InsertText != "activar" {
		t.Fatalf("method completion = %+v", completion.Items[1])
	}
	if completion.Items[1].Detail != "" {
		t.Fatalf("method detail should be empty, signature belongs in Documentation: %+v", completion.Items[1])
	}
	if doc := completion.Items[1].Documentation; doc == nil || doc.Kind != lsp.Markdown || !strings.Contains(doc.Value, "```cometa\nfn activar(valor bool)\n```") {
		t.Fatalf("method documentation = %+v", doc)
	}
}

func TestCompletesReceiverFieldsAndMethodsAfterAt(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///completar-receptor.cometa")
	source := "tipo Usuario\n\tnombre cadena\n\tfn activar(valor bool)\n\t\t@\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	completion, err := harness.Completion(uri, 3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(completion.Items) != 2 {
		t.Fatalf("completion items = %+v", completion.Items)
	}
	if completion.Items[0].Label != "nombre" || completion.Items[0].Kind == nil || *completion.Items[0].Kind != lsp.CompletionItemKindField {
		t.Fatalf("field completion = %+v", completion.Items[0])
	}
	if completion.Items[1].Label != "activar(...)" || completion.Items[1].Kind == nil || *completion.Items[1].Kind != lsp.CompletionItemKindMethod || completion.Items[1].InsertText != "activar" {
		t.Fatalf("method completion = %+v", completion.Items[1])
	}
}

func TestCompletesReceiverMembersInInlineMethod(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///completar-receptor-inline.cometa")
	source := "tipo Usuario\n\tnombre cadena\n\tfn activar() @nom\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	completion, err := harness.Completion(uri, 2, 18)
	if err != nil {
		t.Fatal(err)
	}
	if len(completion.Items) != 2 || completion.Items[0].Label != "nombre" || completion.Items[1].Label != "activar()" {
		t.Fatalf("completion items = %+v", completion.Items)
	}
}

func TestPublishesAndClearsDiagnosticsOnChange(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///diagnostico.cometa")
	if err := harness.DidOpen(uri, "cometa", "fn inicio()\n    imprimir(\"mal\")\n"); err != nil {
		t.Fatal(err)
	}
	diagnostics := waitForDiagnostics(t, harness, uri)
	if len(diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diagnostics))
	}
	diagnostic := diagnostics[0]
	if diagnostic.Source != "cometa" || diagnostic.Message == "" {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}
	if diagnostic.Range.Start.Line != 1 || diagnostic.Range.Start.Character != 0 {
		t.Fatalf("unexpected diagnostic range: %+v", diagnostic.Range)
	}
	if diagnostic.Range.End.Line != 1 || diagnostic.Range.End.Character != 4 {
		t.Fatalf("diagnostic did not cover the invalid indentation: %+v", diagnostic.Range)
	}

	harness.ClearDiagnostics()
	if err := harness.DidChange(uri, 2, "fn inicio()\n\timprimir(\"bien\")\n"); err != nil {
		t.Fatal(err)
	}
	diagnostics = waitForDiagnostics(t, harness, uri)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics were not cleared: %+v", diagnostics)
	}
}

func TestDocumentSymbolsAreHierarchical(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///usuario.cometa")
	source := "tipo Usuario\n\tnombre cadena\n\tfn saludar(mensaje cadena)\n\t\timprimir(mensaje)\n\nfn inicio()\n\timprimir(\"hola\")\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	symbols, err := harness.DocumentSymbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 2 {
		t.Fatalf("got %d top-level symbols, want 2: %+v", len(symbols), symbols)
	}
	if symbols[0].Name != "Usuario" || symbols[0].Kind != lsp.SymbolKindStruct {
		t.Fatalf("unexpected type symbol: %+v", symbols[0])
	}
	if len(symbols[0].Children) != 2 || symbols[0].Children[0].Kind != lsp.SymbolKindField || symbols[0].Children[1].Kind != lsp.SymbolKindMethod {
		t.Fatalf("unexpected type children: %+v", symbols[0].Children)
	}
	if symbols[1].Name != "inicio" || symbols[1].Kind != lsp.SymbolKindFunction {
		t.Fatalf("unexpected function symbol: %+v", symbols[1])
	}
}

func TestDiagnosticColumnsUseUTF16(t *testing.T) {
	err := &sema.Error{Pos: ast.Pos{Line: 1, Column: 2}, Message: "error"}
	diagnostic := diagnosticFromError(err, "😀x")
	if diagnostic.Range.Start.Character != 2 {
		t.Fatalf("UTF-16 character = %d, want 2", diagnostic.Range.Start.Character)
	}
	if diagnostic.Range.End.Character != 3 {
		t.Fatalf("UTF-16 end character = %d, want 3", diagnostic.Range.End.Character)
	}
}

func TestDiagnosticCoversRelevantToken(t *testing.T) {
	source := "fn inicio()\n\tvar usuario = Usario{}\n"
	_, _, err := compiler.Analyze("diagnostico.cometa", []byte(source))
	if err == nil {
		t.Fatal("expected unknown-type diagnostic")
	}
	diagnostic := diagnosticFromError(err, source)
	want := lsp.Range{
		Start: lsp.Position{Line: 1, Character: 15},
		End:   lsp.Position{Line: 1, Character: 21},
	}
	if diagnostic.Range != want {
		t.Fatalf("diagnostic range = %+v, want %+v", diagnostic.Range, want)
	}
}

// completionText returns the signature text carried by a completion item:
// Detail for non-function items, or the raw signature unwrapped from the
// Documentation code block for function/method items (which leave Detail
// empty to avoid showing the signature twice in clients that render Detail
// as the doc panel's header).
func completionText(item lsp.CompletionItem) string {
	if item.Detail != "" {
		return item.Detail
	}
	if item.Documentation == nil {
		return ""
	}
	value := strings.TrimPrefix(item.Documentation.Value, "```cometa\n")
	if end := strings.Index(value, "\n```"); end >= 0 {
		return value[:end]
	}
	return value
}

// flattenWrapped undoes wrapSignature's line breaks so tests can match a
// signature substring regardless of whether it wrapped onto multiple lines.
func flattenWrapped(text string) string {
	text = strings.ReplaceAll(text, "(\n\t", "(")
	text = strings.ReplaceAll(text, ",\n\t", ", ")
	text = strings.ReplaceAll(text, ",\n)", ")")
	return text
}

func waitForDiagnostics(t *testing.T, harness *servertest.Harness, uri lsp.DocumentURI) []lsp.Diagnostic {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	diagnostics, err := harness.WaitForDiagnostics(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	return diagnostics
}
