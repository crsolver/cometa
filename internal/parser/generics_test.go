package parser

import (
	"hacha/internal/ast"
	"hacha/internal/lexer"
	"testing"
)

func TestGenericSyntaxAndComparisonDisambiguation(t *testing.T) {
	source := `interfaz I<T>
	fn obtener() T
tipo Caja<T I<num>>
	valor T
enum E<T>
	Dato T
fn f<T>(v T) T v
fn inicio()
	var a = 1 < 2
	var b = 3 > 2
	var c = f<Caja<Caja<num>>>(x)
	var d = Caja<num> {valor: 1}
	var e = E<num>.Dato(1)
	var z = v como Caja<num> o d
	casos v |p|
		Caja<num> => imprimir(p)
		[num] => imprimir(p)
		_ => imprimir(0)
`
	tokens, err := lexer.Lex("generic.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("generic.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	body := program.Decls[4].(*ast.FuncDecl).Body
	for _, i := range []int{0, 1} {
		if _, ok := body[i].(*ast.VarDeclStmt).Value.(*ast.BinaryExpr); !ok {
			t.Fatalf("comparison %d parsed as generic", i)
		}
	}
	call := body[2].(*ast.VarDeclStmt).Value.(*ast.CallExpr)
	instance, ok := call.Callee.(*ast.InstantiateExpr)
	if !ok || len(instance.Args[0].Args) != 1 {
		t.Fatalf("bad nested type arguments: %#v", call)
	}
	literal := body[3].(*ast.VarDeclStmt).Value.(*ast.StructLiteralExpr)
	if literal.Type == nil || literal.Type.Args[0].Name != "num" {
		t.Fatalf("bad literal type: %#v", literal)
	}
	match := body[6].(*ast.MatchStmt).Match
	if match.Arms[0].TypePattern == nil || match.Arms[1].TypePattern.Element == nil {
		t.Fatal("missing type patterns")
	}
}
