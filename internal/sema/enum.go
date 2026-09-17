package sema

import (
	"hacha/internal/ast"
	"hacha/internal/gameapi"
	"strings"
)

type EnumInfo struct {
	Type     Type
	Decl     *ast.EnumDecl
	Variants map[string]VariantInfo
}

type VariantInfo struct {
	Decl    *ast.VariantDecl
	Tag     int
	Payload Type // Void denotes a payload-free variant.
}

type ConstructorInfo struct {
	Enum    *EnumInfo
	Variant VariantInfo
}

func (c *checker) checkContextualVariant(expr *ast.ContextualVariantExpr, call *ast.CallExpr, expected *Type) (Type, error) {
	if expected != nil && gameapi.Constants[expected.Name] != nil {
		value, ok := gameapi.Constants[expected.Name][expr.Name]
		if !ok || call != nil {
			return Type{}, c.fail(expr.Pos, "constante inválida .%s para %s", expr.Name, expected.Name)
		}
		c.model.Game.Constants[expr] = value
		return *expected, nil
	}
	if call != nil {
		if err := c.plainArguments(call); err != nil {
			return Type{}, err
		}
	}
	if expected == nil || expected.Kind != Enum && !expected.Wrapped() {
		return Type{}, c.fail(expr.Pos, "no se puede inferir el enum de .%s; se requiere un tipo enum, opcional o resultado esperado", expr.Name)
	}
	info := c.model.EnumFor(*expected)
	variant, exists := info.Variants[expr.Name]
	if !exists {
		return Type{}, c.fail(expr.NamePos, "la variante %q no existe en %s", expr.Name, expected.String())
	}
	var node ast.Expr = expr
	if call == nil {
		if variant.Payload.Kind != Void {
			return Type{}, c.fail(expr.NamePos, "la variante %s.%s requiere un payload", expected.String(), expr.Name)
		}
	} else {
		node = call
		if variant.Payload.Kind == Void {
			return Type{}, c.fail(expr.Pos, "la variante %s.%s no acepta paréntesis ni payload", expected.String(), expr.Name)
		}
		if len(call.Args) != 1 {
			return Type{}, c.fail(expr.Pos, "la variante %s.%s espera exactamente un payload", expected.String(), expr.Name)
		}
		actual, err := c.checkExprExpected(call.Args[0], &variant.Payload)
		if err != nil {
			return Type{}, err
		}
		if !c.model.Assignable(actual, variant.Payload) {
			return Type{}, c.fail(call.Args[0].Position(), "el payload debe ser %s, no %s", variant.Payload.String(), actual.String())
		}
	}
	c.model.Constructors[node] = ConstructorInfo{Enum: info, Variant: variant}
	return *expected, nil
}

// A local variable shadows a type name, as it does for ordinary member access.
func (c *checker) enumMember(member *ast.MemberExpr) (*EnumInfo, VariantInfo, bool, error) {
	var ident *ast.IdentExpr
	var ref ast.TypeRef
	switch object := member.Object.(type) {
	case *ast.IdentExpr:
		ident = object
		ref = ast.TypeRef{Pos: object.Pos, Name: object.Name}
	case *ast.InstantiateExpr:
		ident = &ast.IdentExpr{Pos: object.Pos, Name: object.Name}
		ref = ast.TypeRef{Pos: object.Pos, Name: object.Name, Args: object.Args}
	default:
		return nil, VariantInfo{}, false, nil
	}
	if _, local := c.vars[ident.Name]; local {
		return nil, VariantInfo{}, false, nil
	}
	info := c.model.Enums[ident.Name]
	if info == nil {
		return nil, VariantInfo{}, false, nil
	}
	t, err := c.resolveType(ref)
	if err != nil {
		return info, VariantInfo{}, true, err
	}
	c.model.ExprTypes[member.Object] = t
	info = c.model.EnumFor(t)
	variant, exists := info.Variants[member.Name]
	if !exists {
		return info, variant, true, c.fail(member.NamePos, "la variante %q no existe en %s", member.Name, ident.Name)
	}
	return info, variant, true, nil
}

func cloneVars(vars map[string]Type) map[string]Type {
	copy := make(map[string]Type, len(vars))
	for name, t := range vars {
		copy[name] = t
	}
	return copy
}

