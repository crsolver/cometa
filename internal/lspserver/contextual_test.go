package lspserver

import (
	"strings"
	"testing"

	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestContextualCompletion(t *testing.T) {
	const decl = "enum IP\n\tV4\n\tV6\n\tOtro IP\nfn f(ip IP) cadena\n\t\"ok\"\nfn g(n cadena, ip IP)\n\timprimir(n)\n"
	for _, tt := range []struct {
		name, source string
		want         bool
	}{
		{"call", "fn inicio()\n\timprimir(f(.|))\n", true},
		{"named call", "fn inicio()\n\tg(ip = .|, n = \"ok\")\n", true},
		{"variadic call", "fn v(ips ...IP) imprimir(ips)\nfn inicio()\n\tv(IP.V4, .|)\n", true},
		{"named variadic call", "fn v(ips ...IP) imprimir(ips)\nfn inicio()\n\tv(ips = [.|])\n", true},
		{"unclosed call", "fn inicio()\n\timprimir(f(.|\n", true},
		{"missing later argument", "fn h(ip IP, n num)\n\timprimir(n)\nfn inicio()\n\th(.|\n", true},
		{"partial", "fn inicio()\n\tf(.V|4)\n", true},
		{"payload", "fn inicio()\n\tf(.Otro(.|))\n", true},
		{"variable", "fn inicio()\n\tvar ip IP = .|\n", true},
		{"return", "fn crear() IP\n\t.|\n", true},
		{"assignment", "fn inicio()\n\tvar ip IP = IP.V4\n\tip = .|\n", true},
		{"list", "fn inicio()\n\tvar ips [IP] = [.|]\n", true},
		{"field", "tipo Caja\n\tip IP\nfn inicio()\n\tvar caja Caja = {ip: .|}\n", true},
		{"method", "tipo Caja\n\tip IP\n\tfn poner(ip IP)\n\t\t@ip = ip\nfn inicio()\n\tvar caja Caja = {ip: IP.V4}\n\tcaja.poner(.|)\n", true},
		{"receiver method", "tipo Caja\n\tip IP\n\tfn poner(ip IP)\n\t\t@ip = ip\n\tfn usar()\n\t\t@poner(.|)\n", true},
		{"branch return", "fn crear() IP\n\tsi verdadero\n\t\t.|\n\tsino\n\t\tIP.V4\n", true},
		{"arm result", "fn crear(ip IP) IP\n\tcasos ip\n\t\t.V4 => .|\n\t\t_ => IP.V6\n", true},
		{"label", "fn inicio()\n\tcasos IP.V4\n\t\t.|\n", true},
		{"label block body", "fn inicio()\n\tcasos IP.V4\n\t\t.V| =>\n\t\t\timprimir(1)\n", true},
		{"qualified label", "fn inicio()\n\tcasos IP.V4\n\t\tIP.V| => 1\n", true},
		{"prior arm", "fn inicio()\n\tcasos IP.V4\n\t\t.V4 => 1\n\t\t.|\n", true},
		{"utf16", "fn inicio()\n\tg(\"😀\", .|)\n", true},
		{"no context", "fn inicio()\n\tvar ip = .|\n", false},
		{"print", "fn inicio()\n\timprimir(.|)\n", false},
		{"comment", "fn inicio()\n\t// f(.|)\n", false},
		{"string", "fn inicio()\n\timprimir(\"f(.|)\")\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Put declarations after the cursor to exercise full-document recovery.
			source := tt.source + decl
			mark := strings.Index(source, "|")
			before := source[:mark]
			line := strings.Count(before, "\n")
			column := utf16Length(before[strings.LastIndex(before, "\n")+1:])
			source = strings.Replace(source, "|", "", 1)
			h := servertest.New(t, NewHandler())
			uri := lsp.DocumentURI("file:///contextual.hacha")
			if err := h.DidOpen(uri, "hacha", source); err != nil {
				t.Fatal(err)
			}
			_ = waitForDiagnostics(t, h, uri)
			list, err := h.Completion(uri, line, column)
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if tt.want {
				want = 3
			}
			if len(list.Items) != want {
				t.Fatalf("items = %+v, want %d", list.Items, want)
			}
			for i, item := range list.Items {
				if item.Label != []string{"V4", "V6", "Otro"}[i] || item.TextEdit == nil {
					t.Fatalf("item = %+v", item)
				}
				edit := item.TextEdit.TextEdit
				if edit.NewText != item.Label || edit.Range.Start.Line != line || edit.Range.Start.Character > column || edit.Range.End.Character < column {
					t.Fatalf("edit = %+v", edit)
				}
			}
		})
	}
}

func TestContextualHover(t *testing.T) {
	source := "enum IP\n\tV4\n\tOtro IP\nfn f(ip IP) IP\n\tcasos ip\n\t\tIP.V4 => .V4\n\t\t.Otro => .Otro(.V4)\n"
	h := servertest.New(t, NewHandler())
	uri := lsp.DocumentURI("file:///hover-contextual.hacha")
	if err := h.DidOpen(uri, "hacha", source); err != nil {
		t.Fatal(err)
	}
	if d := waitForDiagnostics(t, h, uri); len(d) != 0 {
		t.Fatal(d)
	}
	for _, tt := range []struct {
		line, col int
		want      string
	}{
		{5, 2, "enum IP"}, {5, 5, "IP.V4"}, {5, 12, "IP.V4"}, {6, 3, "IP.Otro(IP)"}, {6, 12, "IP.Otro(IP)"}, {6, 18, "IP.V4"},
	} {
		hover, err := h.Hover(uri, tt.line, tt.col)
		if err != nil || hover == nil || !strings.Contains(hover.Contents.Value(), tt.want) {
			t.Fatalf("hover %d:%d = %+v, %v; want %s", tt.line, tt.col, hover, err, tt.want)
		}
	}
}
