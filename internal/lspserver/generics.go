package lspserver

import (
	"github.com/owenrumney/go-lsp/lsp"
	"hacha/internal/ast"
	"hacha/internal/sema"
	"sort"
	"strings"
)

func typeParamDetail(params []ast.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	var parts []string
	for _, p := range params {
		s := p.Name
		if p.Constraint != nil {
			s += " " + typeRefString(*p.Constraint)
		}
		parts = append(parts, s)
	}
	return "<" + strings.Join(parts, ", ") + ">"
}

func signatureDetail(f sema.FuncInfo) string {
	if f.Decl == nil {
		return ""
	}
	name := f.Decl.Name + typeParamDetail(f.Decl.TypeParams)
	if len(f.TypeArgs) > 0 {
		var args []string
		for _, t := range f.TypeArgs {
			args = append(args, t.String())
		}
		name = f.Decl.Name + "<" + strings.Join(args, ", ") + ">"
	}
	var params []string
	for i, p := range f.Decl.Params {
		t := f.Params[i].String()
		if p.Variadic {
			t = "..." + f.Params[i].Elem.String()
		}
		s := p.Name + " " + t
		if p.Default != nil {
			s += " = …"
		}
		params = append(params, s)
	}
	detail := "fn " + name + "(" + strings.Join(params, ", ") + ")"
	if f.Return.Kind != sema.Void {
		detail += " " + f.Return.String()
	}
	return detail
}

func resolvedMemberItems(model *sema.Model, t sema.Type) *lsp.CompletionList {
	items := []lsp.CompletionItem{}
	if t.Kind == sema.Named {
		if info := model.StructInfo(t); info != nil {
			kind := lsp.CompletionItemKindField
			members := model.Members(t)
			seen := map[string]bool{}
			for _, f := range info.Decl.Fields {
				items = append(items, lsp.CompletionItem{Label: f.Name, Kind: &kind, Detail: info.Fields[f.Name].Type.String()})
				seen[f.Name] = true
			}
			var names []string
			for name, member := range members {
				if !seen[name] && !member.Ambiguous && member.Field.Decl != nil {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			for _, name := range names {
				items = append(items, lsp.CompletionItem{Label: name, Kind: &kind, Detail: members[name].Field.Type.String()})
			}
		}
	}
	methods := model.Methods(t)
	var names []string
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	kind := lsp.CompletionItemKindMethod
	for _, name := range names {
		items = append(items, lsp.CompletionItem{Label: name, Kind: &kind, Detail: signatureDetail(methods[name])})
	}
	return &lsp.CompletionList{Items: items}
}

func genericEnumCompletion(filename string, lines []string, line int, prefix string, dot int) *lsp.CompletionList {
	model := completionModel(filename, lines, line, closeCompletionDelimiters(prefix[:dot+1]+"__hacha_generic_probe__"))
	if model != nil {
		for expr, t := range model.ExprTypes {
			if _, ok := expr.(*ast.InstantiateExpr); ok && expr.Position().Line == line+1 && t.Kind == sema.Enum {
				return enumCompletionItems(model.EnumFor(t))
			}
		}
	}
	return &lsp.CompletionList{}
}
