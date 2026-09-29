package sema

import (
	"cometa/internal/ast"
	"cometa/internal/stdlib"
	"fmt"
	"sort"
)

type InterfaceInfo struct {
	Decl    *ast.InterfaceDecl
	Methods map[string]FuncInfo
	Embeds  []Type
}

type typeUse struct {
	Type Type
	Pos  ast.Pos
}

func paramKey(t Type) string { return t.Owner + ":" + t.Name }

func declParams(d ast.Decl) []ast.TypeParam {
	switch d := d.(type) {
	case *ast.TypeDecl:
		return d.TypeParams
	case *ast.EnumDecl:
		return d.TypeParams
	case *ast.InterfaceDecl:
		return d.TypeParams
	case *ast.FuncDecl:
		return d.TypeParams
	}
	return nil
}

func declName(d ast.Decl) string {
	switch d := d.(type) {
	case *ast.TypeDecl:
		return d.Name
	case *ast.EnumDecl:
		return d.Name
	case *ast.InterfaceDecl:
		return d.Name
	case *ast.FuncDecl:
		return d.Name
	case *ast.GlobalDecl:
		return d.Name
	}
	return ""
}

func (c *checker) setScope(d ast.Decl) {
	c.typeScope = map[string]Type{}
	for _, p := range c.model.TypeParams[d] {
		c.typeScope[p.Name] = p
	}
}

func (m *Model) declaration(name string) ast.Decl {
	if d := m.Types[name]; d != nil {
		return d.Decl
	}
	if d := m.Enums[name]; d != nil {
		return d.Decl
	}
	if d := m.Interfaces[name]; d != nil {
		return d.Decl
	}
	return nil
}

