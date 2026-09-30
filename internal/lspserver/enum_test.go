package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
	"strings"
	"testing"
)

const enumSource = "tipo Boton\n\tcaracter cadena\nenum Evento\n\tCargar\n\tBoton Boton\n\tTexto cadena\n"

func TestEnumCompletion(t *testing.T) {
	tests := []struct {
		name, source, marker string
		labels               []string
	}{
		{"enum variants", enumSource + "fn inicio()\n\tvar e = Evento.\n", "Evento.", []string{"Cargar", "Boton", "Texto"}},
		{"enum after cursor", "fn inicio()\n\tvar e = Evento.\n" + enumSource, "Evento.", []string{"Cargar", "Boton", "Texto"}},
		{"payload inline", enumSource + "fn inicio()\n\tcasos Evento.Cargar |e|\n\t\t.Boton => imprimir(e.)\n", "e.", []string{"caracter", "copiar()"}},
		{"payload multiline", enumSource + "fn inicio()\n\tcasos Evento.Cargar |e|\n\t\t.Boton =>\n\t\t\timprimir(e.)\n", "e.", []string{"caracter", "copiar()"}},
		{"payload value match", enumSource + "fn f(evento Evento) cadena\n\tcasos evento |e|\n\t\t.Boton => e.\n", "e.", []string{"caracter", "copiar()"}},
		{"payload unavailable", enumSource + "fn inicio()\n\tcasos Evento.Cargar |e|\n\t\t.Cargar => imprimir(e.)\n", "e.", nil},
		{"wildcard unavailable", enumSource + "fn inicio()\n\tcasos Evento.Cargar |e|\n\t\t_ => imprimir(e.)\n", "e.", nil},
		{"enum instance", enumSource + "fn inicio()\n\tvar e = Evento.Cargar\n\te.\n", "e.", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			harness := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///enum-completion.cometa")
			if err := harness.DidOpen(uri, "cometa", tt.source); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, harness, uri)
			line, column := 0, 0
			for i, value := range strings.Split(tt.source, "\n") {
				if at := strings.LastIndex(value, tt.marker); at >= 0 {
					line, column = i, utf16Length(value[:at+len(tt.marker)])
				}
			}
			list, err := harness.Completion(uri, line, column)
			if err != nil {
				t.Fatal(err)
			}
			if len(list.Items) != len(tt.labels) {
				t.Fatalf("items = %+v, want %v", list.Items, tt.labels)
			}
			for i, label := range tt.labels {
				if list.Items[i].Label != label {
					t.Errorf("item %d = %q, want %q", i, list.Items[i].Label, label)
				}
			}
		})
	}
}

func TestEnumSymbolsHoverAndDiagnostics(t *testing.T) {
	harness := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///enum.cometa")
	source := enumSource + "fn f(evento Evento) cadena\n\tcasos evento |e|\n\t\t.Boton =>\n\t\t\tvar texto = e.caracter\n\t\t\ttexto\n\t\t_ => \"otro\"\n"
	if err := harness.DidOpen(uri, "cometa", source); err != nil {
		t.Fatal(err)
	}
	if d := waitForDiagnostics(t, harness, uri); len(d) != 0 {
		t.Fatalf("diagnostics: %+v", d)
	}
	symbols, err := harness.DocumentSymbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 3 || symbols[1].Kind != lsp.SymbolKindEnum || len(symbols[1].Children) != 3 || symbols[1].Children[1].Kind != lsp.SymbolKindEnumMember {
		t.Fatalf("symbols = %+v", symbols)
	}
	for _, tt := range []struct {
		line, col int
		want      string
	}{
		{2, 6, "enum Evento"}, {4, 2, "Evento.Boton(Boton)"}, {8, 3, "Evento.Boton(Boton)"},
		{9, 15, "var e Boton"}, {9, 8, "var texto cadena"},
	} {
		hover, err := harness.Hover(uri, tt.line, tt.col)
		if err != nil {
			t.Fatal(err)
		}
		if hover == nil || !strings.Contains(hover.Contents.Value(), tt.want) {
			t.Fatalf("hover at %d:%d = %+v, want %q", tt.line, tt.col, hover, tt.want)
		}
	}
	harness.ClearDiagnostics()
	invalid := strings.Replace(source, "\t\t_ => \"otro\"\n", "", 1)
	if err := harness.DidChange(uri, 2, invalid); err != nil {
		t.Fatal(err)
	}
	if d := waitForDiagnostics(t, harness, uri); len(d) != 1 || !strings.Contains(d[0].Message, "faltan: Cargar, Texto") {
		t.Fatalf("coverage diagnostics: %+v", d)
	}
	harness.ClearDiagnostics()
	if err := harness.DidChange(uri, 3, source); err != nil {
		t.Fatal(err)
	}
	if d := waitForDiagnostics(t, harness, uri); len(d) != 0 {
		t.Fatalf("diagnostic not cleared: %+v", d)
	}
}
