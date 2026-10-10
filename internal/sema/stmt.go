package sema

import (
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/stdlib"
)

// checkStmt checks one statement. A failure is reported and counted in
// c.damaged, and the statements after it are still checked.
func (c *checker) checkStmt(stmt ast.Stmt) {
	if bad, ok := stmt.(*ast.BadStmt); ok {
		c.damaged++
		if bad.Name != "" {
			c.vars[bad.Name] = Type{Kind: Invalid}
		}
		return
	}
	err := c.checkStatement(stmt)
	if err != nil {
		c.report(err)
		c.damaged++
		if d, ok := stmt.(*ast.VarDeclStmt); ok {
			if _, exists := c.vars[d.Name]; !exists {
				c.vars[d.Name] = Type{Kind: Invalid}
			}
		}
	}
}

func (c *checker) checkStatement(stmt ast.Stmt) error {
	switch statement := stmt.(type) {
	case *ast.MatchStmt:
		_, err := c.checkMatch(statement.Match, false, nil)
		return err
	case *ast.ExprStmt:
		t, err := c.checkExpr(statement.Expr)
		if err == nil && t.Wrapped() {
			return c.fail(statement.Pos, "no se puede descartar %s; maneje el valor explícitamente", t.String())
		}
		return err
	case *ast.AssignStmt:
		target, err := c.assignmentTarget(statement.Target)
		if err != nil {
			return err
		}
		if index, ok := statement.Target.(*ast.IndexExpr); ok && statement.Compound != "" {
			if objectType, known := c.model.ExprTypes[index.Object]; known && objectType.Kind == Map {
				if err := c.defaultCompoundMapRead(statement, *objectType.Elem); err != nil {
					return err
				}
			}
		}
		value, err := c.checkExprExpected(statement.Value, &target)
		if err != nil {
			return err
		}
		if !c.model.Assignable(value, target) {
			return c.fail(statement.Pos, "no se puede asignar %s a %s%s", value.String(), target.String(), c.unmet(value, target))
		}
		return nil
	case *ast.VarDeclStmt:
		if stdlib.Reserved(statement.Name) {
			return c.fail(statement.Pos, "nombre reservado: %s", statement.Name)
		}
		c.model.LocalNames[statement.Name] = true
		if _, exists := c.vars[statement.Name]; exists {
			return c.fail(statement.Pos, "la variable %q ya fue declarada", statement.Name)
		}
		var declared *Type
		if statement.Type != nil {
			resolved, err := c.resolveType(*statement.Type)
			if err != nil {
				return err
			}
			declared = &resolved
		}
		actual, err := c.checkExprExpected(statement.Value, declared)
		if err != nil {
			return err
		}
		if actual.Kind == Void {
			return c.fail(statement.Value.Position(), "una variable requiere un valor")
		}
		if declared != nil && !c.model.Assignable(actual, *declared) {
			return c.fail(statement.Pos, "no se puede asignar %s a %s%s", actual.String(), declared.String(), c.unmet(actual, *declared))
		}
		c.vars[statement.Name] = actual
		c.model.VarTypes[statement] = actual
		c.bindings[statement.Name] = statement
		if actual.Kind == Result {
			c.reads[statement] = false
		}
		return nil
	case *ast.IfStmt:
		for _, branch := range statement.Branches {
			outer := c.vars
			c.vars = cloneVars(outer)
			err := c.checkCondition(branch.Condition, branch.Binding, branch.BindingPos)
			if err != nil {
				c.report(err)
				c.damaged++
				if branch.Binding != "" {
					c.vars[branch.Binding] = Type{Kind: Invalid}
				}
			}
			c.checkNestedBlock(branch.Body)
			c.vars = outer
		}
		c.checkNestedBlock(statement.Else)
		return nil
	case *ast.ScopeStmt:
		t, err := c.checkExpr(statement.Value)
		if err != nil {
			return err
		}
		methods := c.model.Methods(t)
		for _, name := range []string{"entrar", "salir"} {
			if t.Kind == Named {
				if _, err := c.member(t, name, statement.Pos); err != nil {
					return err
				}
			}
			method, ok := methods[name]
			if !ok || len(method.Params) != 0 || method.Return.Kind != Void {
				return c.fail(statement.Pos, "con requiere entrar() y salir() sin parámetros ni resultado")
			}
		}
		c.checkNestedBlock(statement.Body)
		return nil
	case *ast.RepeatStmt:
		return c.checkRepeat(statement)
	case *ast.ContinueStmt:
		if c.loopDepth == 0 {
			return c.fail(statement.Pos, "'continuar' solo puede usarse dentro de un ciclo")
		}
		return nil
	case *ast.BreakStmt:
		if c.loopDepth == 0 {
			return c.fail(statement.Pos, "'romper' solo puede usarse dentro de un ciclo")
		}
		return nil
	default:
		return c.fail(stmt.Position(), "sentencia no compatible")
	}
}

