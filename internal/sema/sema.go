package sema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"hacha/internal/ast"
)

type Kind int

const (
	Invalid Kind = iota
	Void
	Number
	String
	Boolean
	Named
	Slice
	Enum
	Optional
	Result
	Never
	Interface
	TypeParameter
)

type Type struct {
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
	case Number:
		return "num"
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
	if t.Kind == Slice || t.Wrapped() {
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

type Model struct {
	Interfaces    map[string]*InterfaceInfo
	TypeParams    map[ast.Decl][]Type
	Constraints   map[string]Type
	RawTypes      map[ast.Expr]Type
	TypeRefs      map[*ast.TypeRef]Type
	Wraps         map[ast.Expr]Type
	Calls         map[*ast.CallExpr]CallInfo
	LocalNames    map[string]bool
	Enums         map[string]*EnumInfo
	Constructors  map[ast.Expr]ConstructorInfo
	Types         map[string]*TypeInfo
	Functions     map[string]FuncInfo
	ExprTypes     map[ast.Expr]Type
	ExpectedTypes map[ast.Expr]Type
	PatternTypes  map[*ast.MatchArm]Type
	VarTypes      map[*ast.VarDeclStmt]Type
}

type Error struct {
	Filename string
	Pos      ast.Pos
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Filename, e.Pos.Line, e.Pos.Column, e.Message)
}

type checker struct {
	typeScope  map[string]Type
	pending    []typeUse
	expansions []typeEdge
	returnType Type
	inDefault  bool
	bindings   map[string]*ast.VarDeclStmt
	reads      map[*ast.VarDeclStmt]bool
	tooling    bool
	filename   string
	model      *Model
	vars       map[string]Type
	receiver   *TypeInfo
	loopDepth  int
}

func Check(filename string, program *ast.Program) (*Model, error) {
	return newChecker(filename).check(program)
}

// CheckForTooling retains successfully resolved information before an error.
// It is only for editor queries on incomplete documents, never compilation.
func CheckForTooling(filename string, program *ast.Program) (*Model, error) {
	c := newChecker(filename)
	c.tooling = true
	_, err := c.check(program)
	return c.model, err
}

func newChecker(filename string) *checker {
	return &checker{filename: filename, model: &Model{
		Interfaces: map[string]*InterfaceInfo{}, TypeParams: map[ast.Decl][]Type{}, Constraints: map[string]Type{}, RawTypes: map[ast.Expr]Type{}, TypeRefs: map[*ast.TypeRef]Type{},
		Wraps:         map[ast.Expr]Type{},
		Calls:         map[*ast.CallExpr]CallInfo{},
		LocalNames:    map[string]bool{},
		ExpectedTypes: map[ast.Expr]Type{},
		PatternTypes:  map[*ast.MatchArm]Type{},
		Enums:         map[string]*EnumInfo{}, Constructors: map[ast.Expr]ConstructorInfo{},
		Types: map[string]*TypeInfo{}, Functions: map[string]FuncInfo{}, ExprTypes: map[ast.Expr]Type{}, VarTypes: map[*ast.VarDeclStmt]Type{},
	}}

}

func (c *checker) signature(function *ast.FuncDecl) (FuncInfo, error) {
	info := FuncInfo{Decl: function, Return: Type{Kind: Void}}
	seen := map[string]bool{}
	optional := false
	for index, param := range function.Params {
		if param.Default != nil {
			if param.Variadic {
				return FuncInfo{}, c.fail(param.Pos, "un parámetro variádico no acepta un valor predeterminado")
			}
			optional = true
		} else if optional && !param.Variadic {
			return FuncInfo{}, c.fail(param.Pos, "un parámetro obligatorio no puede seguir a uno con valor predeterminado")
		}
		if param.Variadic && index != len(function.Params)-1 {
			return FuncInfo{}, c.fail(param.Pos, "el parámetro variádico debe ser el último")
		}
		if seen[param.Name] {
			return FuncInfo{}, c.fail(param.Pos, "el parámetro %q está duplicado", param.Name)
		}
		seen[param.Name] = true
		paramType, err := c.resolveType(param.Type)
		if err != nil {
			return FuncInfo{}, err
		}
		if param.Variadic {
			element := paramType
			paramType = Type{Kind: Slice, Elem: &element}
		}
		info.Params = append(info.Params, paramType)
	}
	if function.ReturnType != nil {
		result, err := c.resolveType(*function.ReturnType)
		if err != nil {
			return FuncInfo{}, err
		}
		info.Return = result
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
		return Type{Kind: Number}, nil
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
				return err
			}
			if !c.model.Assignable(actual, expected) {
				return c.fail(param.Default.Position(), "el valor predeterminado de %q debe ser %s, no %s", param.Name, expected.String(), actual.String())
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
	if len(unread) > 0 {
		return c.fail(unread[0].Pos, "el resultado %q debe usarse o manejarse explícitamente", unread[0].Name)
	}
	return nil
}

func (c *checker) mustResolve(function *ast.FuncDecl, index int) Type {
	t, _ := c.resolveType(function.Params[index].Type)
	if function.Params[index].Variadic {
		return Type{Kind: Slice, Elem: &t}
	}
	return t
}

func (c *checker) checkStmt(stmt ast.Stmt) error {
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
		value, err := c.checkExprExpected(statement.Value, &target)
		if err != nil {
			return err
		}
		if !c.model.Assignable(value, target) {
			return c.fail(statement.Pos, "no se puede asignar %s a %s", value.String(), target.String())
		}
		return nil
	case *ast.VarDeclStmt:
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
			if err == nil {
				err = c.checkNestedBlock(branch.Body)
			}
			c.vars = outer
			if err != nil {
				return err
			}
		}
		if err := c.checkNestedBlock(statement.Else); err != nil {
			return err
		}
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
			return err
		}
		elementType := Type{Kind: Number}
		if stmt.RangeEnd != nil {
			if iterable.Kind != Number {
				return c.fail(stmt.Iterable.Position(), "el inicio del rango debe ser num, no %s", iterable.String())
			}
			end, err := c.checkExpr(stmt.RangeEnd)
			if err != nil {
				return err
			}
			if end.Kind != Number {
				return c.fail(stmt.RangeEnd.Position(), "el final del rango debe ser num, no %s", end.String())
			}
		} else if iterable.Kind != Slice {
			return c.fail(stmt.Iterable.Position(), "repetir requiere una lista, no %s", iterable.String())
		} else {
			elementType = *iterable.Elem
		}
		if _, exists := c.vars[stmt.Element]; exists {
			return c.fail(stmt.Pos, "la variable %q ya fue declarada", stmt.Element)
		}
		c.vars[stmt.Element] = elementType
		if stmt.Index != "" {
			if stmt.Index == stmt.Element {
				return c.fail(stmt.Pos, "las variables de elemento e índice deben tener nombres distintos")
			}
			if _, exists := c.vars[stmt.Index]; exists {
				return c.fail(stmt.Pos, "la variable %q ya fue declarada", stmt.Index)
			}
			c.vars[stmt.Index] = Type{Kind: Number}
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
	switch target := expr.(type) {
	case *ast.IndexExpr:
		return c.checkExpr(target)
	case *ast.IdentExpr:
		valueType, exists := c.vars[target.Name]
		if !exists {
			return Type{}, c.fail(target.Pos, "el nombre %q no existe; los campos requieren '@'", target.Name)
		}
		c.model.ExprTypes[target] = valueType
		return valueType, nil
	case *ast.ReceiverExpr:
		if c.receiver == nil {
			return Type{}, c.fail(target.Pos, "'@%s' solo puede usarse dentro de un método", target.Name)
		}
		member, err := c.member(c.receiverType(), target.Name, target.Pos)
		if err != nil {
			return Type{}, err
		}
		field := member.Field
		if field.Decl == nil {
			return Type{}, c.fail(target.Pos, "el campo %q no existe en %s", target.Name, c.receiver.Decl.Name)
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
		member, err := c.member(objectType, target.Name, target.Pos)
		if err != nil {
			return Type{}, err
		}
		field := member.Field
		if field.Decl == nil {
			return Type{}, c.fail(target.Pos, "el campo %q no existe en %s", target.Name, objectType.Name)
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
		case *ast.StructLiteralExpr, *ast.ListLiteralExpr:
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
		case "num":
			result = Type{Kind: Number}
		case "cadena":
			if _, unquoteErr := strconv.Unquote(expression.Value); unquoteErr != nil {
				err = c.fail(expression.Pos, "cadena inválida: %v", unquoteErr)
			} else {
				result = Type{Kind: String}
			}
		case "bool":
			result = Type{Kind: Boolean}
		}
	case *ast.IdentExpr:
		if binding := c.bindings[expression.Name]; binding != nil {
			if _, tracked := c.reads[binding]; tracked {
				c.reads[binding] = true
			}
		}
		var exists bool
		result, exists = c.vars[expression.Name]
		if !exists {
			err = c.fail(expression.Pos, "el nombre %q no existe; los campos requieren '@'", expression.Name)
		}
	case *ast.ReceiverExpr:
		if c.receiver == nil {
			err = c.fail(expression.Pos, "'@%s' solo puede usarse dentro de un método", expression.Name)
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
				err = c.fail(expression.Pos, "el miembro %q no existe en %s", expression.Name, c.receiver.Decl.Name)
			}
		} else {
			result = field.Type
		}
	case *ast.MemberExpr:
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
				err = c.fail(expression.Pos, "el miembro %q no existe en %s", expression.Name, objectType.String())
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
			err = c.fail(expression.Pos, "el miembro %q no existe en %s", expression.Name, objectType.Name)
		}
	case *ast.IndexExpr:
		result, err = c.checkIndex(expression)
	case *ast.StructLiteralExpr:
		result, err = c.checkStructLiteral(expression, expected)
	case *ast.ListLiteralExpr:
		result, err = c.checkListLiteral(expression, expected)
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
	if objectType.Kind != Slice {
		return Type{}, c.fail(expr.Object.Position(), "solo se pueden indexar listas, no %s", objectType.String())
	}
	indexType, err := c.checkExpr(expr.Index)
	if err != nil {
		return Type{}, err
	}
	if indexType.Kind != Number {
		return Type{}, c.fail(expr.Index.Position(), "el índice de una lista debe ser num, no %s", indexType.String())
	}
	return *objectType.Elem, nil
}

func (c *checker) checkStructLiteral(expr *ast.StructLiteralExpr, expected *Type) (Type, error) {
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
	info := c.model.StructInfo(result)
	seen := map[string]bool{}
	for _, value := range expr.Fields {
		if seen[value.Name] {
			return Type{}, c.fail(value.Pos, "el campo %q está duplicado", value.Name)
		}
		seen[value.Name] = true
		field, exists := info.Fields[value.Name]
		if !exists {
			return Type{}, c.fail(value.Pos, "el campo %q no existe en %s", value.Name, result.Name)
		}
		actual, err := c.checkExprExpected(value.Value, &field.Type)
		if err != nil {
			return Type{}, err
		}
		if !c.model.Assignable(actual, field.Type) {
			return Type{}, c.fail(value.Value.Position(), "el campo %q debe ser %s, no %s", value.Name, field.Type.String(), actual.String())
		}
	}
	for _, field := range info.Decl.Fields {
		if !seen[field.Name] {
			if c.model.Types[result.Name].Fields[field.Name].Type.Kind == TypeParameter {
				return Type{}, c.fail(expr.Pos, "el campo %s requiere inicialización explícita", field.Name)
			}
			if missing := c.missingDefault(info.Fields[field.Name].Type, field.Name); missing != "" {
				return Type{}, c.fail(expr.Pos, "el campo %s requiere inicialización explícita", missing)
			}
		}
	}
	return result, nil
}

func (c *checker) checkListLiteral(expr *ast.ListLiteralExpr, expected *Type) (Type, error) {
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
			return Type{}, err
		}
		if index == 0 && element == nil {
			if actual.Kind == Void {
				return Type{}, c.fail(value.Position(), "un elemento de lista requiere un valor")
			}
			inferred := actual
			element = &inferred
		} else if !c.model.Assignable(actual, *element) {
			return Type{}, c.fail(value.Position(), "el elemento debe ser %s, no %s", element.String(), actual.String())
		}
	}
	return Type{Kind: Slice, Elem: element}, nil
}

