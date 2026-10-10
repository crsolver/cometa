package sema

import (
	"fmt"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
)

func (c *checker) assertionTarget(source Type, ref ast.TypeRef) (Type, error) {
	target, err := c.resolveType(ref)
	if err != nil {
		return Type{}, err
	}
	if source.Kind != Interface {
		return Type{}, c.fail(ref.Pos, "como requiere una interfaz, no %s", source.String())
	}
	if target.Kind != Interface && target.Kind != TypeParameter && !c.model.Assignable(target, source) {
		return Type{}, c.fail(ref.Pos, "%s no puede implementar %s%s", target.String(), source.String(), c.unmet(target, source))
	}
	return target, nil
}

func (c *checker) checkAssertion(e *ast.AssertExpr) (Type, error) {
	source, err := c.checkExpr(e.Value)
	if err != nil {
		return Type{}, err
	}
	target, err := c.assertionTarget(source, e.Target)
	if err != nil {
		return Type{}, err
	}
	c.model.TypeRefs[&e.Target] = target
	return Type{Kind: Optional, Elem: &target}, nil
}

func (c *checker) checkTypeMatch(m *ast.MatchExpr, value bool, expected *Type, source Type) (Type, error) {
	outer, depth := c.vars, c.loopDepth
	outerBindings := c.bindings
	defer func() { c.vars = outer; c.loopDepth = depth; c.bindings = outerBindings }()
	if value {
		c.loopDepth = 0
	}
	if m.Binding != "" {
		if _, ok := outer[m.Binding]; ok || m.Binding == "_" {
			return Type{}, c.fail(m.BindingPos, "la variable %q ya fue declarada o está reservada", m.Binding)
		}
	}
	var seen []Type
	wildcard := false
	result := Type{Kind: Void}
	broken := false
	for i, arm := range m.Arms {
		if arm.Invalid {
			broken = true
			c.damaged++
			continue
		}
		err := func() error {
			if wildcard {
				return c.fail(arm.Pos, "no se permiten ramas después de '_'")
			}
			c.vars = cloneVars(outer)
			c.bindings = cloneBindings(outerBindings)
			if arm.Pattern == "_" {
				wildcard = true
			} else {
				if arm.TypePattern == nil {
					return c.fail(arm.Pos, "casos sobre una interfaz requiere un tipo o '_'")
				}
				t, err := c.assertionTarget(source, *arm.TypePattern)
				if err != nil {
					return err
				}
				for _, prior := range seen {
					if prior.Equal(t) {
						return c.fail(arm.Pos, "tipo duplicado en casos: %s", t.String())
					}
				}
				seen = append(seen, t)
				c.model.PatternTypes[arm] = t
				if m.Binding != "" {
					c.vars[m.Binding] = t
				}
			}
			if value {
				want := expected
				if want == nil && i > 0 && result.Kind != Void {
					want = &result
				}
				t, err := c.checkValueBlock(arm.Body, want)
				if err != nil {
					return err
				}
				if result.Kind == Void || result.Kind == Never {
					result = t
				} else if t.Kind != Never && !result.Equal(t) {
					if result.Kind == Integer && t.Kind == Decimal {
						for _, prior := range m.Arms[:i] {
							c.promoteBlock(prior.Body)
						}
						result = t
						return nil
					}
					return c.fail(arm.Pos, "las ramas producen %s y %s", result.String(), t.String())
				}
			} else {
				for _, stmt := range arm.Body {
					c.checkStmt(stmt)
				}
			}
			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
			c.damaged++
		}
	}
	if broken {
		return Type{}, errInvalid
	}
	if !wildcard {
		return Type{}, c.fail(m.Pos, "casos sobre una interfaz requiere una rama '_' final")
	}
	c.model.ExprTypes[m] = result
	return result, nil
}

// Unmet explains why from does not implement the interface to, naming the first
// requirement (in alphabetical order) that fails. It returns "" when to is not
// an interface or from implements it. It follows the rules of Assignable.
func (m *Model) Unmet(from, to Type) string {
	if to.Wrapped() && !from.Wrapped() {
		to = *to.Elem
	}
	if to.Kind != Interface || from.Kind == Void || from.Kind == Invalid || m.Assignable(from, to) {
		return ""
	}
	if from.Kind == Map {
		return "las operaciones de un mapa no cuentan como métodos de una interfaz"
	}
	wanted := m.Methods(to)
	have := m.Methods(from)
	var members map[string]MemberInfo
	if from.Kind == Named {
		members = m.Members(from)
	}
	for _, name := range sortedKeys(wanted) {
		want := wanted[name]
		full := from.String() + "." + name
		member, declared := members[name]
		switch {
		case member.Ambiguous:
			return full + " es ambiguo: llega por más de un tipo embebido"
		case declared && member.Method.Decl == nil:
			return full + " es un campo, no un método"
		}
		actual, ok := have[name]
		if !ok {
			return "a " + from.String() + " le falta el método " + methodSignature(name, want)
		}
		if declared && !member.Public() {
			for _, field := range member.Path {
				if !field.Public {
					return full + " viene del tipo embebido " + field.Name + ", que no es pub"
				}
			}
			return full + " existe pero no es pub"
		}
		if sameSignature(actual, want) {
			continue
		}
		if sameSignature(FuncInfo{Decl: actual.Decl, Params: actual.Params, Return: want.Return}, want) {
			return full + " " + returns(actual.Return, "devuelve %s", "no devuelve nada") + " y " + to.String() + " pide " + returns(want.Return, "%s", "que no devuelva nada")
		}
		return full + " es " + methodSignature(name, actual) + " y " + to.String() + " pide " + methodSignature(name, want)
	}
	return ""
}

func returns(t Type, format, none string) string {
	if t.Kind == Void {
		return none
	}
	return fmt.Sprintf(format, t.String())
}

// methodSignature writes a method the way its declaration does, without 'fn'.
func methodSignature(name string, f FuncInfo) string {
	var b strings.Builder
	b.WriteString(name + "(")
	for i, param := range f.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		if f.Decl != nil && i < len(f.Decl.Params) {
			b.WriteString(f.Decl.Params[i].Name + " ")
			if f.Decl.Params[i].Variadic && param.Kind == Slice {
				b.WriteString("..." + param.Elem.String())
				continue
			}
		}
		b.WriteString(param.String())
	}
	b.WriteString(")")
	if f.Return.Kind != Void {
		b.WriteString(" " + f.Return.String())
	}
	return b.String()
}

// unmet is Unmet as the ending of a type mismatch message.
func (c *checker) unmet(from, to Type) string {
	if reason := c.model.Unmet(from, to); reason != "" {
		return ": " + reason
	}
	return ""
}