func (c *checker) check(program *ast.Program) (*Model, error) {
	c.globalState = map[string]int{}
	for name := range program.InvalidNames {
		c.invalid[name] = true
	}
	if err := c.installGameAPI(); err != nil {
		return nil, err
	}
	c.model.Game.Modules = program.NativeModules
	c.model.Game.Used = len(program.NativeModules) > 0
	// Collect identities before resolving any signatures, including forward references.
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		err := func() error {
			name := declName(d)
			if stdlib.Reserved(name) {
				return c.fail(d.Position(), "el nombre %q está reservado por el runtime", name)
			}
			if c.model.declaration(name) != nil {
				return c.fail(d.Position(), "el tipo %q ya fue declarado", name)
			}
			switch d := d.(type) {
			case *ast.TypeDecl:
				c.model.Types[name] = &TypeInfo{Decl: d, Fields: map[string]FieldInfo{}, Methods: map[string]FuncInfo{}}
			case *ast.EnumDecl:
				c.model.Enums[name] = &EnumInfo{Decl: d, Type: Type{Kind: Enum, Name: name}, Variants: map[string]VariantInfo{}}
			case *ast.InterfaceDecl:
				c.model.Interfaces[name] = &InterfaceInfo{Decl: d, Methods: map[string]FuncInfo{}}
			case *ast.GlobalDecl:
				if c.model.Globals[name] != nil {
					return c.fail(d.Pos, "la global %q ya fue declarada", name)
				}
				c.model.Globals[name] = &GlobalInfo{Decl: d, Constant: d.Constant}
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			c.invalid[declName(d)] = true
		}
	}
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		err := func() error {
			seen := map[string]bool{}
			for _, p := range declParams(d) {
				if stdlib.Reserved(p.Name) || seen[p.Name] || p.Name == "_" || c.model.declaration(p.Name) != nil || p.Name == declName(d) {
					return c.fail(p.Pos, "parámetro de tipo duplicado o en conflicto %q", p.Name)
				}
				seen[p.Name] = true
				t := Type{Kind: TypeParameter, Name: p.Name, Owner: fmt.Sprintf("%s@%d", declName(d), d.Position().Line)}
				c.model.TypeParams[d] = append(c.model.TypeParams[d], t)
				c.model.Constraints[paramKey(t)] = Type{Kind: Interface} // universal constraint
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			c.invalid[declName(d)] = true
		}
	}
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		err := func() error {
			c.setScope(d)
			for i, p := range declParams(d) {
				if p.Constraint == nil {
					continue
				}
				t, err := c.resolveType(*p.Constraint)
				if err != nil {
					return err
				}
				if t.Kind != Interface {
					return c.fail(p.Pos, "la restricción debe ser una interfaz")
				}
				c.model.Constraints[paramKey(c.model.TypeParams[d][i])] = t
			}

			return nil
		}()
		if err != nil {
			c.report(err)
			c.invalid[declName(d)] = true
		}
	}
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		err := func() error {
			broken := false
			c.setScope(d)
			switch d := d.(type) {
			case *ast.TypeDecl:
				info := c.model.Types[d.Name]
				for _, f := range d.Fields {
					err := func() error {
						if _, exists := info.Fields[f.Name]; exists {
							return c.fail(f.Pos, "el campo %q ya fue declarado", f.Name)
						}
						t, err := c.resolveType(f.Type)
						if err != nil {
							return err
						}
						if f.Embedded && (t.Kind != Named || stdlib.IsType(t.Name)) {
							return c.fail(f.Pos, "solo se pueden embeber tipos de estructura declarados")
						}
						info.Fields[f.Name] = FieldInfo{Decl: f, Type: t}

						return nil
					}()
					if err != nil {
						c.report(err)
						broken = true
					}
				}
				for _, method := range d.Methods {
					err := func() error {
						if _, exists := info.Methods[method.Name]; exists {
							return c.fail(method.Pos, "el método %q ya fue declarado", method.Name)
						}
						if _, exists := info.Fields[method.Name]; exists {
							return c.fail(method.Pos, "el miembro %q ya fue declarado como campo", method.Name)
						}
						sig, err := c.signature(method)
						if err != nil {
							return err
						}
						info.Methods[method.Name] = sig

						return nil
					}()
					if err != nil {
						c.report(err)
						broken = true
					}
				}
			case *ast.EnumDecl:
				info := c.model.Enums[d.Name]
				info.Type.Args = c.model.TypeParams[d]
				for i, v := range d.Variants {
					err := func() error {
						if _, exists := info.Variants[v.Name]; exists || v.Name == "_" {
							return c.fail(v.Pos, "variante duplicada o reservada %q", v.Name)
						}
						payload := Type{Kind: Void}
						if v.Payload != nil {
							var err error
							payload, err = c.resolveType(*v.Payload)
							if err != nil {
								return err
							}
						}
						info.Variants[v.Name] = VariantInfo{Decl: v, Tag: i + 1, Payload: payload}

						return nil
					}()
					if err != nil {
						c.report(err)
						broken = true
					}
				}
			case *ast.InterfaceDecl:
				info := c.model.Interfaces[d.Name]
				for _, method := range d.Methods {
					err := func() error {
						if _, exists := info.Methods[method.Name]; exists {
							return c.fail(method.Pos, "el método %q ya fue declarado", method.Name)
						}
						sig, err := c.signature(method)
						if err != nil {
							return err
						}
						info.Methods[method.Name] = sig

						return nil
					}()
					if err != nil {
						c.report(err)
						broken = true
					}
				}
				for _, ref := range d.Embeds {
					err := func() error {
						t, err := c.resolveType(ref)
						if err != nil {
							return err
						}
						if t.Kind != Interface {
							return c.fail(ref.Pos, "solo se puede incrustar una interfaz")
						}
						info.Embeds = append(info.Embeds, t)

						return nil
					}()
					if err != nil {
						c.report(err)
						broken = true
					}
				}
			case *ast.FuncDecl:
				if c.model.declaration(d.Name) != nil {
					return c.fail(d.Pos, "el nombre %q ya fue declarado como tipo", d.Name)
				}
				if _, exists := c.model.Functions[d.Name]; exists {
					return c.fail(d.Pos, "la función %q ya fue declarada", d.Name)
				}
				if c.model.Globals[d.Name] != nil {
					return c.fail(d.Pos, "el nombre %q ya fue declarado como global", d.Name)
				}
				sig, err := c.signature(d)
				if err != nil {
					return err
				}
				sig.TypeParams = c.model.TypeParams[d]
				for _, p := range sig.TypeParams {
					sig.Constraints = append(sig.Constraints, c.model.Constraints[paramKey(p)])
				}
				if d.Name == "inicio" && (len(d.Params) > 0 || sig.Return.Kind != Void || len(sig.TypeParams) > 0) {
					return c.fail(d.Pos, "inicio debe declararse como fn inicio() sin parámetros ni resultado")
				}
				c.model.Functions[d.Name] = sig
			case *ast.GlobalDecl:
				if c.model.declaration(d.Name) != nil {
					return c.fail(d.Pos, "el nombre %q ya fue declarado como tipo", d.Name)
				}
				if _, exists := c.model.Functions[d.Name]; exists {
					return c.fail(d.Pos, "el nombre %q ya fue declarado como función", d.Name)
				}
			}

			if broken {
				return errInvalid
			}
			return nil
		}()
		if err != nil {
			c.report(err)
			c.invalid[declName(d)] = true
		}
	}
	c.propagateInvalidTypes()
	if err := c.flattenInterfaces(); err != nil {
		c.report(err)
	}
	if err := c.validateTypeUses(); err != nil {
		c.report(err)
	}
	if err := c.checkExpansions(); err != nil {
		c.report(err)
		// Expansion failures make recursive generic instantiation unsafe.
		for d, params := range c.model.TypeParams {
			if len(params) > 0 {
				c.invalid[declName(d)] = true
			}
		}
	}
	if err := c.checkRequiredCycles(program); err != nil {
		c.report(err)
	}
	c.propagateInvalidTypes()
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		if global, ok := d.(*ast.GlobalDecl); ok {
			if err := c.checkGlobal(global.Name); err != nil {
				c.report(err)
			}
		}
	}
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		switch d := d.(type) {
		case *ast.TypeDecl:
			if err := c.checkFieldDefaults(d); err != nil {
				c.report(err)
			}
			for _, method := range d.Methods {
				if err := c.checkFunction(method, c.model.Types[d.Name]); err != nil {
					c.report(err)
				}
			}
		case *ast.FuncDecl:
			if err := c.checkFunction(d, nil); err != nil {
				c.report(err)
			}
		}
	}
	if err := c.safeCheck(func() error { return c.checkInitializationCycles(program) }); err != nil {
		c.report(err)
	}
	if err := c.validateTypeUses(); err != nil {
		c.report(err)
	}
	if err := c.checkExpansions(); err != nil {
		c.report(err)
		// Expansion failures make recursive generic instantiation unsafe.
		for d, params := range c.model.TypeParams {
			if len(params) > 0 {
				c.invalid[declName(d)] = true
			}
		}
	}
	if err := c.safeCheck(func() error { return c.checkGame(program) }); err != nil {
		c.report(err)
	}
	return c.model, nil
}