func (c *checker) checkRepeat(stmt *ast.RepeatStmt) error {
	defer c.bindingScope()()
	c.model.LocalNames[stmt.Element] = true
	c.model.LocalNames[stmt.Index] = true
	outer := c.vars
	c.vars = make(map[string]Type, len(outer)+2)
	for name, variableType := range outer {
		c.vars[name] = variableType
	}
	defer func() { c.vars = outer }()

	if stmt.Iterable != nil {
		iterable, err := c.checkExpr(stmt.Iterable)
		if err != nil {
			c.report(err)
			c.damaged++
		}
		elementType := Type{Kind: Integer}
		if stmt.RangeEnd != nil {
			if err == nil && iterable.Kind != Integer {
				c.report(c.fail(stmt.Iterable.Position(), "el inicio del rango debe ser entero, no %s", iterable.String()))
				c.damaged++
			}
			end, err := c.checkExpr(stmt.RangeEnd)
			if err != nil {
				c.report(err)
				c.damaged++
			} else if end.Kind != Integer {
				c.report(c.fail(stmt.RangeEnd.Position(), "el final del rango debe ser entero, no %s", end.String()))
				c.damaged++
			}
			c.checkRangeStep(stmt)
		} else if err != nil {
			elementType = Type{Kind: Invalid}
		} else if iterable.Kind == Integer {
			// `repetir (n)` gives n turns, counted from 0 like `repetir (0..n)`.
			c.model.CountLoops[stmt] = true
		} else if iterable.Kind != String && iterable.Kind != Slice && iterable.Kind != Map {
			c.report(c.fail(stmt.Iterable.Position(), "repetir requiere una lista, un mapa, una cadena o una cantidad entera de vueltas, no %s", iterable.String()))
			c.damaged++
			elementType = Type{Kind: Invalid}
		} else if stmt.Element == "" {
			c.report(c.fail(stmt.Iterable.Position(), "para recorrer %s falta el nombre de cada elemento: repetir (...) |elemento|", iterable.String()))
			c.damaged++
		} else if iterable.Kind == String {
			// Una cadena se recorre letra por letra, como dividir("").
			elementType = Type{Kind: String}
		} else {
			elementType = *iterable.Elem
		}
		if stmt.Element != "" {
			if _, exists := c.vars[stmt.Element]; exists {
				return c.fail(stmt.Pos, "la variable %q ya fue declarada", stmt.Element)
			}
			c.vars[stmt.Element] = elementType
		}
		if stmt.Index != "" {
			if stmt.Index == stmt.Element {
				return c.fail(stmt.Pos, "las variables de elemento e índice deben tener nombres distintos")
			}
			if _, exists := c.vars[stmt.Index]; exists {
				return c.fail(stmt.Pos, "la variable %q ya fue declarada", stmt.Index)
			}
			c.vars[stmt.Index] = Type{Kind: Integer}
			if iterable.Kind == Map && stmt.RangeEnd == nil {
				c.vars[stmt.Index] = *iterable.Key
			}
		}
	}

	c.loopDepth++
	defer func() { c.loopDepth-- }()
	body := stmt.Body
	if stmt.While {
		// Report a condition that is not bool in the terms the user wrote,
		// not as a misuse of the `!` of the exit test.
		condition := body[0].(*ast.IfStmt).Branches[0].Condition.(*ast.UnaryExpr).Value
		if t, err := c.checkExpr(condition); err != nil {
			c.report(err)
			c.damaged++
			body = body[1:]
		} else if t.Kind != Boolean && t.Kind != Never {
			c.report(c.fail(condition.Position(), "la condición de mientras debe ser bool, no %s", t.String()))
			c.damaged++
			body = body[1:]
		}
	}
	for _, statement := range body {
		c.checkStmt(statement)
	}
	return nil
}

