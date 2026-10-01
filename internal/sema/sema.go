package sema

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/stdlib"
)

type Kind int

const (
	Invalid Kind = iota
	Void
	Decimal
	Integer
	String
	Boolean
	Named
	Slice
	Map
	Enum
	Optional
	Result
	Never
	Interface
	TypeParameter
)

type Type struct {
	Key   *Type
	Args  []Type
	Owner string // Declaration identity for a type parameter.
	Kind  Kind
	Name  string
	Elem  *Type
	Err   *Type
}

func (t Type) String() string {
	switch t.Kind {
	case Optional:
		if t.Elem.Kind == Result {
			return "(" + t.Elem.String() + ")?"
		}
		return t.Elem.String() + "?"
	case Result:
		payload := t.Elem.String()
		if t.Elem.Kind == Void {
			payload = ""
		}
		if t.Elem.Kind == Result {
			payload = "(" + payload + ")"
		}
		if t.Err.Kind == String {
			return payload + "!"
		}
		errorType := t.Err.String()
		if t.Err.Wrapped() {
			errorType = "(" + errorType + ")"
		}
		return payload + "!" + errorType
	case Never:
		return "no retorna"
	case Void:
		return "sin valor"
	case Decimal:
		return "decimal"
	case Integer:
		return "entero"
	case String:
		return "cadena"
	case Boolean:
		return "bool"
	case Named, Enum, Interface, TypeParameter:
		name := t.Name
		if len(t.Args) > 0 {
			var args []string
			for _, arg := range t.Args {
				args = append(args, arg.String())
			}
			name += "<" + strings.Join(args, ", ") + ">"
		}
		return name
	case Slice:
		return "[" + t.Elem.String() + "]"
	case Map:
		return "[" + t.Key.String() + ": " + t.Elem.String() + "]"
	default:
		return "tipo inválido"
	}
}

func (t Type) Equal(other Type) bool {
	if t.Kind != other.Kind || t.Name != other.Name || t.Owner != other.Owner || len(t.Args) != len(other.Args) {
		return false
	}
	for i := range t.Args {
		if !t.Args[i].Equal(other.Args[i]) {
			return false
		}
	}
	if t.Kind == Map && (t.Key == nil || other.Key == nil || !t.Key.Equal(*other.Key)) {
		return false
	}
	if t.Kind == Slice || t.Kind == Map || t.Wrapped() {
		return t.Elem != nil && other.Elem != nil && t.Elem.Equal(*other.Elem) && (t.Kind != Result || t.Err.Equal(*other.Err))
	}
	return true
}

type FieldInfo struct {
	Decl *ast.Field
	Type Type
}

type FuncInfo struct {
	EmbeddedPath []string
	TypeParams   []Type
	Constraints  []Type
	TypeArgs     []Type
	Decl         *ast.FuncDecl
	Params       []Type
	Return       Type
}

type TypeInfo struct {
	Decl    *ast.TypeDecl
	Fields  map[string]FieldInfo
	Methods map[string]FuncInfo
}

type GlobalInfo struct {
	Decl     *ast.GlobalDecl
	Type     Type
	Constant bool
}

type Model struct {
	HasScopes        bool
	NumericCoercions map[ast.Expr]Type
	NumericCalls     map[*ast.CallExpr]Type
	Assets           map[*ast.CallExpr]Asset
	Game             GameModel
	Interfaces       map[string]*InterfaceInfo
	TypeParams       map[ast.Decl][]Type
	Constraints      map[string]Type
	RawTypes         map[ast.Expr]Type
	TypeRefs         map[*ast.TypeRef]Type
	Wraps            map[ast.Expr]Type
	Calls            map[*ast.CallExpr]CallInfo
	ListCalls        map[*ast.CallExpr]string
	MapCalls         map[*ast.CallExpr]string
	StringCalls      map[*ast.CallExpr]string
	NumberCalls      map[*ast.CallExpr]string
	StructCopies     map[*ast.CallExpr]Type // built-in copiar() calls on user structs
	EnumCompares     map[*ast.BinaryExpr]EnumCompare
	LocalNames       map[string]bool
	Enums            map[string]*EnumInfo
	Constructors     map[ast.Expr]ConstructorInfo
	Types            map[string]*TypeInfo
	Functions        map[string]FuncInfo
	Globals          map[string]*GlobalInfo
	GlobalRefs       map[*ast.IdentExpr]*GlobalInfo
	ExprTypes        map[ast.Expr]Type
	ExpectedTypes    map[ast.Expr]Type
	PatternTypes     map[*ast.MatchArm]Type
	VarTypes         map[*ast.VarDeclStmt]Type
}

type Error struct {
	Filename string
	Pos      ast.Pos
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Filename, e.Pos.Line, e.Pos.Column, e.Message)
}

func (e *Error) Diagnostic() (string, ast.Pos, string, string) {
	return e.Filename, e.Pos, "sema", e.Message
}

// errInvalid propagates a previously diagnosed failure without cascading errors.
var errInvalid = errors.New("fragmento inválido")

func (c *checker) report(err error) {
	for _, e := range diagnostic.Flatten(err) {
		if e != errInvalid {
			c.errors.Add(e)
		}
	}
}

// Graph-wide validations require complete declarations. Local checks remain useful.
func (c *checker) safeCheck(check func() error) error {
	if len(c.invalid) != 0 || len(c.errors) != 0 {
		return nil
	}
	return check()
}

type checker struct {
	errors      diagnostic.List
	invalid     map[string]bool
	damaged     int
	typeScope   map[string]Type
	pending     []typeUse
	expansions  []typeEdge
	returnType  Type
	inDefault   bool
	bindings    map[string]*ast.VarDeclStmt
	reads       map[*ast.VarDeclStmt]bool
	tooling     bool
	filename    string
	model       *Model
	vars        map[string]Type
	receiver    *TypeInfo
	loopDepth   int
	globalState map[string]int
	globalStack []string
}

func Check(filename string, program *ast.Program) (*Model, error) {
	c := newChecker(filename)
	_, err := c.check(program)
	c.report(err)
	return c.model, c.errors.Err()
}

// CheckForTooling retains successfully resolved information before an error.
// It is only for editor queries on incomplete documents, never compilation.
func CheckForTooling(filename string, program *ast.Program) (*Model, error) {
	c := newChecker(filename)
	c.tooling = true
	_, err := c.check(program)
	c.report(err)
	return c.model, c.errors.Err()
}

