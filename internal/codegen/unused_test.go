package codegen

import (
	"strings"
	"testing"
)

func TestUnusedLocals(t *testing.T) {
	for _, tt := range []struct{ name, body, want, absent string }{
		{"transitive", `a := int64(1); b := a; _ = b`, "func main()", "a :="},
		{"effects", `a := effect(); b := a; _ = b`, "_ = effect()", "a :="},
		{"write only", `a := 1; a = effect()`, "_ = effect()", "a :="},
		{"shadow", `a := 1; { a := 2; _ = a }; println(a)`, "println(a)", "a := 2"},
		{"index panic", `a := []int{}; b := a[0]; _ = b`, "_ = a[0]", "b :="},
		{"range", `for i, v := range []int{1} { _ = i; _ = v; effect() }`, "for range []int{1}", "i,"},
		{"type switch", `var a any = 1; switch v := a.(type) { case int: _ = v; effect(); default: _ = v }`, "switch a.(type)", "v :="},
		{"assertion", `var a any = 1; v, ok := a.(int); _ = v; println(ok)`, "_, ok := a.(int)", "v,"},
		{"global writes", `global = 2`, "global = 2", "_ = 2"},
		{"mixed short declaration", `a := 0; a, b := effect(), effect(); println(a); _ = b`, "a, _ = effect(), effect()", "a, _ :="},
		{"compound assignment", `a := 0; a += effect(); println(a)`, "a += effect()", "_ = effect()"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := "package main\nvar global int\nfunc effect() int { println(1); return 1 }\nfunc main() {" + tt.body + "}\n"
			got, err := eliminateUnusedLocals([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			if err := validate("unused", got); err != nil {
				t.Fatalf("%v\n%s", err, got)
			}
			if !strings.Contains(string(got), tt.want) || strings.Contains(string(got), tt.absent) {
				t.Fatalf("unexpected output:\n%s", got)
			}
		})
	}
}
