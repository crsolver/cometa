package sema

import "cometa/internal/ast"

func (c *checker) assertionTarget(source Type, ref ast.TypeRef) (Type, error) {
	target, err := c.resolveType(ref)
	if err != nil {
		return Type{}, err
	}
	if source.Kind != Interface {
		return Type{}, c.fail(ref.Pos, "como requiere una interfaz, no %s", source.String())
	}
	if target.Kind != Interface && target.Kind != TypeParameter && !c.model.Assignable(target, source) {
		return Type{}, c.fail(ref.Pos, "%s no puede implementar %s", target.String(), source.String())
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
					if err := c.checkStmt(stmt); err != nil {
						return err
					}
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