func newChecker(filename string) *checker {
	return &checker{filename: filename, invalid: map[string]bool{}, model: &Model{
		NumericCoercions: map[ast.Expr]Type{}, NumericCalls: map[*ast.CallExpr]Type{},
		Interfaces: map[string]*InterfaceInfo{}, TypeParams: map[ast.Decl][]Type{}, Constraints: map[string]Type{}, RawTypes: map[ast.Expr]Type{}, TypeRefs: map[*ast.TypeRef]Type{},
		Wraps:         map[ast.Expr]Type{},
		Calls:         map[*ast.CallExpr]CallInfo{},
		ListCalls:     map[*ast.CallExpr]string{},
		MapCalls:      map[*ast.CallExpr]string{},
		StringCalls:   map[*ast.CallExpr]string{},
		NumberCalls:   map[*ast.CallExpr]string{},
		StructCopies:  map[*ast.CallExpr]Type{},
		EnumCompares:  map[*ast.BinaryExpr]EnumCompare{},
		LocalNames:    map[string]bool{},
		ExpectedTypes: map[ast.Expr]Type{},
		PatternTypes:  map[*ast.MatchArm]Type{},
		Enums:         map[string]*EnumInfo{}, Constructors: map[ast.Expr]ConstructorInfo{},
		Types: map[string]*TypeInfo{}, Functions: map[string]FuncInfo{}, Globals: map[string]*GlobalInfo{}, GlobalRefs: map[*ast.IdentExpr]*GlobalInfo{}, ExprTypes: map[ast.Expr]Type{}, VarTypes: map[*ast.VarDeclStmt]Type{},
	}}

}

func (c *checker) signature(function *ast.FuncDecl) (FuncInfo, error) {
	info := FuncInfo{Decl: function, Return: Type{Kind: Void}}
	broken := false
	seen := map[string]bool{}
	optional := false
	for index, param := range function.Params {
		err := func() error {
			if param.Default != nil {
				if param.Variadic {
					return c.fail(param.Pos, "un parámetro variádico no acepta un valor predeterminado")
				}
				optional = true
			} else if optional && !param.Variadic {
				return c.fail(param.Pos, "un parámetro obligatorio no puede seguir a uno con valor predeterminado")
			}
			if param.Variadic && index != len(function.Params)-1 {
				return c.fail(param.Pos, "el parámetro variádico debe ser el último")
			}
			if seen[param.Name] {
				return c.fail(param.Pos, "el parámetro %q está duplicado", param.Name)
			}
			seen[param.Name] = true
			paramType, err := c.resolveType(param.Type)
			if err != nil {
				return err
			}
			if param.Variadic {
				element := paramType
				paramType = Type{Kind: Slice, Elem: &element}
			}
			info.Params = append(info.Params, paramType)
			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}

	}
	if function.ReturnType != nil {
		result, err := c.resolveType(*function.ReturnType)
		if err != nil {
			return FuncInfo{}, err
		}
		info.Return = result
	}
	if broken {
		return FuncInfo{}, errInvalid
	}
	return info, nil
}

func (c *checker) resolveBuiltinType(ref ast.TypeRef) (Type, error) {
	if ref.Wrapper != "" {
		payload, err := c.resolveType(*ref.Payload)
		if err != nil {
			return Type{}, err
		}
		t := Type{Kind: Optional, Elem: &payload}
		if ref.Wrapper == "!" {
			t.Kind = Result
			e, err := c.resolveType(*ref.ErrorType)
			if err != nil {
				return Type{}, err
			}
			t.Err = &e
		}
		return t, nil
	}
	if ref.Key != nil {
		key, err := c.resolveType(*ref.Key)
		if err != nil {
			return Type{}, err
		}
		if !mapKeyAllowed(key) {
			return Type{}, c.fail(ref.Key.Pos, "una clave de mapa debe ser cadena, entero o bool, no %s", key.String())
		}
		element, err := c.resolveType(*ref.Element)
		if err != nil {
			return Type{}, err
		}
		if element.Kind == Void {
			return Type{}, c.fail(ref.Pos, "un mapa requiere valores")
		}
		return Type{Kind: Map, Key: &key, Elem: &element}, nil
	}
	if ref.IsSlice() {
		element, err := c.resolveType(*ref.Element)
		if err != nil {
			return Type{}, err
		}
		return Type{Kind: Slice, Elem: &element}, nil
	}
	switch ref.Name {
	case "$unidad":
		return Type{Kind: Void}, nil
	case "num":
		return Type{}, c.fail(ref.Pos, "el tipo num fue eliminado; use entero o decimal")
	case "entero":
		return Type{Kind: Integer}, nil
	case "decimal":
		return Type{Kind: Decimal}, nil
	case "cadena":
		return Type{Kind: String}, nil
	case "bool":
		return Type{Kind: Boolean}, nil
	default:
		if c.model.Enums[ref.Name] != nil {
			return Type{Kind: Enum, Name: ref.Name}, nil
		}
		if _, exists := c.model.Types[ref.Name]; !exists {
			return Type{}, c.fail(ref.Pos, "el tipo %q no existe", ref.Name)
		}
		return Type{Kind: Named, Name: ref.Name}, nil
	}
}

func (c *checker) checkFunction(function *ast.FuncDecl, receiver *TypeInfo) error {
	oldVars, oldBindings, oldReads, oldScope := c.vars, c.bindings, c.reads, c.typeScope
	oldReceiver, oldReturn, oldDepth, oldDefault := c.receiver, c.returnType, c.loopDepth, c.inDefault
	defer func() {
		c.vars, c.bindings, c.reads, c.typeScope = oldVars, oldBindings, oldReads, oldScope
		c.receiver, c.returnType, c.loopDepth, c.inDefault = oldReceiver, oldReturn, oldDepth, oldDefault
	}()
	before := c.damaged
	for _, p := range function.Params {
		if stdlib.Reserved(p.Name) {
			return c.fail(p.Pos, "nombre de parámetro reservado: %s", p.Name)
		}
	}
	if receiver != nil {
		c.setScope(receiver.Decl)
	} else {
		c.setScope(function)
	}
	c.bindings = map[string]*ast.VarDeclStmt{}
	c.reads = map[*ast.VarDeclStmt]bool{}
	c.vars = map[string]Type{}
	c.receiver = receiver
	c.loopDepth = 0
	for index, param := range function.Params {
		c.model.LocalNames[param.Name] = true
		expected := c.mustResolve(function, index)
		if param.Default != nil {
			c.inDefault = true
			actual, err := c.checkExprExpected(param.Default, &expected)
			c.inDefault = false
			if err != nil {
				c.report(err)
				c.damaged++
			} else if !c.model.Assignable(actual, expected) {
				c.report(c.fail(param.Default.Position(), "el valor predeterminado de %q debe ser %s, no %s", param.Name, expected.String(), actual.String()))
				c.damaged++
			}
		}
		c.vars[param.Name] = expected
	}
	var expected Type
	if receiver == nil {
		expected = c.model.Functions[function.Name].Return
	} else {
		expected = receiver.Methods[function.Name].Return
	}
	c.returnType = expected
	if expected.Kind != Void {
		actual, err := c.checkValueBlock(function.Body, &expected)
		if err != nil {
			return err
		}
		if !c.model.Assignable(actual, expected) {
			return c.fail(function.Pos, "la función %q produce %s, pero declara %s", function.Name, actual.String(), expected.String())
		}
	} else {
		for _, stmt := range function.Body {
			if err := c.checkStmt(stmt); err != nil {
				return err
			}
		}
	}
	var unread []*ast.VarDeclStmt
	for binding, read := range c.reads {
		if !read {
			unread = append(unread, binding)
		}
	}
	sort.Slice(unread, func(i, j int) bool {
		if unread[i].Pos.Line != unread[j].Pos.Line {
			return unread[i].Pos.Line < unread[j].Pos.Line
		}
		return unread[i].Pos.Column < unread[j].Pos.Column
	})
	if c.damaged == before {
		for _, binding := range unread {
			c.report(c.fail(binding.Pos, "el resultado %q debe usarse o manejarse explícitamente", binding.Name))
		}
	}
	return nil
}

