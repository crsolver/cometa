package sema

import (
	"strconv"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/stdlib"
)

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
			if t.Kind == Void {
				err = c.fail(part.Expr.Position(), "la interpolación requiere un valor, pero esta expresión no produce ninguno")
				break
			}
			if t.Kind != String && !t.Numeric() && t.Kind != Boolean && t.Kind != Never {
				// Lists, maps, structures, enums and wrappers are written as
				// imprimir writes them, which needs its formatting runtime.
				c.model.FormatsValues = true
			}
		}
		if err == nil {
			result = Type{Kind: String}
		}
	case *ast.IdentExpr:
		if constant, ok := stdlib.LookupConstant(expression.Name); ok {
			result, err = c.resolveType(ast.TypeRef{Pos: expression.Pos, Name: constant.Type})
			c.model.Stdlib.Constants[expression] = constant.GoValue
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
		// retro.Icono.Arriba: una constante de un tipo de la biblioteca escrita con
		// su tipo, equivalente a .Arriba con ese tipo esperado.
		if id, ok := expression.Object.(*ast.IdentExpr); ok && stdlib.Constants[id.Name] != nil {
			owner, typeErr := c.resolveType(ast.TypeRef{Pos: id.Pos, Name: id.Name})
			if typeErr != nil {
				err = typeErr
				break
			}
			c.model.ExprTypes[id] = owner
			value, exists := stdlib.Constants[id.Name][expression.Name]
			if !exists {
				err = c.fail(expression.NamePos, "la constante %q no existe en %s%s", expression.Name, owner.String(), diagnostic.Hint(expression.Name, sortedKeys(stdlib.Constants[id.Name])))
				break
			}
			result = owner
			c.model.Stdlib.Constants[expression] = value
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
		if native, ok := stdlib.LookupType(objectType.Name); ok && native.Channel(expression.Name) != "" {
			result = Type{Kind: Integer}
			c.model.Stdlib.Channels[expression] = native.Channel(expression.Name)
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
	if expr.Operator == "-" && isVector(value) {
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
	// A vector adds to and subtracts from another of its type, and a number
	// scales it: v * n, n * v and v / n.
	scales := expr.Operator == "*" || expr.Operator == "/"
	if isVector(left) && right.Kind == Integer && scales {
		c.promoteNumeric(expr.Right)
		right.Kind = Decimal
	}
	if isVector(right) && left.Kind == Integer && expr.Operator == "*" {
		c.promoteNumeric(expr.Left)
		left.Kind = Decimal
	}
	switch {
	case (expr.Operator == "+" || expr.Operator == "-") && isVector(left) && right.Kind == Named && right.Name == left.Name:
		return left, nil
	case scales && isVector(left) && right.Kind == Decimal:
		return left, nil
	case expr.Operator == "*" && left.Kind == Decimal && isVector(right):
		return right, nil
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
		c.model.Builtins[call] = Builtin{Kind: Conversion, Name: id.Name, Type: t}
		return t, nil
	}
	if t, ok, err := c.stdlibCall(call); ok {
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
				c.model.Builtins[call] = Builtin{Kind: StructCopy, Name: callee.Name, Type: objectType}
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

// isVector reports whether t is a library type with arithmetic, such as mate.Vec2.
func isVector(t Type) bool { return t.Kind == Named && stdlib.Vector(t.Name) != nil }
