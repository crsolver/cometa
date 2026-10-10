package lspserver

import (
	"context"
	"sync"
	"time"

	"github.com/owenrumney/go-lsp/document"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/server"

	"github.com/crsolver/cometa/internal/compiler"
)

const serverVersion = "0.1.1"

type Handler struct {
	projectMu sync.Mutex
	assets    compiler.AssetCache
	projects  map[lsp.DocumentURI]*compiler.Project
	reports   map[lsp.DocumentURI]map[lsp.DocumentURI][]lsp.Diagnostic
	published map[lsp.DocumentURI]bool
	documents *document.Store
	client    *server.Client

	// analyses holds the last analysis of each document. Hover, definition and
	// symbols reuse it until a document or a watched file changes, which bumps
	// generation.
	analysisMu sync.Mutex
	generation uint64
	analyses   map[lsp.DocumentURI]analysis

	// Debounce delays the diagnostics that follow an edit, so a burst of
	// keystrokes is analyzed once. Zero publishes at once.
	Debounce time.Duration
	timerMu  sync.Mutex
	timers   map[lsp.DocumentURI]*time.Timer
}

type analysis struct {
	generation uint64
	project    *compiler.Project
	err        error
}

func NewHandler() *Handler {
	return &Handler{documents: document.NewStore()}
}

func Run(ctx context.Context) error {
	handler := NewHandler()
	handler.Debounce = 150 * time.Millisecond
	srv := server.NewServer(handler)
	return srv.Run(ctx, server.RunStdio())
}

// invalidate discards every cached analysis: some source text changed.
func (h *Handler) invalidate() {
	h.analysisMu.Lock()
	h.generation++
	h.analysisMu.Unlock()
}

// cancelPending stops the delayed diagnostics of a document, because they are
// about to be published right away or the document is gone.
func (h *Handler) cancelPending(uri lsp.DocumentURI) {
	h.timerMu.Lock()
	if timer := h.timers[uri]; timer != nil {
		timer.Stop()
		delete(h.timers, uri)
	}
	h.timerMu.Unlock()
}

func (h *Handler) SetClient(client *server.Client) {
	h.client = client
}

func (h *Handler) Initialize(_ context.Context, _ *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	return &lsp.InitializeResult{
		ServerInfo: &lsp.ServerInfo{Name: "cometa", Version: serverVersion},
		Capabilities: lsp.ServerCapabilities{
			CompletionProvider: &lsp.CompletionOptions{TriggerCharacters: []string{".", "@"}},
		},
	}, nil
}

func (h *Handler) Shutdown(_ context.Context) error { return nil }

func (h *Handler) DidOpen(ctx context.Context, params *lsp.DidOpenTextDocumentParams) error {
	if params.TextDocument.LanguageID != "" && params.TextDocument.LanguageID != "cometa" {
		return nil
	}
	if _, err := h.documents.Open(params); err != nil {
		return err
	}
	h.invalidate()
	h.cancelPending(params.TextDocument.URI)
	return h.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (h *Handler) DidChange(ctx context.Context, params *lsp.DidChangeTextDocumentParams) error {
	if _, err := h.documents.Change(params); err != nil {
		return err
	}
	h.invalidate()
	uri := params.TextDocument.URI
	if h.Debounce <= 0 {
		return h.publishDiagnostics(ctx, uri)
	}
	h.timerMu.Lock()
	defer h.timerMu.Unlock()
	if h.timers == nil {
		h.timers = map[lsp.DocumentURI]*time.Timer{}
	}
	if timer := h.timers[uri]; timer != nil {
		timer.Stop()
	}
	// The request that scheduled this is long answered, so its context is not
	// the one to publish with.
	h.timers[uri] = time.AfterFunc(h.Debounce, func() {
		h.cancelPending(uri)
		_ = h.publishDiagnostics(context.Background(), uri)
	})
	return nil
}

func (h *Handler) DidSave(ctx context.Context, params *lsp.DidSaveTextDocumentParams) error {
	h.cancelPending(params.TextDocument.URI)
	return h.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (h *Handler) DidClose(ctx context.Context, params *lsp.DidCloseTextDocumentParams) error {
	h.documents.Close(params)
	h.invalidate()
	h.cancelPending(params.TextDocument.URI)
	if h.client == nil {
		return nil
	}
	return h.publishProjectDiagnostics(ctx, params.TextDocument.URI)
}
