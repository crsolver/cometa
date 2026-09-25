package sema

import "hacha/internal/ast"

// MemberInfo describes a selector after promotion. Ambiguous members are kept
// so callers can distinguish a collision from a missing member.
type MemberInfo struct {
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
				add(name, MemberInfo{Field: field})
				if field.Decl.Embedded {
					path := append(append([]string(nil), item.path...), name)
					next = append(next, entry{field.Type, path, ancestors})
				}
			}
			for name, method := range info.Methods {
				method.EmbeddedPath = item.path
				add(name, MemberInfo{Method: method})
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
	return member, nil
}
