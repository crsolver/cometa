package codegen

import (
	"fmt"
	"hacha/internal/ast"
	"hacha/internal/sema"
)

func (g *generator) freshName() string {
	for {
		g.nextName++
		name := fmt.Sprintf("_hacha%d", g.nextName)
		if !g.model.LocalNames[name] {
			return name
		}
	}
}

func (g *generator) emitEnum(decl *ast.EnumDecl) {
	g.line(0, "type %s struct {", exported(decl.Name))
	g.line(1, "tag int")
	info := g.model.Enums[decl.Name]
	for _, decl := range decl.Variants {
		variant := info.Variants[decl.Name]
		if variant.Payload.Kind != sema.Void {
			g.line(1, "payload%d %s", variant.Tag, goType(variant.Payload))
		}
	}
	g.line(0, "}")
	g.line(0, "")
}

func (g *generator) emitMatch(m *ast.MatchExpr, indent int, returnValue bool) {
	name := g.freshName()
	info := g.model.Enums[g.model.ExprTypes[m.Value].Name]
	g.line(indent, "{")
	g.line(indent+1, "%s := %s", name, g.expr(m.Value))
	g.line(indent+1, "switch %s.tag {", name)
	wildcard := false
	for _, arm := range m.Arms {
		if arm.Pattern == "_" {
			wildcard = true
			g.line(indent+1, "default:")
		} else {
			variant := info.Variants[arm.Pattern]
			g.line(indent+1, "case %d:", variant.Tag)
			if m.Binding != "" && variant.Payload.Kind != sema.Void {
				binding := localName(m.Binding)
				g.line(indent+2, "%s := %s.payload%d", binding, name, variant.Tag)
				g.line(indent+2, "_ = %s", binding)
			}
		}
		g.emitBlock(arm.Body, indent+2, returnValue)
	}
	// The default also makes exhaustiveness visible to Go's return checker.
	if !wildcard {
		g.line(indent+1, "default:")
		g.line(indent+2, "panic(\"valor enum inválido\")")
	}
	g.line(indent+1, "}")
	g.line(indent, "}")
}

// Only breaks enclosed in a match need labels. Nested loops handle their own
// breaks; expression matches have an independent function/control-flow scope.
func needsLoopLabel(body []ast.Stmt, inMatch bool) bool {
	for _, stmt := range body {
		switch s := stmt.(type) {
		case *ast.BreakStmt:
			if inMatch {
				return true
			}
		case *ast.MatchStmt:
			for _, arm := range s.Match.Arms {
				if needsLoopLabel(arm.Body, true) {
					return true
				}
			}
		case *ast.IfStmt:
			for _, branch := range s.Branches {
				if needsLoopLabel(branch.Body, inMatch) {
					return true
				}
			}
			if needsLoopLabel(s.Else, inMatch) {
				return true
			}
		}
	}
	return false
}
