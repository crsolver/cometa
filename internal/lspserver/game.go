package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"hacha/internal/ast"
	"hacha/internal/gameapi"
	"hacha/internal/sema"
	"sort"
	"strings"
)

func gameNamespaceCompletion(prefix string) *lsp.CompletionList {
	if !completionCodePosition(prefix) {
		return nil
	}
	dot := strings.LastIndex(prefix, ".")
	if dot < 0 {
		return nil
	}
	start := memberReceiverStart(prefix, dot)
	name := prefix[start:dot]
	if !gameapi.IsNamespace(name) {
		return nil
	}
	list := &lsp.CompletionList{}
	kind := lsp.CompletionItemKindFunction
	for _, f := range gameapi.Functions {
		if f.Namespace == name {
			list.Items = append(list.Items, lsp.CompletionItem{Label: f.Name, Kind: &kind, Detail: "fn " + f.Namespace + "." + f.Name + "(" + f.Signature})
		}
	}
	if name == "mate" {
		k := lsp.CompletionItemKindConstant
		list.Items = append(list.Items, lsp.CompletionItem{Label: "pi", Kind: &k, Detail: "const pi num"})
	}
	return list
}

func gameConstantCompletion(t sema.Type) *lsp.CompletionList {
	constants := gameapi.Constants[t.Name]
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
	for name := range gameapi.Fields {
		add(name, "tipo incorporado "+name, lsp.CompletionItemKindStruct)
	}
	for _, f := range gameapi.Functions {
		add(f.Namespace, "API de juego "+f.Namespace, lsp.CompletionItemKindModule)
	}
	for _, sig := range []string{"actualizar(dt num)", "pintar()", "iniciar()"} {
		name := strings.Split(sig, "(")[0]
		add(name, "fn "+sig, lsp.CompletionItemKindFunction)
	}
	sort.Slice(list.Items, func(i, j int) bool { return list.Items[i].Label < list.Items[j].Label })
	return list
}

func gameHover(program *ast.Program, model *sema.Model, pos ast.Pos) (hoverInfo, bool) {
	for call, f := range model.Game.Calls {
		m := call.Callee.(*ast.MemberExpr)
		id := m.Object.(*ast.IdentExpr)
		if m.NamePos == pos {
			return hoverInfo{detail: "fn " + f.Namespace + "." + f.Name + "(" + f.Signature, documentation: "API incorporada de Hacha / Ebitengine."}, true
		}
		if id.Pos == pos {
			return hoverInfo{detail: "API de juego " + f.Namespace}, true
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
				return hoverInfo{detail: "const mate.pi num"}, true
			}
		}
	}
	var name string
	sema.WalkSyntax(program, func(n any) {
		if t, ok := n.(*ast.TypeRef); ok && t.Pos == pos && gameapi.IsType(t.Name) {
			name = t.Name
		}
		if lit, ok := n.(*ast.StructLiteralExpr); ok && lit.Pos == pos && gameapi.IsType(lit.TypeName) {
			name = lit.TypeName
		}
	})
	if name != "" {
		return hoverInfo{detail: "tipo incorporado " + name}, true
	}
	return hoverInfo{}, false
}
