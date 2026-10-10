package lspserver

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/sema"
	"github.com/crsolver/cometa/internal/stdlib"
	"github.com/owenrumney/go-lsp/lsp"
	"sort"
	"strings"
)

func constantCompletion(t sema.Type) *lsp.CompletionList {
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

func builtinTypeCompletion(prefix string) *lsp.CompletionList {
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

func stdlibHover(program *ast.Program, model *sema.Model, pos ast.Pos) (hoverInfo, bool) {
	for call, f := range model.Stdlib.Calls {
		if calleePosition(call.Callee) == pos {
			// f is the variant the call resolved to: mate.minimo(1, 2) shows integers.
			return hoverInfo{detail: "fn " + f.Namespace + "." + f.Name + f.Signature, documentation: "Biblioteca estándar de Cometa."}, true
		}
	}
	for expr := range model.Stdlib.Constants {
		switch e := expr.(type) {
		case *ast.ContextualVariantExpr:
			if e.NamePos == pos {
				return hoverInfo{detail: model.ExpectedTypes[e].String() + "." + e.Name}, true
			}
		case *ast.MemberExpr:
			if e.NamePos == pos {
				return hoverInfo{detail: model.ExprTypes[e].String() + "." + e.Name}, true
			}
		}
	}
	var name string
	ast.Inspect(program, func(n any) bool {
		if t, ok := n.(*ast.TypeRef); ok && t.Pos == pos && stdlib.IsType(t.Name) {
			name = t.Name
		}
		if lit, ok := n.(*ast.StructLiteralExpr); ok && lit.Pos == pos && stdlib.IsType(lit.TypeName) {
			name = lit.TypeName
		}
		return true
	})
	if name != "" {
		return hoverInfo{detail: "tipo incorporado " + name}, true
	}
	return hoverInfo{}, false
}
