package codegen

import (
	"hacha/internal/ast"
	"hacha/internal/sema"
	"sort"
	"strings"
)

func typeArguments(args []sema.Type) string {
	if len(args) == 0 {
		return ""
	}
	var names []string
	for _, t := range args {
		names = append(names, goType(t))
	}
	return "[" + strings.Join(names, ", ") + "]"
}

func namedGoType(t sema.Type) string { return exported(t.Name) + typeArguments(t.Args) }

func (g *generator) typeParams(decl ast.Decl) string {
	params := g.model.TypeParams[decl]
	if len(params) == 0 {
		return ""
	}
	var entries []string
	for _, p := range params {
		entries = append(entries, goType(p)+" "+goType(g.model.Constraints[p.Owner+":"+p.Name]))
	}
	return "[" + strings.Join(entries, ", ") + "]"
}

func (g *generator) emitInterface(d *ast.InterfaceDecl) {
	g.line(0, "type %s%s interface {", exported(d.Name), g.typeParams(d))
	info := g.model.Interfaces[d.Name]
	var names []string
	for n := range info.Methods {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		f := info.Methods[n]
		var params []string
		for i, t := range f.Params {
			name := goType(t)
			if f.Decl.Params[i].Variadic {
				name = "..." + goType(*t.Elem)
			}
			params = append(params, name)
		}
		result := ""
		if f.Return.Kind != sema.Void {
			result = " " + goType(f.Return)
		}
		g.line(1, "%s(%s)%s", exported(n), strings.Join(params, ", "), result)
	}
	g.line(0, "}")
	g.line(0, "")
}

func defaultMethod(d *ast.FuncDecl) string { return "_hacha_default_" + exported(d.Name) }

// Keep the public method's signature exact for structural interface satisfaction.
func (g *generator) emitDefaultBridge(d *ast.FuncDecl, f sema.FuncInfo) {
	var params, args []string
	for range g.defaultFlags[d] {
		args = append(args, "false")
	}
	for i, p := range d.Params {
		t := goType(f.Params[i])
		arg := localName(p.Name)
		if p.Variadic {
			t = "..." + goType(*f.Params[i].Elem)
			arg += "..."
		}
		params = append(params, localName(p.Name)+" "+t)
		args = append(args, arg)
	}
	result, ret := "", ""
	if f.Return.Kind != sema.Void {
		result = " " + goType(f.Return)
		ret = "return "
	}
	g.line(0, "func (_self *%s%s) %s(%s)%s {", exported(d.Receiver), typeArguments(g.model.TypeParams[g.model.Types[d.Receiver].Decl]), exported(d.Name), strings.Join(params, ", "), result)
	g.line(1, "%s_self.%s(%s)", ret, defaultMethod(d), strings.Join(args, ", "))
	g.line(0, "}")
	g.line(0, "")
}

func (g *generator) flowTypeMatch(m *ast.MatchExpr, indent int, target string) {
	value := g.flowExpr(m.Value, indent)
	if value == "" {
		return
	}
	name := g.freshName()
	g.line(indent, "switch %s := %s.(type) {", name, value)
	for _, arm := range m.Arms {
		if arm.Pattern == "_" {
			g.line(indent, "default:")
		} else {
			g.line(indent, "case %s:", goType(g.model.PatternTypes[arm]))
		}
		g.line(indent+1, "_ = %s", name)
		if arm.Pattern != "_" && m.Binding != "" {
			g.line(indent+1, "%s := %s", localName(m.Binding), name)
			g.line(indent+1, "_ = %s", localName(m.Binding))
		}
		g.flowBlock(arm.Body, indent+1, target)
	}
	g.line(indent, "}")
}
