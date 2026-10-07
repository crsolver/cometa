package sema

import (
	"fmt"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/stdlib"
)

// check runs every pass over a program. A declaration that fails a pass is
// reported, marked invalid and skipped by the later ones, so one mistake does
// not hide the rest of the file. The passes run in dependency order:
// identities, type parameters and their constraints, members and signatures,
// whole-program validations, and finally initializers and bodies.
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

	typeNames := map[string]string{}
	c.eachValid(program, func(d ast.Decl) error { return c.collectDeclaration(d, typeNames) })
	c.eachValid(program, c.declareTypeParams)
	c.eachValid(program, c.resolveConstraints)
	c.eachValid(program, c.resolveMembers)

	c.propagateInvalidTypes()
	if err := c.flattenInterfaces(); err != nil {
		c.report(err)
	}
	if err := c.validateTypeUses(); err != nil {
		c.report(err)
	}
	c.rejectExpandingGenerics()
	if err := c.checkRequiredCycles(program); err != nil {
		c.report(err)
	}
	c.propagateInvalidTypes()

	c.eachValid(program, func(d ast.Decl) error {
		if global, ok := d.(*ast.GlobalDecl); ok {
			return c.checkGlobal(global.Name)
		}
		return nil
	})
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
	c.rejectExpandingGenerics()
	if err := c.safeCheck(func() error { return c.checkGame(program) }); err != nil {
		c.report(err)
	}
	return c.model, nil
}

// eachValid runs pass over the declarations that are still valid. A failure
// is reported and invalidates its declaration.
func (c *checker) eachValid(program *ast.Program, pass func(ast.Decl) error) {
	for _, d := range program.Decls {
		if c.invalid[declName(d)] {
			continue
		}
		if err := pass(d); err != nil {
			c.report(err)
			c.invalid[declName(d)] = true
		}
	}
}

// collectDeclaration records the identity of a type, enum, interface or
// global before any signature is resolved, so forward references work.
func (c *checker) collectDeclaration(d ast.Decl, typeNames map[string]string) error {
	name := declName(d)
	if stdlib.Reserved(name) {
		return c.fail(d.Position(), "el nombre %q está reservado por el runtime", name)
	}
	if c.model.declaration(name) != nil {
		return c.fail(d.Position(), "el tipo %q ya fue declarado", name)
	}
	if _, global := d.(*ast.GlobalDecl); !global {
		if _, function := d.(*ast.FuncDecl); !function {
			if earlier, clash := claimGoName(typeNames, name); clash {
				return c.fail(d.Position(), "los tipos %q y %q solo se distinguen por la mayúscula inicial; cambia uno de los dos nombres", earlier, name)
			}
		}
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
}

// declareTypeParams gives each type parameter its identity, with the
// universal constraint until resolveConstraints reads the declared one.
func (c *checker) declareTypeParams(d ast.Decl) error {
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
}

func (c *checker) resolveConstraints(d ast.Decl) error {
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
}

// resolveMembers resolves what a declaration exposes: the fields and method
// signatures of a type, the variants of an enum, the requirements of an
// interface, the signature of a function. Every member is tried, so several
// mistakes in one declaration are all reported.
func (c *checker) resolveMembers(d ast.Decl) error {
	c.setScope(d)
	broken := false
	try := func(member func() error) {
		if err := member(); err != nil {
			c.report(err)
			broken = true
		}
	}
	switch d := d.(type) {
	case *ast.TypeDecl:
		info := c.model.Types[d.Name]
		memberNames := map[string]string{}
		for _, f := range d.Fields {
			try(func() error {
				if _, exists := info.Fields[f.Name]; exists {
					return c.fail(f.Pos, "el campo %q ya fue declarado", f.Name)
				}
				if earlier, clash := claimGoName(memberNames, f.Name); clash {
					return c.fail(f.Pos, "los miembros %q y %q solo se distinguen por la mayúscula inicial; cambia uno de los dos nombres", earlier, f.Name)
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
			})
		}
		for _, method := range d.Methods {
			try(func() error {
				if _, exists := info.Methods[method.Name]; exists {
					return c.fail(method.Pos, "el método %q ya fue declarado", method.Name)
				}
				if _, exists := info.Fields[method.Name]; exists {
					return c.fail(method.Pos, "el miembro %q ya fue declarado como campo", method.Name)
				}
				if earlier, clash := claimGoName(memberNames, method.Name); clash {
					return c.fail(method.Pos, "los miembros %q y %q solo se distinguen por la mayúscula inicial; cambia uno de los dos nombres", earlier, method.Name)
				}
				sig, err := c.signature(method)
				if err != nil {
					return err
				}
				info.Methods[method.Name] = sig
				return nil
			})
		}
	case *ast.EnumDecl:
		info := c.model.Enums[d.Name]
		info.Type.Args = c.model.TypeParams[d]
		for i, v := range d.Variants {
			try(func() error {
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
			})
		}
	case *ast.InterfaceDecl:
		info := c.model.Interfaces[d.Name]
		methodNames := map[string]string{}
		for _, method := range d.Methods {
			try(func() error {
				if _, exists := info.Methods[method.Name]; exists {
					return c.fail(method.Pos, "el método %q ya fue declarado", method.Name)
				}
				if earlier, clash := claimGoName(methodNames, method.Name); clash {
					return c.fail(method.Pos, "los métodos %q y %q solo se distinguen por la mayúscula inicial; cambia uno de los dos nombres", earlier, method.Name)
				}
				sig, err := c.signature(method)
				if err != nil {
					return err
				}
				info.Methods[method.Name] = sig
				return nil
			})
		}
		for _, ref := range d.Embeds {
			try(func() error {
				t, err := c.resolveType(ref)
				if err != nil {
					return err
				}
				if t.Kind != Interface {
					return c.fail(ref.Pos, "solo se puede incrustar una interfaz")
				}
				info.Embeds = append(info.Embeds, t)
				return nil
			})
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
}

// rejectExpandingGenerics reports generic declarations whose instantiation
// would never end. Such a failure makes every generic declaration unsafe to
// instantiate, so all of them are invalidated.
func (c *checker) rejectExpandingGenerics() {
	if err := c.checkExpansions(); err != nil {
		c.report(err)
		for d, params := range c.model.TypeParams {
			if len(params) > 0 {
				c.invalid[declName(d)] = true
			}
		}
	}
}
