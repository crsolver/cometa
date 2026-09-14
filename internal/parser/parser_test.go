package parser

import (
	"testing"

	"hacha/internal/ast"
	"hacha/internal/lexer"
)

func TestParsesRangeLoop(t *testing.T) {
	tokens, err := lexer.Lex("rango.hacha", "fn inicio()\n\trepetir ((1 + 2)..-5) |i| imprimir(i)\n")
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("rango.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	loop := program.Decls[0].(*ast.FuncDecl).Body[0].(*ast.RepeatStmt)
	if _, ok := loop.Iterable.(*ast.BinaryExpr); !ok {
		t.Fatalf("start = %T, want binary expression", loop.Iterable)
	}
	if _, ok := loop.RangeEnd.(*ast.UnaryExpr); !ok {
		t.Fatalf("end = %T, want unary expression", loop.RangeEnd)
	}
	if loop.Element != "i" || len(loop.Body) != 1 {
		t.Fatalf("unexpected loop: %#v", loop)
	}
}

func TestParsesTypeMethodsAndConditionalExpression(t *testing.T) {
	source := "tipo Usuario\n\tedad num\n\tactivo bool\n\tfn activar(valor bool)\n\t\t@activo = si (@edad < 18) verdadero sino valor\n"
	tokens, err := lexer.Lex("usuario.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("usuario.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Decls) != 1 {
		t.Fatalf("got %d declarations", len(program.Decls))
	}
	typeDecl := program.Decls[0].(*ast.TypeDecl)
	if len(typeDecl.Fields) != 2 || len(typeDecl.Methods) != 1 {
		t.Fatalf("unexpected type shape: %#v", typeDecl)
	}
	assignment := typeDecl.Methods[0].Body[0].(*ast.AssignStmt)
	if _, ok := assignment.Value.(*ast.IfExpr); !ok {
		t.Fatalf("assignment value is %T, want *ast.IfExpr", assignment.Value)
	}
}

func TestParsesNestedReturningConditionals(t *testing.T) {
	source := "fn elegir(n num) num\n\tsi (n < 0) 0\n\tosi (n == 0) 1\n\tsino\n\t\tsi verdadero\n\t\t\t2\n\t\tsino\n\t\t\t3\n"
	tokens, err := lexer.Lex("elegir.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("elegir.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	function := program.Decls[0].(*ast.FuncDecl)
	outer := function.Body[0].(*ast.IfStmt)
	if len(outer.Branches) != 2 || len(outer.Else) != 1 {
		t.Fatalf("unexpected outer conditional: %#v", outer)
	}
	if _, ok := outer.Else[0].(*ast.IfStmt); !ok {
		t.Fatalf("else contains %T, want nested if", outer.Else[0])
	}
}

func TestParsesVariablesCompositeLiteralsAndMemberCalls(t *testing.T) {
	source := "tipo Usuario\n\tnombre cadena\n\tamigos [Usuario]\n\tfn activar(valor bool)\n\t\timprimir(valor)\nfn inicio()\n\tvar usuario1 = Usuario {\n\t\tnombre: \"andres\",\n\t}\n\tvar usuario2 Usuario = {nombre: \"andres\", amigos: [usuario1]}\n\tusuario2.activar(verdadero)\n"
	tokens, err := lexer.Lex("variables.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("variables.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	function := program.Decls[1].(*ast.FuncDecl)
	first := function.Body[0].(*ast.VarDeclStmt)
	if first.Type != nil {
		t.Fatalf("first variable has explicit type: %#v", first.Type)
	}
	if literal, ok := first.Value.(*ast.StructLiteralExpr); !ok || literal.TypeName != "Usuario" {
		t.Fatalf("first value = %#v", first.Value)
	}
	second := function.Body[1].(*ast.VarDeclStmt)
	if second.Type == nil || second.Type.Name != "Usuario" {
		t.Fatalf("second variable type = %#v", second.Type)
	}
	call := function.Body[2].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	if member, ok := call.Callee.(*ast.MemberExpr); !ok || member.Name != "activar" {
		t.Fatalf("call callee = %#v", call.Callee)
	}
}

func TestParsesListAndInfiniteRepeats(t *testing.T) {
	source := "fn inicio()\n\tvar lista = [1, 2]\n\trepetir (lista) |valor, indice|\n\t\tsi (indice == 0) continuar\n\t\tromper\n\trepetir imprimir(\"hola\")\n"
	tokens, err := lexer.Lex("ciclos.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("ciclos.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	function := program.Decls[0].(*ast.FuncDecl)
	loop := function.Body[1].(*ast.RepeatStmt)
	if loop.Element != "valor" || loop.Index != "indice" || len(loop.Body) != 2 {
		t.Fatalf("unexpected list repeat: %#v", loop)
	}
	conditional := loop.Body[0].(*ast.IfStmt)
	if _, ok := conditional.Branches[0].Body[0].(*ast.ContinueStmt); !ok {
		t.Fatalf("conditional body contains %T, want *ast.ContinueStmt", conditional.Branches[0].Body[0])
	}
	if _, ok := loop.Body[1].(*ast.BreakStmt); !ok {
		t.Fatalf("loop body contains %T, want *ast.BreakStmt", loop.Body[1])
	}
	infinite := function.Body[2].(*ast.RepeatStmt)
	if infinite.Iterable != nil || len(infinite.Body) != 1 {
		t.Fatalf("unexpected infinite repeat: %#v", infinite)
	}
}

func TestParsesListIndexAsFunctionArgument(t *testing.T) {
	source := "fn procesar_usuario(usuario num)\n\timprimir(usuario)\nfn inicio()\n\tvar lista = [1, 2]\n\tvar x = lista[1]\n\tprocesar_usuario(lista[0])\n"
	tokens, err := lexer.Lex("indices.hacha", source)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse("indices.hacha", tokens)
	if err != nil {
		t.Fatal(err)
	}
	function := program.Decls[1].(*ast.FuncDecl)
	variable := function.Body[1].(*ast.VarDeclStmt)
	if _, ok := variable.Value.(*ast.IndexExpr); !ok {
		t.Fatalf("variable value is %T, want *ast.IndexExpr", variable.Value)
	}
	call := function.Body[2].(*ast.ExprStmt).Expr.(*ast.CallExpr)
	index, ok := call.Args[0].(*ast.IndexExpr)
	if !ok {
		t.Fatalf("call argument is %T, want *ast.IndexExpr", call.Args[0])
	}
	if identifier, ok := index.Object.(*ast.IdentExpr); !ok || identifier.Name != "lista" {
		t.Fatalf("indexed object = %#v", index.Object)
	}
}
