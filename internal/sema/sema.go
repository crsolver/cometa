package sema

import (
	"errors"
	"fmt"
	"sort"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
)

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

// enterDeclaration starts checking the body or initializer of a declaration
// with empty local state: no variables, no loop, no result type. scope gives
// the type parameters in view. It returns the function that puts back the
// state of the declaration that was being checked, because a global is
// checked the first time another declaration reads it.
func (c *checker) enterDeclaration(scope ast.Decl, receiver *TypeInfo) (leave func()) {
	vars, bindings, reads, typeScope := c.vars, c.bindings, c.reads, c.typeScope
	outerReceiver, returnType, loopDepth, inDefault := c.receiver, c.returnType, c.loopDepth, c.inDefault
	c.setScope(scope)
	c.vars = map[string]Type{}
	c.bindings = map[string]*ast.VarDeclStmt{}
	c.reads = map[*ast.VarDeclStmt]bool{}
	c.receiver = receiver
	c.returnType = Type{Kind: Void}
	c.loopDepth = 0
	c.inDefault = false
	return func() {
		c.vars, c.bindings, c.reads, c.typeScope = vars, bindings, reads, typeScope
		c.receiver, c.returnType, c.loopDepth, c.inDefault = outerReceiver, returnType, loopDepth, inDefault
	}
}

func (c *checker) fail(pos ast.Pos, format string, args ...any) error {
	filename := c.filename
	if pos.Filename != "" {
		filename = pos.Filename
	}
	return &diagnostic.Error{Stage: diagnostic.Sema, Filename: filename, Pos: pos, Message: fmt.Sprintf(format, args...)}
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
