package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"cometa/internal/ast"
	"cometa/internal/sema"
	"cometa/internal/stdlib"
	"sort"
	"strings"
)

func gameNamespaceCompletion(prefix string) *lsp.CompletionList {
	return nil // Native namespaces are offered through imported modules only.
}

func gameConstantCompletion(t sema.Type) *lsp.CompletionList {
	constants := stdlib.Constants[t.Name]
	if constants == nil {
		return nil
	}
	list := &lsp.CompletionList{}
	kind := lsp.CompletionItemKindConstant
	var names []string
	for n := range constants {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		list.Items = append(list.Items, lsp.CompletionItem{Label: n, Kind: &kind, Detail: t.Name + "." + n})
	}
	return list
}

func gameTopCompletion(prefix string) *lsp.CompletionList {
	end := len(prefix)
	start := end
	for start > 0 && isIdentifierRune(rune(prefix[start-1])) {
		start--
	}
	partial := prefix[start:]
	list := &lsp.CompletionList{}
	seen := map[string]bool{}
	add := func(name, detail string, kind lsp.CompletionItemKind) {
		if !seen[name] && strings.HasPrefix(name, partial) {
			list.Items = append(list.Items, lsp.CompletionItem{Label: name, Kind: &kind, Detail: detail})
			seen[name] = true
		}
	}

	for _, name := range []string{"entero", "decimal"} {
		add(name, "tipo numérico incorporado "+name, lsp.CompletionItemKindKeyword)
	}

	sort.Slice(list.Items, func(i, j int) bool { return list.Items[i].Label < list.Items[j].Label })
	return list
}

func gameHover(program *ast.Program, model *sema.Model, pos ast.Pos) (hoverInfo, bool) {
	for call, f := range model.Game.Calls {
		if calleePosition(call.Callee) == pos {
			signature := f.Signature
			if strings.HasSuffix(f.GoName, "Entero") && f.Namespace == "mate" {
				signature = strings.ReplaceAll(signature, "decimal", "entero")
			}
			return hoverInfo{detail: "fn " + f.Namespace + "." + f.Name + "(" + signature, documentation: "Biblioteca estándar de Cometa."}, true
		}
	}
	for expr := range model.Game.Constants {
		switch e := expr.(type) {
		case *ast.ContextualVariantExpr:
			if e.NamePos == pos {
				return hoverInfo{detail: model.ExpectedTypes[e].String() + "." + e.Name}, true
			}
		case *ast.MemberExpr:
			if e.NamePos == pos {
				return hoverInfo{detail: "const mate.pi decimal"}, true
			}
		}
	}
	var name string
	sema.WalkSyntax(program, func(n any) {
		if t, ok := n.(*ast.TypeRef); ok && t.Pos == pos && stdlib.IsType(t.Name) {
			name = t.Name
		}
		if lit, ok := n.(*ast.StructLiteralExpr); ok && lit.Pos == pos && stdlib.IsType(lit.TypeName) {
			name = lit.TypeName
		}
	})
	if name != "" {
		return hoverInfo{detail: "tipo incorporado " + name}, true
	}
	return hoverInfo{}, false
}