func (c *checker) checkFieldDefaults(d *ast.TypeDecl) error {
	oldVars, oldBindings, oldReads, oldScope := c.vars, c.bindings, c.reads, c.typeScope
	oldReceiver, oldDefault := c.receiver, c.inDefault
	defer func() {
		c.vars, c.bindings, c.reads, c.typeScope = oldVars, oldBindings, oldReads, oldScope
		c.receiver, c.inDefault = oldReceiver, oldDefault
	}()
	c.setScope(d)
	c.vars = map[string]Type{}
	c.bindings = map[string]*ast.VarDeclStmt{}
	c.reads = map[*ast.VarDeclStmt]bool{}
	c.receiver = nil
	info := c.model.Types[d.Name]
	for _, field := range d.Fields {
		if field.Default == nil {
			continue
		}
		expected := info.Fields[field.Name].Type
		c.inDefault = true
		actual, err := c.checkExprExpected(field.Default, &expected)
		c.inDefault = false
		if err != nil {
			c.report(err)
			c.damaged++
			continue
		}
		if !c.model.Assignable(actual, expected) {
			c.report(c.fail(field.Default.Position(), "el valor predeterminado de %q debe ser %s, no %s", field.Name, expected.String(), actual.String()))
			c.damaged++
		}
	}
	return nil
}

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

	oldScope, oldVars, oldBindings, oldReads := c.typeScope, c.vars, c.bindings, c.reads
	oldReceiver, oldReturn, oldLoop := c.receiver, c.returnType, c.loopDepth
	c.setScope(decl)
	c.vars = map[string]Type{}
	c.bindings = map[string]*ast.VarDeclStmt{}
	c.reads = map[*ast.VarDeclStmt]bool{}
	c.receiver = nil
	c.returnType = Type{Kind: Void}
	c.loopDepth = 0
	defer func() {
		c.typeScope, c.vars, c.bindings, c.reads = oldScope, oldVars, oldBindings, oldReads
		c.receiver, c.returnType, c.loopDepth = oldReceiver, oldReturn, oldLoop
	}()

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
		return c.fail(decl.Pos, "no se puede asignar %s a %s", actual.String(), declared.String())
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
	if c.model.Game.Constants[expr] == "math.Pi" {
		return nil
	}
	switch e := expr.(type) {
	case *ast.MemberExpr:
		if c.model.Game.Constants[e] == "math.Pi" {
			return nil
		}
		return c.fail(e.Pos, "se requiere una expresión constante")
	case *ast.LiteralExpr:
		return nil
	case *ast.IdentExpr:
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

func forbiddenGlobalControl(expr ast.Expr) (ast.Pos, bool) {
	return forbiddenGlobalValue(reflect.ValueOf(expr))
}

func forbiddenGlobalValue(value reflect.Value) (ast.Pos, bool) {
	if !value.IsValid() {
		return ast.Pos{}, false
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ast.Pos{}, false
		}
		if node, ok := value.Interface().(ast.Expr); ok {
			switch node.(type) {
			case *ast.ReturnExpr, *ast.TryExpr, *ast.RecoverExpr, *ast.BlockExpr, *ast.IfExpr, *ast.MatchExpr:
				return node.Position(), true
			}
		}
		return forbiddenGlobalValue(value.Elem())
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if pos, found := forbiddenGlobalValue(value.Field(i)); found {
				return pos, true
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if pos, found := forbiddenGlobalValue(value.Index(i)); found {
				return pos, true
			}
		}
	}
	return ast.Pos{}, false
}

func (c *checker) checkInitializationCycles(program *ast.Program) error {
	graph := map[string]map[string]bool{}
	for _, declaration := range program.Decls {
		global, ok := declaration.(*ast.GlobalDecl)
		if !ok {
			continue
		}
		dependencies := map[string]bool{}
		c.collectInitializationDependencies(reflect.ValueOf(global.Value), dependencies, map[*ast.FuncDecl]bool{})
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

func (c *checker) collectInitializationDependencies(value reflect.Value, dependencies map[string]bool, functions map[*ast.FuncDecl]bool) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		if identifier, ok := value.Interface().(*ast.IdentExpr); ok {
			if global := c.model.GlobalRefs[identifier]; global != nil {
				dependencies[global.Decl.Name] = true
			}
		}
		if call, ok := value.Interface().(*ast.CallExpr); ok {
			if function := c.model.Calls[call].Signature.Decl; function != nil && !functions[function] {
				functions[function] = true
				for _, parameter := range function.Params {
					c.collectInitializationDependencies(reflect.ValueOf(parameter.Default), dependencies, functions)
				}
				c.collectInitializationDependencies(reflect.ValueOf(function.Body), dependencies, functions)
			}
		}
		c.collectInitializationDependencies(value.Elem(), dependencies, functions)
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			c.collectInitializationDependencies(value.Field(i), dependencies, functions)
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			c.collectInitializationDependencies(value.Index(i), dependencies, functions)
		}
	}
}

func (c *checker) mustResolve(function *ast.FuncDecl, index int) Type {
	t, _ := c.resolveType(function.Params[index].Type)
	if function.Params[index].Variadic {
		return Type{Kind: Slice, Elem: &t}
	}
	return t
}