func (c *checker) checkUnary(expr *ast.UnaryExpr) (Type, error) {
	value, err := c.checkExpr(expr.Value)
	if err != nil {
		return Type{}, err
	}
	if expr.Operator == "-" && value.Kind == Number {
		return Type{Kind: Number}, nil
	}
	if expr.Operator == "!" && value.Kind == Boolean {
		return Type{Kind: Boolean}, nil
	}
	return Type{}, c.fail(expr.Pos, "el operador %q no acepta %s", expr.Operator, value.String())
}

func (c *checker) checkBinary(expr *ast.BinaryExpr) (Type, error) {
	left, err := c.checkExpr(expr.Left)
	if err != nil {
		return Type{}, err
	}
	right, err := c.checkExpr(expr.Right)
	if err != nil {
		return Type{}, err
	}
	switch expr.Operator {
	case "+", "-", "*", "/":
		if left.Kind == Number && right.Kind == Number {
			return Type{Kind: Number}, nil
		}
	case "<", "<=", ">", ">=":
		if left.Kind == Number && right.Kind == Number {
			return Type{Kind: Boolean}, nil
		}
	case "==", "!=":
		if left.Equal(right) && left.Kind != Void && left.Kind != Never && left.Kind != Slice && left.Kind != Enum && left.Kind != Interface && left.Kind != TypeParameter && !left.Wrapped() {
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
			return Type{}, c.fail(callee.Pos, "la función %q no existe", callee.Name)
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
			return Type{}, c.fail(callee.Pos, "la función %q no existe", callee.Name)
		}
	case *ast.ReceiverExpr:
		if c.receiver == nil {
			return Type{}, c.fail(callee.Pos, "'@%s' solo puede usarse dentro de un método", callee.Name)
		}
		member, err := c.member(c.receiverType(), callee.Name, callee.Pos)
		if err != nil {
			return Type{}, err
		}
		signature = member.Method
		if signature.Decl == nil {
			return Type{}, c.fail(callee.Pos, "el método %q no existe en %s", callee.Name, c.receiver.Decl.Name)
		}
	case *ast.MemberExpr:
		objectType, err := c.checkExpr(callee.Object)
		if err != nil {
			return Type{}, err
		}
		if objectType.Kind != Named && objectType.Kind != Interface && objectType.Kind != TypeParameter {
			return Type{}, c.fail(callee.Pos, "%s no tiene métodos", objectType.String())
		}
		var exists bool
		if objectType.Kind == Named {
			if _, err := c.member(objectType, callee.Name, callee.Pos); err != nil {
				return Type{}, err
			}
		}
		signature, exists = c.model.Methods(objectType)[callee.Name]
		if !exists {
			return Type{}, c.fail(callee.Pos, "el método %q no existe en %s", callee.Name, objectType.Name)
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
	for index, branch := range expr.Branches {
		outer := c.vars
		c.vars = cloneVars(outer)
		if err := c.checkCondition(branch.Condition, branch.Binding, branch.BindingPos); err != nil {
			c.vars = outer
			return Type{}, err
		}
		branchExpected := expected
		if branchExpected == nil && index > 0 {
			branchExpected = &result
		}
		branchType, err := c.checkExprExpected(branch.Value, branchExpected)
		c.vars = outer
		if err != nil {
			return Type{}, err
		}
		if index == 0 || result.Kind == Never {
			result = branchType
		} else if branchType.Kind != Never && !result.Equal(branchType) {
			return Type{}, c.fail(branch.Value.Position(), "las ramas producen %s y %s", result.String(), branchType.String())
		}
	}
	elseExpected := expected
	if elseExpected == nil {
		elseExpected = &result
	}
	elseType, err := c.checkExprExpected(expr.Else, elseExpected)
	if err != nil {
		return Type{}, err
	}
	if result.Kind != Never && elseType.Kind != Never && !result.Equal(elseType) {
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
