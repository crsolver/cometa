package stdlib

import "testing"

// TestDocCoverage keeps every module, function, method and type documented:
// it fails as soon as a catalog entry is added without its Doc, so the
// reference and the editor's hover cannot silently fall behind the API.
func TestDocCoverage(t *testing.T) {
	for _, m := range Modules {
		if m.Doc == "" {
			t.Errorf("module %s has no Doc", m.Name)
		}
		for _, f := range m.Functions {
			if f.Doc == "" {
				t.Errorf("%s.%s has no Doc", m.Name, f.Name)
			}
		}
		for _, typ := range m.Types {
			if typ.Doc == "" {
				t.Errorf("%s.%s has no Doc", m.Name, typ.Name)
			}
			if typ.Interface {
				continue // what a program must write is shown by its signature
			}
			for _, method := range typ.Methods {
				if method.Doc == "" {
					t.Errorf("%s.%s.%s has no Doc", m.Name, typ.Name, method.Name)
				}
			}
		}
	}
}
