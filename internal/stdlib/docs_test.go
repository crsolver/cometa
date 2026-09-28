package stdlib

import "testing"

// TestDocCoverage keeps every stdlib function, method and type documented:
// it fails as soon as a new catalog entry is added without a matching
// FunctionDocs/MethodDocs/TypeDocs entry, so documentation can't silently
// fall behind the API surface.
func TestDocCoverage(t *testing.T) {
	for _, f := range Functions {
		key := f.Namespace + "." + f.Name
		if FunctionDocs[key] == "" {
			t.Errorf("missing FunctionDocs[%q]", key)
		}
	}
	for _, m := range Methods {
		key := PublicName(m.Namespace) + "." + m.Name
		if MethodDocs[key] == "" {
			t.Errorf("missing MethodDocs[%q]", key)
		}
	}
	for name := range TypeModules {
		if TypeDocs[name] == "" {
			t.Errorf("missing TypeDocs[%q]", name)
		}
	}
}
