package sema

// Invalidity follows signature dependencies, never function-body failures.
// This fixed point also catches forward references to a signature diagnosed later.
func (c *checker) invalidType(t Type) bool {
	if t.Kind == Invalid || c.invalid[t.Name] {
		return true
	}
	if t.Elem != nil && c.invalidType(*t.Elem) {
		return true
	}
	if t.Key != nil && c.invalidType(*t.Key) {
		return true
	}
	if t.Err != nil && c.invalidType(*t.Err) {
		return true
	}
	for _, a := range t.Args {
		if c.invalidType(a) {
			return true
		}
	}
	return false
}

func (c *checker) invalidSignature(f FuncInfo) bool {
	if c.invalidType(f.Return) {
		return true
	}
	for _, t := range f.Params {
		if c.invalidType(t) {
			return true
		}
	}
	for _, t := range f.Constraints {
		if c.invalidType(t) {
			return true
		}
	}
	return false
}

func (c *checker) propagateInvalidTypes() {
	for {
		before := len(c.invalid)
		for name, info := range c.model.Types {
			for _, f := range info.Fields {
				if c.invalidType(f.Type) {
					c.invalid[name] = true
				}
			}
			for _, f := range info.Methods {
				if c.invalidSignature(f) {
					c.invalid[name] = true
				}
			}
		}
		for name, info := range c.model.Enums {
			for _, v := range info.Variants {
				if c.invalidType(v.Payload) {
					c.invalid[name] = true
				}
			}
		}
		for name, info := range c.model.Interfaces {
			for _, e := range info.Embeds {
				if c.invalidType(e) {
					c.invalid[name] = true
				}
			}
			for _, f := range info.Methods {
				if c.invalidSignature(f) {
					c.invalid[name] = true
				}
			}
		}
		for name, f := range c.model.Functions {
			if c.invalidSignature(f) {
				c.invalid[name] = true
			}
		}
		if len(c.invalid) == before {
			return
		}
	}
}