// checkRangeStep validates the `paso` of a range. A range never changes
// direction at run time: without a step it counts up, so literal bounds written
// backwards are reported instead of silently giving no turns.
func (c *checker) checkRangeStep(stmt *ast.RepeatStmt) {
	if stmt.Step == nil {
		start, _ := c.scalarLiteralKey(stmt.Iterable, 0)
		end, _ := c.scalarLiteralKey(stmt.RangeEnd, 0)
		from, okFrom := integerKey(start)
		to, okTo := integerKey(end)
		if okFrom && okTo && from > to {
			c.report(c.fail(stmt.Pos, "el rango %d..%d no da ninguna vuelta porque el inicio es mayor que el fin; para contar hacia atrás escribe repetir (%d..%d, paso = -1)", from, to, from, to))
			c.damaged++
		}
		return
	}
	step, err := c.checkExpr(stmt.Step)
	if err != nil {
		c.report(err)
		c.damaged++
		return
	}
	if step.Kind != Integer {
		c.report(c.fail(stmt.Step.Position(), "el paso del rango debe ser entero, no %s", step.String()))
		c.damaged++
		return
	}
	key, _ := c.scalarLiteralKey(stmt.Step, 0)
	if n, ok := integerKey(key); ok && n == 0 {
		c.report(c.fail(stmt.Step.Position(), "el paso de un rango no puede ser 0"))
		c.damaged++
	}
}

func (c *checker) checkNestedBlock(body []ast.Stmt) {
	defer c.bindingScope()()
	outer := c.vars
	c.vars = make(map[string]Type, len(outer))
	for name, variableType := range outer {
		c.vars[name] = variableType
	}
	defer func() { c.vars = outer }()
	for _, statement := range body {
		c.checkStmt(statement)
	}
}

func (c *checker) assignmentTarget(expr ast.Expr) (Type, error) {
	if id, ok := expr.(*ast.IdentExpr); ok {
		if t, exists := c.vars[id.Name]; exists && t.Kind == Invalid || !exists && c.invalid[id.Name] {
			return Type{}, errInvalid
		}
	}
	switch target := expr.(type) {
	case *ast.IndexExpr:
		t, err := c.checkExpr(target)
		if err == nil && c.model.ExprTypes[target.Object].Kind == Map {
			return *c.model.ExprTypes[target.Object].Elem, nil
		}
		return t, err
	case *ast.IdentExpr:
		valueType, exists := c.vars[target.Name]
		if !exists {
			global := c.model.Globals[target.Name]
			if global == nil {
				return Type{}, c.fail(target.Pos, "el nombre %q no existe%s", target.Name, c.nameHint(target.Name))
			}
			if global.Constant {
				return Type{}, c.fail(target.Pos, "la constante %q no puede reasignarse", target.Name)
			}
			if err := c.checkGlobal(target.Name); err != nil {
				return Type{}, err
			}
			valueType = global.Type
			c.model.GlobalRefs[target] = global
		}
		c.model.ExprTypes[target] = valueType
		return valueType, nil
	case *ast.ReceiverExpr:
		if target.Name == "" {
			return Type{}, c.fail(target.Pos, "no se puede asignar a '@'")
		}
		if c.receiver == nil {
			return Type{}, c.fail(target.Pos, "'@%s' solo puede usarse dentro de un método", target.Name)
		}
		member, err := c.member(c.receiverType(), target.Name, target.Pos)
		if err != nil {
			return Type{}, err
		}
		field := member.Field
		if field.Decl == nil {
			return Type{}, c.fail(target.Pos, "el campo %q no existe en %s%s", target.Name, c.receiver.Decl.Name, diagnostic.Hint(target.Name, sortedKeys(c.model.Members(c.receiverType()))))
		}
		c.model.ExprTypes[target] = field.Type
		return field.Type, nil
	case *ast.MemberExpr:
		objectType, err := c.checkExpr(target.Object)
		if err != nil {
			return Type{}, err
		}
		if objectType.Kind != Named {
			return Type{}, c.fail(target.Pos, "%s no tiene campos", objectType.String())
		}
		if native, ok := stdlib.LookupType(objectType.Name); ok && native.Channel(target.Name) != "" {
			return Type{}, c.fail(target.Pos, "%s", native.ReadOnly)
		}
		if stdlib.IsValue(objectType.Name) {
			if _, err := c.assignmentTarget(target.Object); err != nil {
				return Type{}, c.fail(target.Pos, "no se puede asignar un campo de un valor temporal")
			}
		}
		member, err := c.member(objectType, target.Name, target.Pos)
		if err != nil {
			return Type{}, err
		}
		field := member.Field
		if field.Decl == nil {
			return Type{}, c.fail(target.Pos, "el campo %q no existe en %s%s", target.Name, objectType.Name, diagnostic.Hint(target.Name, sortedKeys(c.model.Members(objectType))))
		}
		c.model.ExprTypes[target] = field.Type
		return field.Type, nil
	default:
		return Type{}, c.fail(expr.Position(), "el lado izquierdo de '=' debe ser un parámetro, campo o elemento de lista")
	}
}
