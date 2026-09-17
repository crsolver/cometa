package sema

import (
	"hacha/internal/ast"
	"hacha/internal/gameapi"
)

func (t Type) Wrapped() bool { return t.Kind == Optional || t.Kind == Result }

// EnumFor exposes the same exhaustive-pattern interface for enums and wrappers.
func (m *Model) EnumFor(t Type) *EnumInfo {
	if !t.Wrapped() {
		base := m.Enums[t.Name]
		if base == nil || len(t.Args) == 0 {
			return base
		}
		info := &EnumInfo{Decl: base.Decl, Type: t, Variants: map[string]VariantInfo{}}
		bindings := bindTypes(m.TypeParams[base.Decl], t.Args)
		for n, v := range base.Variants {
			v.Payload = substitute(v.Payload, bindings)
			info.Variants[n] = v
		}
		return info
	}
	info := &EnumInfo{Decl: &ast.EnumDecl{Name: t.String()}, Variants: map[string]VariantInfo{}}
	add := func(name string, tag int, payload Type) {
		decl := &ast.VariantDecl{Name: name}
		info.Decl.Variants = append(info.Decl.Variants, decl)
		info.Variants[name] = VariantInfo{Decl: decl, Tag: tag, Payload: payload}
	}
	if t.Kind == Optional {
		add("Ninguno", 0, Type{Kind: Void})
		add("Alguno", 1, *t.Elem)
	} else {
		add("Ok", 1, *t.Elem)
		add("Error", 2, *t.Err)
	}
	return info
}

func isContextualConstructor(e ast.Expr) bool {
	if _, ok := e.(*ast.ContextualVariantExpr); ok {
		return true
	}
	if call, ok := e.(*ast.CallExpr); ok {
		_, ok = call.Callee.(*ast.ContextualVariantExpr)
		return ok
	}
	return false
}

func (c *checker) checkReturn(e *ast.ReturnExpr) (Type, error) {
	if c.inDefault {
		return Type{}, c.fail(e.Pos, "retornar no se permite en valores predeterminados")
	}
	if e.Value == nil {
		if c.returnType.Kind != Void {
			return Type{}, c.fail(e.Pos, "retornar requiere %s", c.returnType.String())
		}
	} else {
		if c.returnType.Kind == Void {
			return Type{}, c.fail(e.Pos, "esta función no devuelve un valor")
		}
		t, err := c.checkExprExpected(e.Value, &c.returnType)
		if err != nil {
			return Type{}, err
		}
		if !c.model.Assignable(t, c.returnType) {
			return Type{}, c.fail(e.Pos, "retornar requiere %s, no %s", c.returnType.String(), t.String())
		}
	}
	return Type{Kind: Never}, nil
}

func (c *checker) checkTry(e *ast.TryExpr) (Type, error) {
	if c.inDefault {
		return Type{}, c.fail(e.Pos, "intentar no se permite en valores predeterminados")
	}
	t, err := c.checkExpr(e.Value)
	if err != nil {
		return Type{}, err
	}
	if !t.Wrapped() {
		return Type{}, c.fail(e.Pos, "intentar requiere un opcional o resultado, no %s", t.String())
	}
	if t.Kind != c.returnType.Kind || t.Kind == Result && !t.Err.Equal(*c.returnType.Err) {
		return Type{}, c.fail(e.Pos, "no se puede propagar %s desde una función que devuelve %s", t.String(), c.returnType.String())
	}
	return *t.Elem, nil
}

func (c *checker) checkRecovery(e *ast.RecoverExpr) (Type, error) {
	defer c.bindingScope()()
	t, err := c.checkExpr(e.Value)
	if err != nil {
		return Type{}, err
	}
	kind := Optional
	if e.Error {
		kind = Result
	}
	if t.Kind != kind {
		return Type{}, c.fail(e.Pos, "el operador de recuperación no acepta %s", t.String())
	}
	outer := c.vars
	c.vars = cloneVars(outer)
	defer func() { c.vars = outer }()
	if e.Binding != "" {
		if err := c.bindPayload(e.Binding, e.BindingPos, *t.Err); err != nil {
			return Type{}, err
		}
	}
	if t.Elem.Kind == Void {
		for _, stmt := range e.Body {
			if err := c.checkStmt(stmt); err != nil {
				return Type{}, err
			}
		}
	} else {
		actual, err := c.checkValueBlock(e.Body, t.Elem)
		if err != nil {
			return Type{}, err
		}
		if !c.model.Assignable(actual, *t.Elem) {
			return Type{}, c.fail(e.Pos, "la recuperación debe producir %s, no %s", t.Elem.String(), actual.String())
		}
	}
	return *t.Elem, nil
}

func cloneBindings(bindings map[string]*ast.VarDeclStmt) map[string]*ast.VarDeclStmt {
	copy := map[string]*ast.VarDeclStmt{}
	for name, binding := range bindings {
		copy[name] = binding
	}
	return copy
}

