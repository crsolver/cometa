package lspserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

// Hover, definition and symbols reuse the analysis of the last edit instead
// of analyzing the project again for each request.
func TestAnalysisIsReusedUntilTheTextChanges(t *testing.T) {
	handler := NewHandler()
	harness := servertest.New(t, handler)
	uri := lsp.DocumentURI("file:///reuso.cometa")
	if err := harness.DidOpen(uri, "cometa", "fn inicio()\n\timprimir(1)\n"); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	first, _ := handler.project(uri, nil)
	if _, err := harness.Hover(uri, 1, 3); err != nil {
		t.Fatal(err)
	}
	if again, _ := handler.project(uri, nil); again != first {
		t.Fatal("la misma versión del documento se analizó dos veces")
	}
	probe := "fn inicio()\n\timprimir(2)\n"
	if override, _ := handler.project(uri, &probe); override == first {
		t.Fatal("un texto de sondeo no debe salir de la caché")
	}
	if again, _ := handler.project(uri, nil); again != first {
		t.Fatal("un texto de sondeo no debe ocupar la caché")
	}

	harness.ClearDiagnostics()
	if err := harness.DidChange(uri, 2, "fn inicio()\n\timprimir(falta)\n"); err != nil {
		t.Fatal(err)
	}
	_ = waitForDiagnostics(t, harness, uri)
	changed, err := handler.project(uri, nil)
	if changed == first || err == nil || !strings.Contains(err.Error(), "falta") {
		t.Fatalf("tras editar debe analizarse el texto nuevo: %v", err)
	}
	if err := harness.DidChangeWatchedFiles(&lsp.DidChangeWatchedFilesParams{}); err != nil {
		t.Fatal(err)
	}
	// The notification is handled in the background.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if again, _ := handler.project(uri, nil); again != changed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("un cambio en disco debe descartar el análisis")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// While someone types, only the last text of a burst is analyzed and
// published.
func TestDiagnosticsWaitForTypingToPause(t *testing.T) {
	handler := NewHandler()
	handler.Debounce = 80 * time.Millisecond
	harness := servertest.New(t, handler)
	uri := lsp.DocumentURI("file:///pausa.cometa")
	if err := harness.DidOpen(uri, "cometa", "fn inicio()\n\timprimir(1)\n"); err != nil {
		t.Fatal(err)
	}
	if diagnostics := waitForDiagnostics(t, harness, uri); len(diagnostics) != 0 {
		t.Fatalf("abrir publica de inmediato y sin errores: %+v", diagnostics)
	}
	harness.ClearDiagnostics()
	for version, text := range []string{"fn inicio()\n\timprimir(a\n", "fn inicio()\n\timprimir(ab\n", "fn inicio()\n\timprimir(abc)\n"} {
		if err := harness.DidChange(uri, version+2, text); err != nil {
			t.Fatal(err)
		}
	}
	// Nothing is published while the burst is recent.
	early, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	if _, err := harness.WaitForDiagnostics(early, uri); err == nil {
		t.Fatal("se publicaron diagnósticos antes de la pausa")
	}
	cancel()
	diagnostics := waitForDiagnostics(t, harness, uri)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, `"abc"`) {
		t.Fatalf("se esperaba solo el error del último texto: %+v", diagnostics)
	}

	// Saving does not wait.
	harness.ClearDiagnostics()
	if err := harness.DidChange(uri, 9, "fn inicio()\n\timprimir(1)\n"); err != nil {
		t.Fatal(err)
	}
	if err := harness.DidSave(uri); err != nil {
		t.Fatal(err)
	}
	immediate, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	if diagnostics, err := harness.WaitForDiagnostics(immediate, uri); err != nil || len(diagnostics) != 0 {
		t.Fatalf("guardar debe publicar de inmediato: %+v, %v", diagnostics, err)
	}
}
