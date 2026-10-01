package sema

import "github.com/crsolver/cometa/internal/ast"

// MemberInfo describes a selector after promotion. Ambiguous members are kept
// so callers can distinguish a collision from a missing member.
type MemberInfo struct {
	Path      []*ast.Field
	Field     FieldInfo
	Method    FuncInfo
	Ambiguous bool
}

// Members resolves fields and methods in one namespace, breadth first. Separate
// paths remain separate even when they reach the same declaration (diamonds).
// The per-path declaration guard also makes partial tooling models safe.
func (m *Model) Members(t Type) map[string]MemberInfo {
	type entry struct {
		typeOf    Type
		fields    []*ast.Field
		path      []string
		ancestors map[string]bool
	}
	result := map[string]MemberInfo{}
	level := []entry{{typeOf: t}}
	for len(level) > 0 {
		current := map[string]MemberInfo{}
		var next []entry
		add := func(name string, member MemberInfo) {
			if _, shadowed := result[name]; shadowed {
				return
			}
			if _, exists := current[name]; exists {
				member = MemberInfo{Ambiguous: true}
			}
			current[name] = member
		}
		for _, item := range level {
			if item.typeOf.Kind != Named || item.ancestors[item.typeOf.Name] {
				continue
			}
			info := m.StructInfo(item.typeOf)
			if info == nil {
				continue
			}
			ancestors := map[string]bool{item.typeOf.Name: true}
			for name := range item.ancestors {
				ancestors[name] = true
			}
			for name, field := range info.Fields {
				add(name, MemberInfo{Field: field, Path: item.fields})
				if field.Decl.Embedded {
					path := append(append([]string(nil), item.path...), name)
					fields := append(append([]*ast.Field(nil), item.fields...), field.Decl)
					next = append(next, entry{typeOf: field.Type, fields: fields, path: path, ancestors: ancestors})
				}
			}
			for name, method := range info.Methods {
				method.EmbeddedPath = item.path
				add(name, MemberInfo{Method: method, Path: item.fields})
			}
		}
		for name, member := range current {
			result[name] = member
		}
		level = next
	}
	return result
}

func (c *checker) receiverType() Type {
	return Type{Kind: Named, Name: c.receiver.Decl.Name, Args: c.model.TypeParams[c.receiver.Decl]}
}

func (c *checker) member(t Type, name string, pos ast.Pos) (MemberInfo, error) {
	if c.invalidType(t) {
		return MemberInfo{}, errInvalid
	}
	member := c.model.Members(t)[name]
	if member.Ambiguous {
		return member, c.fail(pos, "el miembro %q es ambiguo en %s; use la ruta explícita del tipo embebido", name, t.String())
	}
	if !member.Accessible(pos) {
		return member, c.fail(pos, "el miembro %q es privado", name)
	}
	return member, nil
}

// Accessible checks the complete promotion path without altering name resolution.
func (m MemberInfo) Accessible(pos ast.Pos) bool {
	for _, f := range m.Path {
		if !ast.Accessible(f.Public, f.Pos, pos) {
			return false
		}
	}
	if f := m.Field.Decl; f != nil {
		return ast.Accessible(f.Public, f.Pos, pos)
	}
	if f := m.Method.Decl; f != nil {
		return ast.Accessible(f.Public, f.Pos, pos)
	}
	return true
}

func (m MemberInfo) Public() bool {
	for _, f := range m.Path {
		if !f.Public {
			return false
		}
	}
	return m.Method.Decl != nil && m.Method.Decl.Public
}
