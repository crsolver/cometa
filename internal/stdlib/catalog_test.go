package stdlib

import "testing"

func TestCatalogDeclarations(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range Functions {
		d := f.Declaration()
		if !d.Public || d.Name != FunctionSymbol(f.Namespace, f.Name) || seen[f.GoName] {
			t.Fatalf("invalid or duplicate intrinsic: %+v", f)
		}
		seen[f.GoName] = true
	}
	for n := range Fields {
		d := TypeDeclaration(n)
		if !d.Public || d.Name != n {
			t.Fatal(n)
		}
		for _, field := range d.Fields {
			if !field.Public {
				t.Fatalf("private native field: %s.%s", n, field.Name)
			}
		}
	}
}