func (c *checker) resolveType(ref ast.TypeRef) (Type, error) {
	if c.invalid[ref.Name] {
		return Type{}, errInvalid
	}
	if ref.Wrapper != "" || ref.Element != nil {
		return c.resolveBuiltinType(ref)
	}
	if t, exists := c.typeScope[ref.Name]; exists {
		if len(ref.Args) > 0 {
			return Type{}, c.fail(ref.Pos, "un parámetro de tipo no acepta argumentos")
		}
		return t, nil
	}
	d := c.model.declaration(ref.Name)
	if d == nil {
		if len(ref.Args) > 0 {
			return Type{}, c.fail(ref.Pos, "el tipo %s no acepta argumentos", ref.Name)
		}
		return c.resolveBuiltinType(ref)
	}
	params := c.model.TypeParams[d]
	if len(params) != len(ref.Args) {
		return Type{}, c.fail(ref.Pos, "%s requiere %d argumentos de tipo", ref.Name, len(params))
	}
	t := Type{Kind: Named, Name: ref.Name}
	switch d.(type) {
	case *ast.EnumDecl:
		t.Kind = Enum
	case *ast.InterfaceDecl:
		t.Kind = Interface
	}
	for _, arg := range ref.Args {
		a, err := c.resolveType(arg)
		if err != nil {
			return Type{}, err
		}
		t.Args = append(t.Args, a)
	}
	if len(t.Args) > 0 {
		c.pending = append(c.pending, typeUse{Type: t, Pos: ref.Pos})
		c.recordExpansion(params, t.Args, ref.Pos)
	}
	return t, nil
}