func (c *checker) checkStmt(stmt ast.Stmt) error {
	if bad, ok := stmt.(*ast.BadStmt); ok {
		c.damaged++
		if bad.Name != "" {
			c.vars[bad.Name] = Type{Kind: Invalid}
		}
		return nil
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
	return nil
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
			return c.fail(statement.Pos, "no se puede asignar %s a %s", value.String(), target.String())
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
			return c.fail(statement.Pos, "no se puede asignar %s a %s", actual.String(), declared.String())
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
		if err := c.checkNestedBlock(statement.Else); err != nil {
			return err
		}
		return nil
	case *ast.ScopeStmt:
		c.model.HasScopes = true
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
		return c.checkNestedBlock(statement.Body)
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
		} else if err != nil {
			elementType = Type{Kind: Invalid}
		} else if iterable.Kind != Slice && iterable.Kind != Map {
			c.report(c.fail(stmt.Iterable.Position(), "repetir requiere una lista o mapa, no %s", iterable.String()))
			c.damaged++
			elementType = Type{Kind: Invalid}
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
	for _, statement := range stmt.Body {
		if err := c.checkStmt(statement); err != nil {
			return err
		}
	}
	return nil
}

func (c *checker) checkNestedBlock(body []ast.Stmt) error {
	defer c.bindingScope()()
	outer := c.vars
	c.vars = make(map[string]Type, len(outer))
	for name, variableType := range outer {
		c.vars[name] = variableType
	}
	defer func() { c.vars = outer }()
	for _, statement := range body {
		if err := c.checkStmt(statement); err != nil {
			return err
		}
	}
	return nil
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

func (c *checker) checkExpr(expr ast.Expr) (Type, error) {
	return c.checkExprExpected(expr, nil)
}

func (c *checker) checkExprExpected(expr ast.Expr, expected *Type) (Type, error) {
	t, err := c.checkExpression(expr, expected)
	if err != nil {
		c.report(err)
		return Type{Kind: Invalid}, errInvalid
	}
	if t.Kind == Invalid {
		return t, errInvalid
	}
	return t, nil
}

func (c *checker) checkExpression(expr ast.Expr, expected *Type) (Type, error) {
	if id, ok := expr.(*ast.IdentExpr); ok {
		if t, exists := c.vars[id.Name]; exists {
			if t.Kind == Invalid {
				return t, errInvalid
			}
		} else if c.invalid[id.Name] {
			return Type{}, errInvalid
		}
	}
	delete(c.model.NumericCoercions, expr)
	delete(c.model.Wraps, expr)
	delete(c.model.RawTypes, expr)
	if expected != nil && expected.Kind == Never {
		expected = nil
	}
	if expected != nil {
		c.model.ExpectedTypes[expr] = *expected
	}
	wrapperExpected := expected
	if expected != nil && expected.Wrapped() && !isContextualConstructor(expr) {
		switch expr.(type) {
		case *ast.StructLiteralExpr, *ast.ListLiteralExpr, *ast.MapLiteralExpr:
			expected = expected.Elem
		}
	}
	var result Type
	var err error
	switch expression := expr.(type) {
	case *ast.AssertExpr:
		result, err = c.checkAssertion(expression)
	case *ast.ReturnExpr:
		result, err = c.checkReturn(expression)
	case *ast.TryExpr:
		result, err = c.checkTry(expression)
	case *ast.RecoverExpr:
		result, err = c.checkRecovery(expression)
	case *ast.BlockExpr:
		result, err = c.checkValueBlock(expression.Body, expected)
	case *ast.ContextualVariantExpr:
		result, err = c.checkContextualVariant(expression, nil, expected)
	case *ast.LiteralExpr:
		switch expression.Kind {
		case "entero":
			result = Type{Kind: Integer}
			if _, parseErr := strconv.ParseInt(expression.Value, 10, 64); parseErr != nil {
				err = c.fail(expression.Pos, "literal entero fuera del rango de entero (el máximo es 9223372036854775807)")
			}
		case "decimal":
			result = Type{Kind: Decimal}
			if _, parseErr := strconv.ParseFloat(expression.Value, 64); parseErr != nil {
				err = c.fail(expression.Pos, "literal decimal fuera de rango")
			}
		case "cadena":
			if _, unquoteErr := strconv.Unquote(expression.Value); unquoteErr != nil {
				err = c.fail(expression.Pos, "cadena inválida: %v", unquoteErr)
			} else {
				result = Type{Kind: String}
			}
		case "bool":
			result = Type{Kind: Boolean}
		}
	case *ast.InterpolatedStringExpr:
		for _, part := range expression.Parts {
			if part.Expr == nil {
				continue
			}
			t, checkErr := c.checkExpr(part.Expr)
			if checkErr != nil {
				err = checkErr
				break
			}
			if t.Kind != String && !t.Numeric() && t.Kind != Boolean && t.Kind != Never {
				err = c.fail(part.Expr.Position(), "la interpolación requiere cadena, entero, decimal o bool, no %s", t.String())
				break
			}
		}
		if err == nil {
			result = Type{Kind: String}
		}
	case *ast.IdentExpr:
		if expression.Name == stdlib.FunctionSymbol("mate", "pi") {
			result = Type{Kind: Decimal}
			c.model.Game.Constants[expression] = "math.Pi"
			break
		}
		if binding := c.bindings[expression.Name]; binding != nil {
			if _, tracked := c.reads[binding]; tracked {
				c.reads[binding] = true
			}
		}
		var exists bool
		result, exists = c.vars[expression.Name]
		if !exists {
			if global := c.model.Globals[expression.Name]; global != nil {
				err = c.checkGlobal(expression.Name)
				result = global.Type
				c.model.GlobalRefs[expression] = global
			} else {
				err = c.fail(expression.Pos, "el nombre %q no existe%s", expression.Name, c.nameHint(expression.Name))
			}
		}
	case *ast.ReceiverExpr:
		if c.receiver == nil {
			if expression.Name == "" {
				err = c.fail(expression.Pos, "'@' solo puede usarse dentro de un método")
			} else {
				err = c.fail(expression.Pos, "'@%s' solo puede usarse dentro de un método", expression.Name)
			}
			break
		}
		if expression.Name == "" {
			result = c.receiverType()
			break
		}
		member, memberErr := c.member(c.receiverType(), expression.Name, expression.Pos)
		if memberErr != nil {
			err = memberErr
			break
		}
		field := member.Field
		if field.Decl == nil {
			if member.Method.Decl != nil {
				err = c.fail(expression.Pos, "el método @%s debe llamarse con paréntesis", expression.Name)
			} else {
				err = c.fail(expression.Pos, "el miembro %q no existe en %s%s", expression.Name, c.receiver.Decl.Name, diagnostic.Hint(expression.Name, sortedKeys(c.model.Members(c.receiverType()))))
			}
		} else {
			result = field.Type
		}
	case *ast.MemberExpr:
		if id, ok := expression.Object.(*ast.IdentExpr); ok && id.Name == stdlib.Symbol("mate") && expression.Name == "pi" {
			result = Type{Kind: Decimal}
			c.model.Game.Constants[expression] = "math.Pi"
			c.model.Game.Used = true
			break
		}
		if info, variant, found, enumErr := c.enumMember(expression); found {
			err = enumErr
			if err == nil {
				if variant.Payload.Kind != Void {
					err = c.fail(expression.NamePos, "la variante %s.%s requiere un payload", info.Decl.Name, expression.Name)
				} else {
					result = info.Type
					c.model.Constructors[expression] = ConstructorInfo{Enum: info, Variant: variant}
				}
			}
			break
		}
		objectType, objectErr := c.checkExpr(expression.Object)
		if objectErr != nil {
			err = objectErr
			break
		}
		if objectType.Kind == Interface || objectType.Kind == TypeParameter {
			if _, exists := c.model.Methods(objectType)[expression.Name]; exists {
				err = c.fail(expression.Pos, "el método %s.%s debe llamarse con paréntesis", objectType.String(), expression.Name)
			} else {
				err = c.fail(expression.Pos, "el miembro %q no existe en %s%s", expression.Name, objectType.String(), diagnostic.Hint(expression.Name, sortedKeys(c.model.Methods(objectType))))
			}
			break
		}
		if objectType.Kind == String {
			if _, exists := StringMethods()[expression.Name]; exists {
				err = c.fail(expression.Pos, "el método cadena.%s debe llamarse con paréntesis", expression.Name)
			} else {
				err = c.fail(expression.Pos, "el miembro %q no existe en cadena%s", expression.Name, diagnostic.Hint(expression.Name, sortedKeys(StringMethods())))
			}
			break
		}
		if objectType.Kind != Named {
			err = c.fail(expression.Pos, "%s no tiene miembros", objectType.String())
			break
		}
		member, memberErr := c.member(objectType, expression.Name, expression.Pos)
		if memberErr != nil {
			err = memberErr
			break
		}
		if member.Field.Decl != nil {
			result = member.Field.Type
		} else if member.Method.Decl != nil {
			err = c.fail(expression.Pos, "el método %s.%s debe llamarse con paréntesis", objectType.Name, expression.Name)
		} else {
			err = c.fail(expression.Pos, "el miembro %q no existe en %s%s", expression.Name, objectType.Name, diagnostic.Hint(expression.Name, sortedKeys(c.model.Members(objectType))))
		}
	case *ast.IndexExpr:
		result, err = c.checkIndex(expression)
	case *ast.StructLiteralExpr:
		result, err = c.checkStructLiteral(expression, expected)
	case *ast.ListLiteralExpr:
		result, err = c.checkListLiteral(expression, expected)
	case *ast.MapLiteralExpr:
		result, err = c.checkMapLiteral(expression, expected)
	case *ast.UnaryExpr:
		result, err = c.checkUnary(expression)
	case *ast.BinaryExpr:
		result, err = c.checkBinary(expression)
	case *ast.CallExpr:
		if variant, ok := expression.Callee.(*ast.ContextualVariantExpr); ok {
			if expected != nil {
				c.model.ExpectedTypes[variant] = *expected
			}
			result, err = c.checkContextualVariant(variant, expression, expected)
		} else {
			result, err = c.checkCall(expression)
		}
	case *ast.IfExpr:
		result, err = c.checkIfExpr(expression, expected)
	case *ast.MatchExpr:
		result, err = c.checkMatch(expression, true, expected)
	default:
		err = c.fail(expr.Position(), "expresión no compatible")
	}
	if err == nil {
		if c.model.eagerExit(expr) {
			result = Type{Kind: Never}
		}
		if wrapperExpected != nil {
			numericExpected := *wrapperExpected
			if numericExpected.Wrapped() {
				numericExpected = *numericExpected.Elem
			}
			if result.Kind == Integer && numericExpected.Kind == Decimal {
				c.model.NumericCoercions[expr] = Type{Kind: Decimal}
				result = Type{Kind: Decimal}
			}
		}
		if wrapperExpected != nil && wrapperExpected.Wrapped() && result.Kind != Never && !result.Equal(*wrapperExpected) && result.Kind != Void && (!result.Wrapped() || result.Equal(*wrapperExpected.Elem)) && c.model.Assignable(result, *wrapperExpected.Elem) {
			c.model.RawTypes[expr] = result
			c.model.Wraps[expr] = *wrapperExpected
			result = *wrapperExpected
		}
		if wrapperExpected != nil && wrapperExpected.Kind == Interface && result.Kind != Never && c.model.Assignable(result, *wrapperExpected) && !result.Equal(*wrapperExpected) {
			if _, exists := c.model.RawTypes[expr]; !exists {
				c.model.RawTypes[expr] = result
			}
			result = *wrapperExpected
		}
		c.model.ExprTypes[expr] = result
	}
	return result, err
}

func (c *checker) checkIndex(expr *ast.IndexExpr) (Type, error) {
	objectType, err := c.checkExpr(expr.Object)
	if err != nil {
		return Type{}, err
	}
	if objectType.Kind == Map {
		key, err := c.checkExprExpected(expr.Index, objectType.Key)
		if err != nil {
			return Type{}, err
		}
		if key.Kind != Never && !key.Equal(*objectType.Key) {
			return Type{}, c.fail(expr.Index.Position(), "la clave debe ser %s, no %s", objectType.Key.String(), key.String())
		}
		if key.Kind == Never {
			return key, nil
		}
		return Type{Kind: Optional, Elem: objectType.Elem}, nil
	}
	if objectType.Kind != Slice {
		return Type{}, c.fail(expr.Object.Position(), "solo se pueden indexar listas o mapas, no %s", objectType.String())
	}
	indexType, err := c.checkExpr(expr.Index)
	if err != nil {
		return Type{}, err
	}
	if indexType.Kind != Integer {
		return Type{}, c.fail(expr.Index.Position(), "el índice de una lista debe ser entero, no %s", indexType.String())
	}
	return *objectType.Elem, nil
}

func (c *checker) checkStructLiteral(expr *ast.StructLiteralExpr, expected *Type) (Type, error) {
	if c.invalid[expr.TypeName] {
		return Type{}, errInvalid
	}
	var result Type
	if expr.TypeName != "" {
		if _, exists := c.model.Types[expr.TypeName]; !exists {
			return Type{}, c.fail(expr.Pos, "el tipo %q no existe", expr.TypeName)
		}
		ref := ast.TypeRef{Pos: expr.Pos, Name: expr.TypeName}
		if expr.Type != nil {
			ref = *expr.Type
		}
		var err error
		result, err = c.resolveType(ref)
		if err != nil {
			return Type{}, err
		}
	} else if expected != nil && expected.Kind == Named {
		result = *expected
	} else {
		return Type{}, c.fail(expr.Pos, "no se puede inferir el tipo del literal de estructura")
	}
	if expected != nil && !c.model.Assignable(result, *expected) {
		return Type{}, c.fail(expr.Pos, "el literal es %s, pero se esperaba %s", result.String(), expected.String())
	}
	if stdlib.IsType(result.Name) && stdlib.Fields[result.Name] == "" {
		return Type{}, c.fail(expr.Pos, "%s es un tipo opaco; use su constructor incorporado", result.Name)
	}
	info := c.model.StructInfo(result)
	broken := false
	seen := map[string]bool{}
	if len(expr.Values) > len(info.Decl.Fields) {
		extra := expr.Values[len(info.Decl.Fields)]
		return Type{}, c.fail(extra.Position(), "el literal de %s tiene más valores posicionales que campos directos", result.String())
	}
	for index, value := range expr.Values {
		err := func() error {
			fieldDecl := info.Decl.Fields[index]
			if !ast.Accessible(fieldDecl.Public, fieldDecl.Pos, value.Position()) {
				return c.fail(value.Position(), "el campo %q es privado", fieldDecl.Name)
			}
			field := info.Fields[fieldDecl.Name]
			seen[fieldDecl.Name] = true
			actual, err := c.checkExprExpected(value, &field.Type)
			if err != nil {
				return err
			}
			if !c.model.Assignable(actual, field.Type) {
				return c.fail(value.Position(), "el campo %q debe ser %s, no %s", fieldDecl.Name, field.Type.String(), actual.String())
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}
	}
	for _, value := range expr.Fields {
		err := func() error {
			if seen[value.Name] {
				return c.fail(value.Pos, "el campo %q está duplicado", value.Name)
			}
			seen[value.Name] = true
			field, exists := info.Fields[value.Name]
			if !exists {
				return c.fail(value.Pos, "el campo %q no existe en %s%s", value.Name, result.Name, diagnostic.Hint(value.Name, sortedKeys(info.Fields)))
			}
			if !ast.Accessible(field.Decl.Public, field.Decl.Pos, value.Pos) {
				return c.fail(value.Pos, "el campo %q es privado", value.Name)
			}
			actual, err := c.checkExprExpected(value.Value, &field.Type)
			if err != nil {
				return err
			}
			if !c.model.Assignable(actual, field.Type) {
				return c.fail(value.Value.Position(), "el campo %q debe ser %s, no %s", value.Name, field.Type.String(), actual.String())
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}
	}
	for _, field := range info.Decl.Fields {
		err := func() error {
			if !seen[field.Name] && field.Default == nil {
				if c.model.Types[result.Name].Fields[field.Name].Type.Kind == TypeParameter {
					return c.fail(expr.Pos, "el campo %s requiere inicialización explícita", field.Name)
				}
				if missing := c.missingDefault(info.Fields[field.Name].Type, field.Name); missing != "" {
					return c.fail(expr.Pos, "el campo %s requiere inicialización explícita", missing)
				}
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			broken = true
		}
	}
	if broken {
		return Type{}, errInvalid
	}
	return result, nil
}

func (c *checker) checkListLiteral(expr *ast.ListLiteralExpr, expected *Type) (Type, error) {
	broken := false
	var element *Type
	if expected != nil && expected.Kind != Interface {
		if expected.Kind != Slice {
			return Type{}, c.fail(expr.Pos, "se esperaba %s, no una lista", expected.String())
		}
		element = expected.Elem
	}
	if len(expr.Elements) == 0 {
		if element == nil {
			return Type{}, c.fail(expr.Pos, "no se puede inferir el tipo de una lista vacía")
		}
		return Type{Kind: Slice, Elem: element}, nil
	}
	for index, value := range expr.Elements {
		actual, err := c.checkExprExpected(value, element)
		if err != nil {
			c.report(err)
			broken = true
			continue
		}
		if element == nil {
			if actual.Kind == Void {
				return Type{}, c.fail(value.Position(), "un elemento de lista requiere un valor")
			}
			inferred := actual
			element = &inferred
		} else if !c.model.Assignable(actual, *element) {
			if expected == nil && actual.Kind == Decimal && element.Kind == Integer {
				*element = Type{Kind: Decimal}
				for _, prior := range expr.Elements[:index] {
					c.promoteNumeric(prior)
				}
				continue
			}
			c.report(c.fail(value.Position(), "el elemento debe ser %s, no %s", element.String(), actual.String()))
			broken = true
		}
	}
	if broken {
		return Type{}, errInvalid
	}
	return Type{Kind: Slice, Elem: element}, nil
}

func (c *checker) checkUnary(expr *ast.UnaryExpr) (Type, error) {
	if lit, ok := expr.Value.(*ast.LiteralExpr); ok && expr.Operator == "-" && lit.Kind == "entero" && lit.Value == "9223372036854775808" {
		c.model.ExprTypes[lit] = Type{Kind: Integer}
		return Type{Kind: Integer}, nil
	}
	value, err := c.checkExpr(expr.Value)
	if err != nil {
		return Type{}, err
	}
	if expr.Operator == "-" && value.Numeric() {
		return value, nil
	}
	if expr.Operator == "-" && value.Name == stdlib.Symbol("Vec2") {
		return value, nil
	}
	if expr.Operator == "!" && value.Kind == Boolean {
		return Type{Kind: Boolean}, nil
	}
	return Type{}, c.fail(expr.Pos, "el operador %q no acepta %s", expr.Operator, value.String())
}

func (c *checker) checkBinary(expr *ast.BinaryExpr) (Type, error) {
	if expr.Operator == "==" || expr.Operator == "!=" {
		if t, handled, err := c.checkContextualEnumEquality(expr); handled {
			return t, err
		}
	}
	left, err := c.checkExpr(expr.Left)
	right, rightErr := c.checkExpr(expr.Right)
	if err != nil || rightErr != nil {
		return Type{}, errInvalid
	}
	if left.Numeric() && right.Numeric() {
		if expr.Operator == "%" {
			if left.Kind == Integer && right.Kind == Integer {
				return Type{Kind: Integer}, nil
			}
			return Type{}, c.fail(expr.Pos, "el operador %% requiere enteros")
		}
		common := Type{Kind: Integer}
		if left.Kind == Decimal || right.Kind == Decimal {
			common.Kind = Decimal
			c.promoteNumeric(expr.Left)
			c.promoteNumeric(expr.Right)
		}
		switch expr.Operator {
		case "+", "-", "*", "/":
			return common, nil
		case "<", "<=", ">", ">=", "==", "!=":
			return Type{Kind: Boolean}, nil
		}
	}
	if left.Name == stdlib.Symbol("Vec2") && right.Kind == Integer && (expr.Operator == "*" || expr.Operator == "/") {
		c.promoteNumeric(expr.Right)
		right.Kind = Decimal
	}
	if right.Name == stdlib.Symbol("Vec2") && left.Kind == Integer && expr.Operator == "*" {
		c.promoteNumeric(expr.Left)
		left.Kind = Decimal
	}
	if (expr.Operator == "+" || expr.Operator == "-") && left.Name == stdlib.Symbol("Vec2") && right.Name == stdlib.Symbol("Vec2") || (expr.Operator == "*" || expr.Operator == "/") && left.Name == stdlib.Symbol("Vec2") && right.Kind == Decimal || expr.Operator == "*" && left.Kind == Decimal && right.Name == stdlib.Symbol("Vec2") {
		return Type{Kind: Named, Name: stdlib.Symbol("Vec2")}, nil
	}
	switch expr.Operator {
	case "+", "-", "*", "/":
		if left.Kind == Decimal && right.Kind == Decimal {
			return Type{Kind: Decimal}, nil
		}
		if expr.Operator == "+" && left.Kind == String && right.Kind == String {
			return Type{Kind: String}, nil
		}
		if expr.Operator == "+" && left.Kind == Slice && left.Equal(right) {
			return left, nil
		}
		if expr.Operator == "+" && left.Kind == Slice && right.Equal(*left.Elem) {
			return Type{}, c.fail(expr.Pos, "el operador \"+\" no acepta %s y %s; + une dos listas del mismo tipo, para añadir un elemento usa agregar", left.String(), right.String())
		}
	case "<", "<=", ">", ">=":
		if left.Kind == Decimal && right.Kind == Decimal {
			return Type{Kind: Boolean}, nil
		}
	case "==", "!=":
		if left.Kind == Enum && left.Equal(right) {
			for _, pair := range [][2]ast.Expr{{expr.Right, expr.Left}, {expr.Left, expr.Right}} {
				if constructor, ok := c.model.Constructors[pair[0]]; ok && constructor.Variant.Payload.Kind == Void {
					c.model.EnumCompares[expr] = EnumCompare{Value: pair[1], Tag: constructor.Variant.Tag}
					return Type{Kind: Boolean}, nil
				}
			}
			return Type{}, c.fail(expr.Pos, "los enums solo se comparan con una variante sin payload (estado == .Corriendo); para lo demás usa casos")
		}
		if left.Equal(right) && left.Kind != Void && left.Kind != Never && left.Kind != Slice && left.Kind != Map && left.Kind != Enum && left.Kind != Interface && left.Kind != TypeParameter && !left.Wrapped() {
			return Type{Kind: Boolean}, nil
		}
	case "&&", "||":
		if left.Kind == Boolean && right.Kind == Boolean {
			return Type{Kind: Boolean}, nil
		}
	}
	return Type{}, c.fail(expr.Pos, "el operador %q no acepta %s y %s", expr.Operator, left.String(), right.String())
}

func (c *checker) checkCall(call *ast.CallExpr) (Type, error) {
	if instance, ok := call.Callee.(*ast.InstantiateExpr); ok && c.invalid[instance.Name] {
		return Type{}, errInvalid
	}
	if id, ok := call.Callee.(*ast.IdentExpr); ok && c.invalid[id.Name] {
		return Type{}, errInvalid
	}
	if id, ok := call.Callee.(*ast.IdentExpr); ok && (id.Name == "entero" || id.Name == "decimal") {
		if err := c.plainArguments(call); err != nil {
			return Type{}, err
		}
		if len(call.Args) != 1 {
			return Type{}, c.fail(call.Pos, "%s requiere un argumento numérico", id.Name)
		}
		arg, err := c.checkExpr(call.Args[0])
		if err != nil {
			return Type{}, err
		}
		if !arg.Numeric() {
			return Type{}, c.fail(call.Pos, "%s requiere un argumento numérico", id.Name)
		}
		t := Type{Kind: Integer}
		if id.Name == "decimal" {
			t.Kind = Decimal
		}
		c.model.NumericCalls[call] = t
		return t, nil
	}
	if t, ok, err := c.gameCall(call); ok {
		return t, err
	}
	if member, ok := call.Callee.(*ast.MemberExpr); ok {
		if info, variant, found, err := c.enumMember(member); found {
			if err := c.plainArguments(call); err != nil {
				return Type{}, err
			}
			if err != nil {
				return Type{}, err
			}
			if variant.Payload.Kind == Void {
				return Type{}, c.fail(call.Pos, "la variante %s.%s no acepta paréntesis ni payload", info.Decl.Name, member.Name)
			}
			if len(call.Args) != 1 {
				return Type{}, c.fail(call.Pos, "la variante %s.%s espera exactamente un payload", info.Decl.Name, member.Name)
			}
			actual, err := c.checkExprExpected(call.Args[0], &variant.Payload)
			if err != nil {
				return Type{}, err
			}
			if !c.model.Assignable(actual, variant.Payload) {
				return Type{}, c.fail(call.Args[0].Position(), "el payload debe ser %s, no %s", variant.Payload.String(), actual.String())
			}
			c.model.Constructors[call] = ConstructorInfo{Enum: info, Variant: variant}
			return info.Type, nil
		}
	}
	var signature FuncInfo
	switch callee := call.Callee.(type) {
	case *ast.InstantiateExpr:
		base, exists := c.model.Functions[callee.Name]
		if !exists {
			return Type{}, c.fail(callee.Pos, "la función %q no existe%s", callee.Name, c.nameHint(callee.Name))
		}
		var err error
		signature, err = c.instantiateFunction(base, callee.Args, callee.Pos)
		if err != nil {
			return Type{}, err
		}
	case *ast.IdentExpr:
		if callee.Name == "imprimir" {
			for _, meta := range call.ArgInfo {
				if meta.Spread || meta.Name != "" && meta.Name != "valor" {
					return Type{}, c.fail(meta.Pos, "imprimir acepta un argumento 'valor' sin expansión")
				}
			}
			if len(call.Args) != 1 {
				return Type{}, c.fail(call.Pos, "imprimir espera exactamente un argumento")
			}
			argument, err := c.checkExpr(call.Args[0])
			if err != nil {
				return Type{}, err
			}
			if argument.Kind == Void {
				return Type{}, c.fail(call.Args[0].Position(), "imprimir requiere un valor")
			}
			return Type{Kind: Void}, nil
		}
		var exists bool
		signature, exists = c.model.Functions[callee.Name]
		if !exists {
			return Type{}, c.fail(callee.Pos, "la función %q no existe%s", callee.Name, c.nameHint(callee.Name))
		}
	case *ast.ReceiverExpr:
		if callee.Name == "" {
			return Type{}, c.fail(callee.Pos, "'@' no se puede llamar; use un método explícito")
		}
		if c.receiver == nil {
			return Type{}, c.fail(callee.Pos, "'@%s' solo puede usarse dentro de un método", callee.Name)
		}
		member, err := c.member(c.receiverType(), callee.Name, callee.Pos)
		if err != nil {
			return Type{}, err
		}
		signature = member.Method
		if signature.Decl == nil {
			return Type{}, c.fail(callee.Pos, "el método %q no existe en %s%s", callee.Name, c.receiver.Decl.Name, diagnostic.Hint(callee.Name, sortedKeys(c.model.Members(c.receiverType()))))
		}
	case *ast.MemberExpr:
		objectType, err := c.checkExpr(callee.Object)
		if err != nil {
			return Type{}, err
		}
		if objectType.Kind == Slice {
			return c.checkListCall(call, callee, objectType)
		}
		if objectType.Kind == Map {
			return c.checkMapCall(call, callee, objectType)
		}
		if objectType.Kind == String {
			return c.checkStringCall(call, callee)
		}
		if objectType.Numeric() {
			return c.checkNumberCall(call, callee)
		}
		if objectType.Kind != Named && objectType.Kind != Interface && objectType.Kind != TypeParameter {
			return Type{}, c.fail(callee.Pos, "%s no tiene métodos", objectType.String())
		}
		// Every user-declared tipo has a built-in shallow copiar(), unless the
		// type (or something it embeds) declares its own.
		if objectType.Kind == Named && callee.Name == "copiar" && c.model.Types[objectType.Name] != nil && stdlib.GoType(objectType.Name) == "" {
			if _, defined := c.model.Methods(objectType)["copiar"]; !defined {
				if len(call.Args) != 0 {
					return Type{}, c.fail(call.Pos, "copiar no acepta argumentos")
				}
				c.model.StructCopies[call] = objectType
				return objectType, nil
			}
		}
		var exists bool
		if objectType.Kind == Named {
			if _, err := c.member(objectType, callee.Name, callee.Pos); err != nil {
				return Type{}, err
			}
		}
		signature, exists = c.model.Methods(objectType)[callee.Name]
		if !exists {
			return Type{}, c.fail(callee.Pos, "el método %q no existe en %s%s", callee.Name, objectType.Name, diagnostic.Hint(callee.Name, sortedKeys(c.model.Members(objectType))))
		}
		// Catalog methods receive their defaults at the call site, like library functions.
		if NativeMethod(signature.Decl) {
			for i, p := range signature.Decl.Params {
				if p.Default != nil {
					if _, err := c.checkExprExpected(p.Default, &signature.Params[i]); err != nil {
						return Type{}, err
					}
				}
			}
		}
	default:
		return Type{}, c.fail(call.Pos, "solo se pueden llamar funciones o métodos del receptor")
	}
	if len(signature.TypeParams) > 0 && len(signature.TypeArgs) == 0 {
		var err error
		signature, err = c.inferFunction(call, signature)
		if err != nil {
			return Type{}, err
		}
	}
	return c.bindArguments(call, signature)
}

func (c *checker) checkIfExpr(expr *ast.IfExpr, expected *Type) (Type, error) {
	var result Type
	broken := false
	defer c.bindingScope()()
	for index, branch := range expr.Branches {
		outer := c.vars
		c.vars = cloneVars(outer)
		if err := c.checkCondition(branch.Condition, branch.Binding, branch.BindingPos); err != nil {
			c.report(err)
			broken = true
			if branch.Binding != "" {
				c.vars[branch.Binding] = Type{Kind: Invalid}
			}
		}
		branchExpected := expected
		if branchExpected == nil && index > 0 && result.Kind != Invalid {
			branchExpected = &result
		}
		branchType, err := c.checkExprExpected(branch.Value, branchExpected)
		c.vars = outer
		if err != nil {
			c.report(err)
			broken = true
			continue
		}
		if result.Kind == Invalid || result.Kind == Never {
			result = branchType
		} else if branchType.Kind != Never && !result.Equal(branchType) {
			if result.Kind == Integer && branchType.Kind == Decimal {
				for _, prior := range expr.Branches[:index] {
					c.promoteNumeric(prior.Value)
				}
				result = branchType
				continue
			}
			return Type{}, c.fail(branch.Value.Position(), "las ramas producen %s y %s", result.String(), branchType.String())
		}
	}
	elseExpected := expected
	if elseExpected == nil && result.Kind != Invalid {
		elseExpected = &result
	}
	elseType, err := c.checkExprExpected(expr.Else, elseExpected)
	if err != nil {
		return Type{}, err
	}
	if broken {
		return Type{}, errInvalid
	}
	if result.Kind != Never && elseType.Kind != Never && !result.Equal(elseType) {
		if result.Kind == Integer && elseType.Kind == Decimal {
			for _, prior := range expr.Branches {
				c.promoteNumeric(prior.Value)
			}
			return elseType, nil
		}
		return Type{}, c.fail(expr.Else.Position(), "las ramas producen %s y %s", result.String(), elseType.String())
	}
	if result.Kind == Void {
		return Type{}, c.fail(expr.Pos, "una expresión condicional debe producir un valor")
	}
	if result.Kind == Never {
		result = elseType
	}
	return result, nil
}

func (c *checker) fail(pos ast.Pos, format string, args ...any) error {
	filename := c.filename
	if pos.Filename != "" {
		filename = pos.Filename
	}
	return &Error{Filename: filename, Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// sortedKeys lists map keys so suggestions do not depend on iteration order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// nameHint explains an unknown name: a missing '@', a keyword borrowed from
// another language, or a close spelling of something in scope.
func (c *checker) nameHint(name string) string {
	if c.receiver != nil {
		if _, ok := c.model.Members(c.receiverType())[name]; ok {
			return "; para usar un campo o método del objeto escribe '@" + name + "'"
		}
	}
	if foreign := diagnostic.Foreign(name); foreign != "" {
		return "; " + foreign
	}
	candidates := sortedKeys(c.vars)
	candidates = append(candidates, sortedKeys(c.model.Globals)...)
	candidates = append(candidates, sortedKeys(c.model.Functions)...)
	return diagnostic.Hint(name, candidates)
}