func (c *checker) checkMatch(m *ast.MatchExpr, value bool, expected *Type) (Type, error) {
	defer c.bindingScope()()
	outerBindings := c.bindings
	c.model.LocalNames[m.Binding] = true
	t, err := c.checkExpr(m.Value)
	if err != nil {
		return Type{}, err
	}
	if t.Kind == Interface {
		return c.checkTypeMatch(m, value, expected, t)
	}
	if t.Kind != Enum && !t.Wrapped() {
		return Type{}, c.fail(m.Pos, "casos requiere un enum, opcional o resultado, no %s", t.String())
	}
	info := c.model.EnumFor(t)
	outer, depth := c.vars, c.loopDepth
	defer func() { c.vars, c.loopDepth = outer, depth }()
	// Value matches lower to a function and cannot transfer control out of it.
	if value {
		c.loopDepth = 0
	}
	if m.Binding != "" {
		if _, exists := outer[m.Binding]; exists || m.Binding == "_" {
			return Type{}, c.fail(m.BindingPos, "la variable %q ya fue declarada o está reservada", m.Binding)
		}
	}
	seen := map[string]bool{}
	wildcard := false
	result := Type{Kind: Void}
	for index, arm := range m.Arms {
		c.bindings = cloneBindings(outerBindings)
		c.model.PatternTypes[arm] = t
		if arm.TypePattern != nil {
			return Type{}, c.fail(arm.Pos, "las variantes de casos requieren Enum.Variante o .Variante")
		}
		if arm.QualifierType != nil {
			q, err := c.resolveType(*arm.QualifierType)
			if err != nil {
				return Type{}, err
			}
			if !q.Equal(t) {
				return Type{}, c.fail(arm.Pos, "el patrón debe pertenecer a %s", t.String())
			}
		}
		if arm.Qualifier != "" && arm.Qualifier != t.Name {
			return Type{}, c.fail(arm.QualifierPos, "el patrón debe pertenecer a %s, no %s", t.Name, arm.Qualifier)
		}
		if wildcard {
			return Type{}, c.fail(arm.Pos, "no se permiten ramas después de '_'")
		}
		if seen[arm.Pattern] {
			return Type{}, c.fail(arm.Pos, "la variante %q está duplicada en casos", arm.Pattern)
		}
		seen[arm.Pattern] = true
		c.vars = cloneVars(outer)
		if arm.Pattern == "_" {
			wildcard = true
		} else {
			variant, exists := info.Variants[arm.Pattern]
			if !exists {
				return Type{}, c.fail(arm.Pos, "la variante %q no existe en %s", arm.Pattern, t.Name)
			}
			if m.Binding != "" && variant.Payload.Kind != Void {
				c.vars[m.Binding] = variant.Payload
			}
		}
		if value {
			branchExpected := expected
			if branchExpected == nil && index > 0 {
				branchExpected = &result
			}
			actual, err := c.checkValueBlock(arm.Body, branchExpected)
			if err != nil {
				return Type{}, err
			}
			if index == 0 {
				result = actual
			} else if result.Kind == Never {
				result = actual
			} else if actual.Kind != Never && !result.Equal(actual) {
				return Type{}, c.fail(arm.Pos, "las ramas producen %s y %s", result.String(), actual.String())
			}
		} else {
			for _, stmt := range arm.Body {
				if err := c.checkStmt(stmt); err != nil {
					return Type{}, err
				}
			}
		}
	}
	if !wildcard {
		var missing []string
		for _, variant := range info.Decl.Variants {
			if !seen[variant.Name] {
				missing = append(missing, variant.Name)
			}
		}
		if len(missing) > 0 {
			return Type{}, c.fail(m.Pos, "casos no es exhaustivo; faltan: %s", strings.Join(missing, ", "))
		}
	}
	c.model.ExprTypes[m] = result
	return result, nil
}

// Check final expressions with their expected type before asking for a block
// result, so contextual struct/list literals work through nested branches.
func (c *checker) checkValueBlock(body []ast.Stmt, expected *Type) (Type, error) {
	defer c.bindingScope()()
	outer := c.vars
	c.vars = cloneVars(outer)
	defer func() { c.vars = outer }()
	for index, stmt := range body {
		if index < len(body)-1 {
			if err := c.checkStmt(stmt); err != nil {
				return Type{}, err
			}
			if c.model.Terminates(stmt) {
				return Type{Kind: Never}, nil
			}
			continue
		}
		switch final := stmt.(type) {
		case *ast.ExprStmt:
			t, err := c.checkExprExpected(final.Expr, expected)
			if err != nil {
				return Type{}, err
			}
			if t.Kind != Void {
				return t, nil
			}
		case *ast.MatchStmt:
			return c.checkMatch(final.Match, true, expected)
		case *ast.IfStmt:
			if len(final.Else) == 0 {
				break
			}
			var common Type
			for i, branch := range final.Branches {
				branchOuter := c.vars
				c.vars = cloneVars(branchOuter)
				if err := c.checkCondition(branch.Condition, branch.Binding, branch.BindingPos); err != nil {
					c.vars = branchOuter
					return Type{}, err
				}
				branchExpected := expected
				if branchExpected == nil && i > 0 {
					branchExpected = &common
				}
				t, err := c.checkValueBlock(branch.Body, branchExpected)
				c.vars = branchOuter
				if err != nil {
					return Type{}, err
				}
				if i == 0 || common.Kind == Never {
					common = t
				} else if t.Kind != Never && !common.Equal(t) {
					return Type{}, c.fail(branch.Pos, "las ramas producen %s y %s", common.String(), t.String())
				}
			}
			elseExpected := expected
			if elseExpected == nil {
				elseExpected = &common
			}
			t, err := c.checkValueBlock(final.Else, elseExpected)
			if err != nil {
				return Type{}, err
			}
			if common.Kind != Never && t.Kind != Never && !common.Equal(t) {
				return Type{}, c.fail(final.Pos, "las ramas producen %s y %s", common.String(), t.String())
			}
			if common.Kind == Never {
				common = t
			}
			return common, nil
		default:
			if err := c.checkStmt(stmt); err != nil {
				return Type{}, err
			}
		}
	}
	pos := ast.Pos{Line: 1, Column: 1}
	if len(body) > 0 {
		pos = body[len(body)-1].Position()
	}
	name := "un valor"
	if expected != nil {
		name = expected.String()
	}
	return Type{}, c.fail(pos, "el bloque debe producir %s en todos los caminos", name)
}