func substitute(t Type, bindings map[string]Type) Type {
	if t.Key != nil {
		key := substitute(*t.Key, bindings)
		t.Key = &key
	}
	if t.Kind == TypeParameter {
		if actual, ok := bindings[paramKey(t)]; ok {
			return actual
		}
		return t
	}
	if t.Elem != nil {
		elem := substitute(*t.Elem, bindings)
		t.Elem = &elem
	}
	if t.Err != nil {
		e := substitute(*t.Err, bindings)
		t.Err = &e
	}
	if len(t.Args) > 0 {
		args := make([]Type, len(t.Args))
		for i, a := range t.Args {
			args[i] = substitute(a, bindings)
		}
		t.Args = args
	}
	return t
}

func bindTypes(params, args []Type) map[string]Type {
	b := map[string]Type{}
	for i, p := range params {
		if i < len(args) {
			b[paramKey(p)] = args[i]
		}
	}
	return b
}

func substituteFunc(f FuncInfo, bindings map[string]Type) FuncInfo {
	p := make([]Type, len(f.Params))
	for i, t := range f.Params {
		p[i] = substitute(t, bindings)
	}
	f.Params = p
	f.Return = substitute(f.Return, bindings)
	return f
}

func (m *Model) StructInfo(t Type) *TypeInfo {
	base := m.Types[t.Name]
	if base == nil || len(t.Args) == 0 {
		return base
	}
	b := bindTypes(m.TypeParams[base.Decl], t.Args)
	info := &TypeInfo{Decl: base.Decl, Fields: map[string]FieldInfo{}, Methods: map[string]FuncInfo{}}
	for name, f := range base.Fields {
		f.Type = substitute(f.Type, b)
		info.Fields[name] = f
	}
	for name, f := range base.Methods {
		info.Methods[name] = substituteFunc(f, b)
	}
	return info
}

func (m *Model) Methods(t Type) map[string]FuncInfo {
	if t.Kind == Map {
		return MapMethods(t)
	}
	if t.Kind == Slice {
		return ListMethods(t)
	}
	if t.Kind == String {
		return StringMethods()
	}
	if t.Kind == TypeParameter {
		t = m.Constraints[paramKey(t)]
	}
	if t.Kind == Named {
		if info := m.StructInfo(t); info != nil {
			methods := map[string]FuncInfo{}
			for name, member := range m.Members(t) {
				if !member.Ambiguous && member.Method.Decl != nil {
					methods[name] = member.Method
				}
			}
			return methods
		}
	}
	if t.Kind == Interface && t.Name != "" {
		base := m.Interfaces[t.Name]
		if base == nil {
			return nil
		}
		b := bindTypes(m.TypeParams[base.Decl], t.Args)
		methods := map[string]FuncInfo{}
		for n, f := range base.Methods {
			methods[n] = substituteFunc(f, b)
		}
		return methods
	}
	return nil
}

func sameSignature(a, b FuncInfo) bool {
	if len(a.Params) != len(b.Params) || !a.Return.Equal(b.Return) {
		return false
	}
	for i, p := range a.Params {
		if !p.Equal(b.Params[i]) || a.Decl.Params[i].Variadic != b.Decl.Params[i].Variadic {
			return false
		}
	}
	return true
}

