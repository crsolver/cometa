package compiler

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"path/filepath"
	"reflect"
	"strings"
)

// Binding keeps source ASTs intact and resolves the separate compiler AST once.
// The existing semantic checker then sees globally unique nominal identities.
type binder struct {
	project *Project
	module  *Module
	types   map[string]bool
	errors  diagnostic.List
	scope   map[string]bool // locals visible to the name being resolved, for suggestions
}

func copyScope(scope map[string]bool, names ...string) map[string]bool {
	out := map[string]bool{}
	for k, v := range scope {
		out[k] = v
	}
	for _, name := range names {
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func (b *binder) name(name string, pos ast.Pos) string {
	if name == "" {
		return name
	}
	m := b.module
	parts := strings.Split(name, ".")
	if len(parts) == 2 {
		m = m.Imports[parts[0]]
		if m == nil {
			b.fail(pos, "módulo desconocido %q", parts[0])
			return name
		}
		name = parts[1]
		aliasPos := pos
		pos.Column += len([]rune(parts[0])) + 1
		if d := m.Declarations[name]; d != nil && ast.IsPublic(d.Node) {
			b.project.References[aliasPos] = d
		}
	}
	if d := m.Declarations[name]; d != nil {
		if m != b.module && d.Node != nil && !ast.IsPublic(d.Node) {
			b.fail(pos, "la declaración %q es privada en el módulo %s", name, m.Path)
			invalid := "\x00" + m.Path + ":" + name
			b.project.Program.InvalidNames[invalid] = pos
			return invalid
		}
		b.project.References[pos] = d
		return d.Symbol
	}
	if !m.Unavailable {
		var candidates []string
		for known := range m.Declarations {
			candidates = append(candidates, known)
		}
		if m == b.module {
			for local := range b.scope {
				candidates = append(candidates, local)
			}
			candidates = append(candidates, "imprimir")
			hint := diagnostic.Hint(name, candidates)
			if foreign := diagnostic.Foreign(name); foreign != "" {
				hint = "; " + foreign
			}
			b.fail(pos, "el nombre %q no existe%s", name, hint)
		} else {
			b.fail(pos, "el módulo %s no tiene %q%s", strings.TrimSuffix(filepath.Base(m.Path), ".cometa"), name, diagnostic.Hint(name, candidates))
		}
	}
	invalid := "\x00" + m.Path + ":" + name
	b.project.Program.InvalidNames[invalid] = pos
	return invalid
}

func (b *binder) fail(pos ast.Pos, format string, args ...any) {
	b.errors.Add(projectError(pos, format, args...))
}

func (b *binder) typeRef(t *ast.TypeRef) {
	if t == nil {
		return
	}
	b.typeRef(t.Element)
	b.typeRef(t.Key)
	b.typeRef(t.Payload)
	b.typeRef(t.ErrorType)
	for i := range t.Args {
		b.typeRef(&t.Args[i])
	}
	if t.Name == "" || t.Name == "entero" || t.Name == "decimal" || t.Name == "num" || t.Name == "cadena" || t.Name == "bool" || t.Name == "$unidad" || b.types[t.Name] {
		return
	}
	t.Name = b.name(t.Name, t.Pos)
}

func (b *binder) parameters(params []ast.TypeParam) {
	for _, p := range params {
		if b.module.Declarations[p.Name] != nil {
			b.fail(p.Pos, "parámetro de tipo en conflicto %q", p.Name)
		}
		b.types[p.Name] = true
	}
	for i := range params {
		b.typeRef(params[i].Constraint)
	}
}

func (b *binder) bind() error {
	for _, node := range b.module.Bound.Decls {
		b.types = map[string]bool{}
		name, _ := declarationName(node)
		symbol := b.module.Declarations[name].Symbol
		switch d := node.(type) {
		case *ast.TypeDecl:
			b.parameters(d.TypeParams)
			d.Name = symbol
			for _, f := range d.Fields {
				b.typeRef(&f.Type)
				f.Default = b.expr(f.Default, map[string]bool{})
			}
			for _, f := range d.Methods {
				f.Receiver = symbol
				b.function(f)
			}
		case *ast.EnumDecl:
			b.parameters(d.TypeParams)
			d.Name = symbol
			for _, v := range d.Variants {
				b.typeRef(v.Payload)
			}
		case *ast.InterfaceDecl:
			b.parameters(d.TypeParams)
			d.Name = symbol
			for i := range d.Embeds {
				b.typeRef(&d.Embeds[i])
			}
			for _, f := range d.Methods {
				b.function(f)
			}
		case *ast.FuncDecl:
			b.parameters(d.TypeParams)
			d.Name = symbol
			b.function(d)
		case *ast.GlobalDecl:
			d.Name = symbol
			b.typeRef(d.Type)
			d.Value = b.expr(d.Value, map[string]bool{})
		}
	}
	return b.errors.Err()
}

func (b *binder) function(f *ast.FuncDecl) {
	scope := map[string]bool{}
	for i := range f.Params {
		p := &f.Params[i]
		b.typeRef(&p.Type)
		p.Default = b.expr(p.Default, scope)
		scope[p.Name] = true
	}
	b.typeRef(f.ReturnType)
	b.statements(f.Body, scope)
}

func (b *binder) statements(body []ast.Stmt, parent map[string]bool) {
	scope := copyScope(parent)
	for _, stmt := range body {
		switch s := stmt.(type) {
		case *ast.BadStmt:
			if s.Name != "" {
				scope[s.Name] = true
			}
		case *ast.MatchStmt:
			b.expr(s.Match, scope)
		case *ast.VarDeclStmt:
			b.typeRef(s.Type)
			s.Value = b.expr(s.Value, scope)
			scope[s.Name] = true
		case *ast.IfStmt:
			for i := range s.Branches {
				branch := &s.Branches[i]
				branch.Condition = b.expr(branch.Condition, scope)
				b.statements(branch.Body, copyScope(scope, branch.Binding))
			}
			b.statements(s.Else, scope)
		case *ast.ScopeStmt:
			s.Value = b.expr(s.Value, scope)
			b.statements(s.Body, copyScope(scope))
		case *ast.RepeatStmt:
			s.Iterable = b.expr(s.Iterable, scope)
			s.RangeEnd = b.expr(s.RangeEnd, scope)
			b.statements(s.Body, copyScope(scope, s.Element, s.Index))
		default:
			b.children(reflect.ValueOf(stmt), scope)
		}
	}
}

func (b *binder) expr(expr ast.Expr, scope map[string]bool) ast.Expr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *ast.IdentExpr:
		if !scope[e.Name] && !b.types[e.Name] && e.Name != "imprimir" && e.Name != "entero" && e.Name != "decimal" {
			b.scope = scope
			e.Name = b.name(e.Name, e.Pos)
			b.scope = nil
		}
		return e
	case *ast.MemberExpr:
		if id, ok := e.Object.(*ast.IdentExpr); ok && !scope[id.Name] && !b.types[id.Name] && b.module.Imports[id.Name] != nil {
			name := b.name(id.Name+"."+e.Name, id.Pos)
			return &ast.IdentExpr{Pos: e.NamePos, Name: name}
		}
	case *ast.InstantiateExpr:
		original := e.Name
		e.Name = b.name(e.Name, e.Pos)
		if at := strings.Index(original, "."); at >= 0 {
			e.Pos.Column += len([]rune(original[:at])) + 1
		}
		for i := range e.Args {
			b.typeRef(&e.Args[i])
		}
		return e
	case *ast.StructLiteralExpr:
		if e.Type != nil {
			b.typeRef(e.Type)
			e.TypeName = e.Type.Name
		} else {
			e.TypeName = b.name(e.TypeName, e.Pos)
		}
		for i := range e.Fields {
			e.Fields[i].Value = b.expr(e.Fields[i].Value, scope)
		}
		for i := range e.Values {
			e.Values[i] = b.expr(e.Values[i], scope)
		}
		return e
	case *ast.BlockExpr:
		b.statements(e.Body, scope)
		return e
	case *ast.RecoverExpr:
		e.Value = b.expr(e.Value, scope)
		b.statements(e.Body, copyScope(scope, e.Binding))
		return e
	case *ast.IfExpr:
		for i := range e.Branches {
			branch := &e.Branches[i]
			branch.Condition = b.expr(branch.Condition, scope)
			branch.Value = b.expr(branch.Value, copyScope(scope, branch.Binding))
		}
		e.Else = b.expr(e.Else, scope)
		return e
	case *ast.MatchExpr:
		e.Value = b.expr(e.Value, scope)
		for _, arm := range e.Arms {
			for i, literal := range arm.Literals {
				arm.Literals[i] = b.expr(literal, scope)
			}
			b.typeRef(arm.TypePattern)
			b.typeRef(arm.QualifierType)
			if arm.QualifierType != nil {
				arm.Qualifier = arm.QualifierType.Name
			}
			b.statements(arm.Body, copyScope(scope, e.Binding))
		}
		return e
	}
	b.children(reflect.ValueOf(expr), scope)
	return expr
}

// Ordinary AST nodes have no scope boundaries. Walk their typed children while
// retaining the explicit handling above for declarations and lexical scopes.
func (b *binder) children(v reflect.Value, scope map[string]bool) {
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		if e, ok := v.Interface().(ast.Expr); ok {
			v.Set(reflect.ValueOf(b.expr(e, scope)))
			return
		}
		b.children(v.Elem(), scope)
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		if t, ok := v.Interface().(*ast.TypeRef); ok {
			b.typeRef(t)
			return
		}
		b.children(v.Elem(), scope)
		return
	}
	if v.Kind() == reflect.Struct {
		if v.Type() == reflect.TypeOf(ast.TypeRef{}) {
			b.typeRef(v.Addr().Interface().(*ast.TypeRef))
			return
		}
		for i := 0; i < v.NumField(); i++ {
			b.children(v.Field(i), scope)
		}
	}
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			b.children(v.Index(i), scope)
		}
	}
}
