package stdlib

import "testing"

func TestCatalogDeclarations(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range Functions {
		d := f.Declaration()
		if d.Name != FunctionSymbol(f.Namespace, f.Name) || seen[f.GoName] {
			t.Fatalf("invalid or duplicate intrinsic: %+v", f)
		}
		seen[f.GoName] = true
	}
	for n := range Fields {
		if TypeDeclaration(n).Name != n {
			t.Fatal(n)
		}
	}
}