// Assignable is directional. Containers retain exact type identity.
func (m *Model) Assignable(from, to Type) bool {
	if from.Kind == Integer && to.Kind == Decimal {
		return true
	}
	if from.Kind == Never {
		return true
	}
	if from.Equal(to) {
		return true
	}
	if to.Kind != Interface || from.Kind == Void || from.Kind == Invalid {
		return false
	}
	// Map operations are compiler intrinsics, not methods on a Go map value.
	if from.Kind == Map && len(m.Methods(to)) != 0 {
		return false
	}
	have := m.Methods(from)
	if from.Kind == Named {
		for name, member := range m.Members(from) {
			if !member.Public() {
				delete(have, name)
			}
		}
	}
	for name, want := range m.Methods(to) {
		actual, ok := have[name]
		if !ok || !sameSignature(actual, want) {
			return false
		}
	}
	return true
}

func (c *checker) flattenInterfaces() error {
	state := map[string]int{}
	var visit func(string) error
	visit = func(name string) error {
		if c.invalid[name] {
			return errInvalid
		}
		if state[name] == 2 {
			return nil
		}
		if state[name] == 1 {
			return c.fail(c.model.Interfaces[name].Decl.Pos, "ciclo de interfaces incrustadas en %s", name)
		}
		state[name] = 1
		info := c.model.Interfaces[name]
		own := info.Methods
		merged := map[string]FuncInfo{}
		ambiguous := map[string]bool{}
		for _, e := range info.Embeds {
			if err := visit(e.Name); err != nil {
				return err
			}
			for n, f := range c.model.Methods(e) {
				if prev, ok := merged[n]; ok {
					if !sameSignature(prev, f) {
						return c.fail(info.Decl.Pos, "firmas incompatibles para el método %s", n)
					}
					for i, p := range prev.Decl.Params {
						if p.Name != f.Decl.Params[i].Name {
							ambiguous[n] = true
						}
					}
				} else {
					merged[n] = f
				}
			}
		}
		for n, f := range own {
			if prev, ok := merged[n]; ok && !sameSignature(prev, f) {
				return c.fail(f.Decl.Pos, "firmas incompatibles para el método %s", n)
			}
			merged[n] = f
			delete(ambiguous, n)
		}
		var names []string
		for n := range ambiguous {
			names = append(names, n)
		}
		sort.Strings(names)
		if len(names) > 0 {
			return c.fail(info.Decl.Pos, "el método %s requiere redeclaración explícita de los nombres de parámetros", names[0])
		}
		info.Methods = merged
		state[name] = 2
		return nil
	}
	var names []string
	for n := range c.model.Interfaces {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := visit(n); err != nil {
			c.report(err)
			c.invalid[n] = true
		}
	}
	return nil
}

func (c *checker) validateTypeUses() error {
	for _, u := range c.pending {
		if c.invalidType(u.Type) {
			continue
		}
		params := c.model.TypeParams[c.model.declaration(u.Type.Name)]
		if err := c.checkConstraints(params, u.Type.Args, u.Pos); err != nil {
			c.report(err)
		}
	}
	c.pending = nil
	return nil
}

func (c *checker) checkConstraints(params, args []Type, pos ast.Pos) error {
	b := bindTypes(params, args)
	for i, p := range params {
		constraint := substitute(c.model.Constraints[paramKey(p)], b)
		if !c.model.Assignable(args[i], constraint) {
			return c.fail(pos, "%s no satisface la restricción %s de %s", args[i].String(), constraint.String(), p.Name)
		}
	}
	return nil
}

func (c *checker) instantiateFunction(f FuncInfo, refs []ast.TypeRef, pos ast.Pos) (FuncInfo, error) {
	if len(refs) != len(f.TypeParams) || len(refs) == 0 {
		return FuncInfo{}, c.fail(pos, "%s requiere %d argumentos de tipo", f.Decl.Name, len(f.TypeParams))
	}
	var args []Type
	for _, ref := range refs {
		t, err := c.resolveType(ref)
		if err != nil {
			return FuncInfo{}, err
		}
		args = append(args, t)
	}
	return c.specializeFunction(f, args, pos)
}