func (c *checker) bindingScope() func() {
	outer := c.bindings
	c.bindings = cloneBindings(outer)
	return func() { c.bindings = outer }
}

// Terminates identifies checked unconditional exits, including exhaustive branches.
func (m *Model) Terminates(stmt ast.Stmt) bool {
	block := func(body []ast.Stmt) bool {
		for _, s := range body {
			if m.Terminates(s) {
				return true
			}
		}
		return false
	}
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return m.ExprTypes[s.Expr].Kind == Never
	case *ast.VarDeclStmt:
		return m.ExprTypes[s.Value].Kind == Never
	case *ast.AssignStmt:
		return m.ExprTypes[s.Value].Kind == Never
	case *ast.IfStmt:
		if len(s.Else) == 0 || !block(s.Else) {
			return false
		}
		for _, b := range s.Branches {
			if !block(b.Body) {
				return false
			}
		}
		return true
	case *ast.MatchStmt:
		if len(s.Match.Arms) == 0 {
			return false
		}
		for _, arm := range s.Match.Arms {
			if !block(arm.Body) {
				return false
			}
		}
		return true
	}
	return false
}

// A terminating operand prevents evaluation of the rest of an eager expression.
func (m *Model) eagerExit(e ast.Expr) bool {
	exits := func(v ast.Expr) bool { return m.ExprTypes[v].Kind == Never }
	switch v := e.(type) {
	case *ast.AssertExpr:
		return exits(v.Value)
	case *ast.CallExpr:
		for _, arg := range v.Args {
			if exits(arg) {
				return true
			}
		}
	case *ast.StructLiteralExpr:
		for _, field := range v.Fields {
			if exits(field.Value) {
				return true
			}
		}
		for _, value := range v.Values {
			if exits(value) {
				return true
			}
		}
	case *ast.ListLiteralExpr:
		for _, item := range v.Elements {
			if exits(item) {
				return true
			}
		}
	case *ast.InterpolatedStringExpr:
		for _, part := range v.Parts {
			if part.Expr != nil && exits(part.Expr) {
				return true
			}
		}
	case *ast.BinaryExpr:
		return exits(v.Left) || v.Operator != "&&" && v.Operator != "||" && exits(v.Right)
	}
	return false
}

func (c *checker) bindPayload(name string, pos ast.Pos, t Type) error {
	if _, exists := c.vars[name]; exists || name == "_" {
		return c.fail(pos, "la variable %q ya fue declarada o está reservada", name)
	}
	c.vars[name] = t
	c.model.LocalNames[name] = true
	return nil
}

func (c *checker) checkCondition(e ast.Expr, binding string, pos ast.Pos) error {
	t, err := c.checkExpr(e)
	if err != nil {
		return err
	}
	if binding != "" {
		if t.Kind != Optional {
			return c.fail(e.Position(), "si con ligadura requiere un opcional, no %s", t.String())
		}
		return c.bindPayload(binding, pos, *t.Elem)
	}
	if t.Kind != Boolean {
		return c.fail(e.Position(), "la condición debe ser bool, no %s", t.String())
	}
	return nil
}

func (c *checker) checkRequiredCycles(program *ast.Program) error {
	var visit func(Type, map[string]bool, ast.Pos) error
	visit = func(t Type, path map[string]bool, pos ast.Pos) error {
		if t.Kind != Named {
			return nil
		}
		key := t.String()
		if path[key] {
			return c.fail(pos, "ciclo de campos obligatorios en %s; use un opcional o una lista", key)
		}
		path[key] = true
		defer delete(path, key)
		info := c.model.StructInfo(t)
		for _, field := range info.Decl.Fields {
			if err := visit(info.Fields[field.Name].Type, path, field.Pos); err != nil {
				return err
			}
		}
		return nil
	}
	for _, decl := range program.Decls {
		if d, ok := decl.(*ast.TypeDecl); ok {
			if err := visit(Type{Kind: Named, Name: d.Name, Args: c.model.TypeParams[d]}, map[string]bool{}, d.Pos); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *checker) missingDefault(t Type, path string) string {
	if gameapi.IsType(t.Name) && !gameapi.IsValue(t.Name) {
		return path
	}
	if t.Kind == Result || t.Kind == Enum || t.Kind == Interface || t.Kind == TypeParameter {
		return path
	}
	if t.Kind == Named {
		info := c.model.StructInfo(t)
		for _, field := range info.Decl.Fields {
			if c.model.Types[t.Name].Fields[field.Name].Type.Kind == TypeParameter {
				return path + "." + field.Name
			}
			if missing := c.missingDefault(info.Fields[field.Name].Type, path+"."+field.Name); missing != "" {
				return missing
			}
		}
	}
	return ""
}
