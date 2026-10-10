package sema

import (
	"sort"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
)

func (c *checker) checkGlobal(name string) (err error) {
	if c.invalid[name] {
		return errInvalid
	}
	defer func() {
		if err != nil {
			c.invalid[name] = true
			c.report(err)
			err = errInvalid
		}
	}()
	info := c.model.Globals[name]
	if info == nil || c.globalState[name] == 2 {
		return nil
	}
	if c.globalState[name] == 1 {
		chain := append(append([]string{}, c.globalStack...), name)
		return c.fail(info.Decl.NamePos, "ciclo de inicialización global: %s", strings.Join(chain, " -> "))
	}
	c.globalState[name] = 1
	c.globalStack = append(c.globalStack, name)
	defer func() { c.globalStack = c.globalStack[:len(c.globalStack)-1] }()

	decl := info.Decl
	if pos, forbidden := forbiddenGlobalControl(decl.Value); forbidden {
		return c.fail(pos, "el inicializador global no admite control de flujo")
	}

	defer c.enterDeclaration(decl, nil)()

	var declared *Type
	if decl.Type != nil {
		resolved, err := c.resolveType(*decl.Type)
		if err != nil {
			return err
		}
		declared = &resolved
	}
	actual, err := c.checkExprExpected(decl.Value, declared)
	if err != nil {
		return err
	}
	if actual.Kind == Void || actual.Kind == Never {
		return c.fail(decl.Value.Position(), "una global requiere un valor")
	}
	if declared != nil && !c.model.Assignable(actual, *declared) {
		return c.fail(decl.Pos, "no se puede asignar %s a %s%s", actual.String(), declared.String(), c.unmet(actual, *declared))
	}
	if declared != nil {
		actual = *declared
	}
	info.Type = actual
	if decl.Constant {
		if !actual.Numeric() && actual.Kind != String && actual.Kind != Boolean {
			return c.fail(decl.Pos, "una constante solo puede ser entero, cadena o bool, no %s", actual.String())
		}
		if err := c.checkConstantExpression(decl.Value); err != nil {
			return err
		}
	}
	c.globalState[name] = 2
	return nil
}

func (c *checker) checkConstantExpression(expr ast.Expr) error {
	switch e := expr.(type) {
	case *ast.MemberExpr:
		return c.fail(e.Pos, "se requiere una expresión constante")
	case *ast.LiteralExpr:
		return nil
	case *ast.IdentExpr:
		if _, ok := stdlib.LookupConstant(e.Name); ok {
			return nil // mate.pi
		}
		global := c.model.Globals[e.Name]
		if global == nil || !global.Constant {
			return c.fail(e.Pos, "una constante solo puede referirse a otras constantes")
		}
		return c.checkGlobal(e.Name)
	case *ast.UnaryExpr:
		return c.checkConstantExpression(e.Value)
	case *ast.BinaryExpr:
		if err := c.checkConstantExpression(e.Left); err != nil {
			return err
		}
		return c.checkConstantExpression(e.Right)
	default:
		return c.fail(expr.Position(), "el inicializador de una constante debe ser una expresión constante simple")
	}
}

// forbiddenGlobalControl finds the first control-flow expression of a global
// initializer, which may only hold plain values and calls.
func forbiddenGlobalControl(expr ast.Expr) (pos ast.Pos, found bool) {
	ast.Inspect(expr, func(node any) bool {
		if found {
			return false
		}
		switch node.(type) {
		case *ast.ReturnExpr, *ast.TryExpr, *ast.RecoverExpr, *ast.BlockExpr, *ast.IfExpr, *ast.MatchExpr:
			pos, found = node.(ast.Expr).Position(), true
			return false
		}
		return true
	})
	return pos, found
}

func (c *checker) checkInitializationCycles(program *ast.Program) error {
	graph := map[string]map[string]bool{}
	for _, declaration := range program.Decls {
		global, ok := declaration.(*ast.GlobalDecl)
		if !ok {
			continue
		}
		dependencies := map[string]bool{}
		c.collectInitializationDependencies(global.Value, dependencies, map[*ast.FuncDecl]bool{})
		graph[global.Name] = dependencies
	}

	state := map[string]int{}
	var stack []string
	var visit func(string) error
	visit = func(name string) error {
		if state[name] == 2 {
			return nil
		}
		if state[name] == 1 {
			start := 0
			for start < len(stack) && stack[start] != name {
				start++
			}
			chain := append(append([]string{}, stack[start:]...), name)
			return c.fail(c.model.Globals[name].Decl.NamePos, "ciclo de inicialización global: %s", strings.Join(chain, " -> "))
		}
		state[name] = 1
		stack = append(stack, name)
		var dependencies []string
		for dependency := range graph[name] {
			dependencies = append(dependencies, dependency)
		}
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = 2
		return nil
	}
	for _, declaration := range program.Decls {
		if global, ok := declaration.(*ast.GlobalDecl); ok {
			if err := visit(global.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// collectInitializationDependencies records the globals that evaluating root
// reads, following calls into the functions they run.
func (c *checker) collectInitializationDependencies(root any, dependencies map[string]bool, functions map[*ast.FuncDecl]bool) {
	ast.Inspect(root, func(node any) bool {
		switch n := node.(type) {
		case *ast.IdentExpr:
			if global := c.model.GlobalRefs[n]; global != nil {
				dependencies[global.Decl.Name] = true
			}
		case *ast.CallExpr:
			if function := c.model.Calls[n].Signature.Decl; function != nil && !functions[function] {
				functions[function] = true
				for _, parameter := range function.Params {
					c.collectInitializationDependencies(parameter.Default, dependencies, functions)
				}
				c.collectInitializationDependencies(function.Body, dependencies, functions)
			}
		}
		return true
	})
}