func (c *checker) specializeFunction(f FuncInfo, args []Type, pos ast.Pos) (FuncInfo, error) {
	c.recordExpansion(f.TypeParams, args, pos)
	if err := c.checkConstraints(f.TypeParams, args, pos); err != nil {
		return FuncInfo{}, err
	}
	f = substituteFunc(f, bindTypes(f.TypeParams, args))
	f.TypeArgs = args
	return f, nil
}

func (c *checker) inferFunction(call *ast.CallExpr, f FuncInfo) (FuncInfo, error) {
	bindings := map[string]Type{}
	wanted := map[string]bool{}
	for _, p := range f.TypeParams {
		wanted[paramKey(p)] = true
	}
	var unify func(Type, Type) bool
	unify = func(pattern, actual Type) bool {
		if pattern.Kind == TypeParameter && wanted[paramKey(pattern)] {
			key := paramKey(pattern)
			if prior, ok := bindings[key]; ok {
				return prior.Equal(actual)
			}
			bindings[key] = actual
			return true
		}
		if pattern.Kind != actual.Kind || pattern.Name != actual.Name || len(pattern.Args) != len(actual.Args) {
			return true
		}
		if pattern.Key != nil && actual.Key != nil && !unify(*pattern.Key, *actual.Key) {
			return false
		}
		if pattern.Elem != nil && actual.Elem != nil && !unify(*pattern.Elem, *actual.Elem) {
			return false
		}
		if pattern.Err != nil && actual.Err != nil && !unify(*pattern.Err, *actual.Err) {
			return false
		}
		for i, p := range pattern.Args {
			if !unify(p, actual.Args[i]) {
				return false
			}
		}
		return true
	}
	for i, arg := range call.Args {
		index := i
		meta := ast.ArgumentInfo{}
		if i < len(call.ArgInfo) {
			meta = call.ArgInfo[i]
		}
		if meta.Name != "" {
			index = -1
			for j, p := range f.Decl.Params {
				if p.Name == meta.Name {
					index = j
					break
				}
			}
		}
		if index >= len(f.Params) && len(f.Params) > 0 && f.Decl.Params[len(f.Params)-1].Variadic {
			index = len(f.Params) - 1
		}
		if index < 0 || index >= len(f.Params) {
			continue
		} // bindArguments reports shape errors.
		pattern := f.Params[index]
		if f.Decl.Params[index].Variadic && !meta.Spread && meta.Name == "" {
			pattern = *pattern.Elem
		}
		if contextualOnly(arg) {
			continue
		}
		actual, err := c.checkExpr(arg)
		if err != nil {
			return FuncInfo{}, err
		}
		if !unify(pattern, actual) {
			return FuncInfo{}, c.fail(arg.Position(), "argumentos de tipo incompatibles al inferir %s", f.Decl.Name)
		}
	}
	var args []Type
	for _, p := range f.TypeParams {
		t, ok := bindings[paramKey(p)]
		if !ok {
			return FuncInfo{}, c.fail(call.Pos, "no se puede inferir %s; indique todos los argumentos de tipo explícitamente", p.Name)
		}
		args = append(args, t)
	}
	return c.specializeFunction(f, args, call.Pos)
}

func contextualOnly(e ast.Expr) bool {
	if isContextualConstructor(e) {
		return true
	}
	switch e := e.(type) {
	case *ast.StructLiteralExpr:
		return e.TypeName == ""
	case *ast.MapLiteralExpr:
		if len(e.Entries) == 0 {
			return true
		}
		for _, entry := range e.Entries {
			if contextualOnly(entry.Key) || contextualOnly(entry.Value) {
				return true
			}
		}
	case *ast.ListLiteralExpr:
		if len(e.Elements) == 0 {
			return true
		}
		for _, v := range e.Elements {
			if contextualOnly(v) {
				return true
			}
		}
	}
	return false
}
