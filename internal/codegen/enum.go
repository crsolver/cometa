package codegen

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/sema"
)

func (g *generator) freshName() string {
	for {
		g.nextName++
		name := fmt.Sprintf("_cometa%d", g.nextName)
		if !g.model.LocalNames[name] {
			return name
		}
	}
}

func (g *generator) emitEnum(decl *ast.EnumDecl) {
	g.line(0, "type %s%s struct {", exported(decl.Name), g.typeParams(decl))
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
	// imprimir spells variants by name, so the runtime needs them by tag.
	names := make([]string, len(decl.Variants)+1) // tags start at 1
	for _, variant := range decl.Variants {
		if tag := info.Variants[variant.Name].Tag; tag >= 0 && tag < len(names) {
			names[tag] = variant.Name
		}
	}
	if g.printing {
		list := fmt.Sprintf("%q", displayName(decl.Name))
		for _, name := range names[1:] {
			list += fmt.Sprintf(", %q", name)
		}
		g.line(0, "func init() { _hsenumNombres[%q] = []string{%s} }", exported(decl.Name), list)
		g.line(0, "")
	}
}

// displayName drops the module prefix the project binder adds to imported names.
func displayName(name string) string {
	if strings.HasPrefix(name, "CometaModulo") {
		rest := strings.TrimLeft(strings.TrimPrefix(name, "CometaModulo"), "X")
		rest = strings.TrimLeft(rest, "0123456789")
		return strings.TrimPrefix(rest, "_")
	}
	return name
}

// scalarMatch reports whether a casos switches over entero, cadena or bool
// values instead of an enum, optional or result.
func (g *generator) scalarMatch(m *ast.MatchExpr) bool {
	switch g.model.ExprTypes[m.Value].Kind {
	case sema.Integer, sema.String, sema.Boolean:
		return true
	}
	return false
}

// rangeMatch reports whether a scalar casos has a range label (`1..5 =>`).
// Go's switch has no ranges, so such a match becomes a tagless switch whose
// cases compare the value themselves.
func rangeMatch(m *ast.MatchExpr) bool {
	for _, arm := range m.Arms {
		for _, literal := range arm.Literals {
			if _, ok := literal.(*ast.RangeLabel); ok {
				return true
			}
		}
	}
	return false
}

// scalarSwitch writes the opening line of a scalar casos over the variable name.
func (g *generator) scalarSwitch(m *ast.MatchExpr, name string, indent int) (subject string) {
	if rangeMatch(m) {
		g.line(indent, "switch {")
		return name
	}
	g.line(indent, "switch %s {", name)
	return ""
}

// scalarCase writes the `case`/`default` line of a literal arm. A non-empty
// subject is the matched variable of a tagless switch (see rangeMatch).
func (g *generator) scalarCase(arm *ast.MatchArm, indent int, subject string) {
	if arm.Pattern == "_" {
		g.line(indent, "default:")
		return
	}
	literals := make([]string, len(arm.Literals))
	for i, literal := range arm.Literals {
		if r, ok := literal.(*ast.RangeLabel); ok {
			literals[i] = fmt.Sprintf("%s >= %s && %s < %s", subject, g.constExpr(r.Start), subject, g.constExpr(r.End))
		} else if subject != "" {
			literals[i] = fmt.Sprintf("%s == %s", subject, g.constExpr(literal))
		} else {
			literals[i] = g.constExpr(literal)
		}
	}
	g.line(indent, "case %s:", strings.Join(literals, ", "))
}

// Only breaks enclosed in a match need labels. Nested loops handle their own
// breaks; expression matches have an independent function/control-flow scope.
func needsLoopLabel(body []ast.Stmt, inMatch bool) bool {
	for _, stmt := range body {
		switch s := stmt.(type) {
		case *ast.ScopeStmt:
			if needsLoopLabel(s.Body, inMatch) {
				return true
			}
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

// dataFolder names the per-game folder used by std/pincel/datos: the project
// directory plus the entry file, restricted to safe file-name characters.
func dataFolder(entry string) string {
	clean := func(text string) string {
		var b strings.Builder
		for _, r := range text {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
		return b.String()
	}
	stem := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
	directory := filepath.Base(filepath.Dir(entry))
	if directory == "." || directory == string(filepath.Separator) || directory == "" {
		return clean(stem)
	}
	return clean(directory) + "-" + clean(stem)
}
