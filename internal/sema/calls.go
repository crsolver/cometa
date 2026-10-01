package sema

import "github.com/crsolver/cometa/internal/ast"

// CallInfo maps source arguments to parameters without mutating the syntax tree.
type CallInfo struct {
	Signature  FuncInfo
	Parameters []int
	Spread     []bool
	Named      bool
}

func (c *checker) plainArguments(call *ast.CallExpr) error {
	for _, meta := range call.ArgInfo {
		if meta.Name != "" || meta.Spread {
			return c.fail(meta.Pos, "el payload no acepta argumentos nombrados ni expansión")
		}
	}
	return nil
}

func (c *checker) bindArguments(call *ast.CallExpr, signature FuncInfo) (Type, error) {
	broken := false
	info := CallInfo{Signature: signature}
	n := len(signature.Params)
	variadic := n > 0 && signature.Decl.Params[n-1].Variadic
	fixed := n
	if variadic {
		fixed--
	}
	seen := make([]bool, n)
	positional := 0
	variadicSpread := false
	for i, arg := range call.Args {
		meta := ast.ArgumentInfo{Pos: arg.Position()}
		if i < len(call.ArgInfo) {
			meta = call.ArgInfo[i]
		}
		index := positional
		if meta.Name != "" {
			info.Named = true
			index = -1
			for j, param := range signature.Decl.Params {
				if param.Name == meta.Name {
					index = j
					break
				}
			}
			if index < 0 {
				return Type{}, c.fail(meta.Pos, "el parámetro %q no existe", meta.Name)
			}
		} else {
			if info.Named {
				return Type{}, c.fail(meta.Pos, "un argumento posicional no puede seguir a uno nombrado")
			}
			if variadic && index >= fixed {
				index = fixed
			}
			positional++
		}
		if index >= n {
			return Type{}, c.fail(meta.Pos, "se esperaban %d argumentos, se recibieron %d", n, len(call.Args))
		}
		isVariadic := variadic && index == fixed
		spread := meta.Spread || isVariadic && meta.Name != ""
		if seen[index] && (!isVariadic || meta.Name != "" || spread || variadicSpread) {
			return Type{}, c.fail(meta.Pos, "el parámetro %q está duplicado o mezcla expansión con elementos", signature.Decl.Params[index].Name)
		}
		if meta.Spread && (!isVariadic || i != len(call.Args)-1) {
			return Type{}, c.fail(meta.Pos, "la expansión requiere el último argumento de un parámetro variádico")
		}
		seen[index] = true
		expected := signature.Params[index]
		if isVariadic {
			variadicSpread = spread
			if !spread {
				expected = *expected.Elem
			}
		}
		actual, err := c.checkExprExpected(arg, &expected)
		if err != nil {
			c.report(err)
			broken = true
			continue
		}
		if !c.model.Assignable(actual, expected) {
			c.report(c.fail(arg.Position(), "el argumento %d debe ser %s, no %s", i+1, expected.String(), actual.String()))
			broken = true
			continue
		}
		info.Parameters = append(info.Parameters, index)
		info.Spread = append(info.Spread, spread)
	}
	for i := 0; i < fixed; i++ {
		if !seen[i] && signature.Decl.Params[i].Default == nil {
			return Type{}, c.fail(call.Pos, "se esperaban %d argumentos; falta el parámetro %q", fixed, signature.Decl.Params[i].Name)
		}
	}
	c.model.Calls[call] = info
	if broken {
		return Type{}, errInvalid
	}
	return signature.Return, nil
}
