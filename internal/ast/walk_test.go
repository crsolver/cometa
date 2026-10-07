package ast_test

import (
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"reflect"
	"sort"
	"testing"

	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/lexer"
	"github.com/crsolver/cometa/internal/parser"
)

// A program that uses every kind of declaration, statement and expression.
const everyNode = `usar std/mate

pub const limite entero = 3
var puntos [entero: [cadena]] = [:]

interfaz Nombrable<T Otra>
	Otra
	fn nombre(prefijo cadena) cadena

interfaz Otra

enum Evento<T>
	Vacio
	Dato T

tipo Base
	id entero

tipo Caja<T Nombrable<T>>
	Base
	valor T
	etiquetas [cadena] = ["a", "b"]
	destino mate.Vec2?
	estado (entero!cadena)?

	pub fn nombre(prefijo cadena = "x", extras ...entero) cadena!
		si @etiquetas.longitud() > limite
			retornar .Error("demasiadas")
		var total = 0
		repetir (extras) |extra, indice|
			total += extra * indice
		repetir (0..total)
			continuar
		mientras total > 0
			total -= 1
			si total == 2
				romper
		con @abrir()
			imprimir(@)
		var texto = "${prefijo}:${total}"
		var mapa = ["uno": 1, "dos": 2]
		var lista [decimal] = [1.5, -2.0]
		mapa["tres"] = lista.longitud()
		var copia = Base {id: 1}
		var posicional = Base {2}
		var elegido = si total > 1 "muchos" sino "pocos"
		var bloque = si total > 1
			"muchos"
		sino
			"pocos"
		var numero = intentar convertir(texto)
		var seguro = convertir(texto) atrapar |fallo|
			imprimir(fallo)
			0
		var defecto = mapa["cuatro"] o 4
		var generico = identidad<entero>(numero + seguro + defecto)
		var evento Evento<entero> = .Dato(generico)
		var otro = Evento<entero>.Vacio
		casos evento |dato|
			.Dato => imprimir(dato)
			_ => imprimir(otro)
		var nota = casos generico
			0..5 => "baja"
			5, 6 => "media"
			_ => "alta"
		var comoNombrable = (@ como Otra)
		imprimir(comoNombrable)
		imprimir(!(total == 0) && nota != elegido || bloque == posicional.id.formato(0))
		imprimir(copia)
		texto

	fn abrir() Ambito
		Ambito {}

tipo Ambito
	fn entrar()
		imprimir(1)
	fn salir()
		imprimir(2)

fn convertir(texto cadena) entero!
	texto.a_entero() o retornar .Error("no es un número")

fn identidad<T>(valor T) T valor

fn inicio()
	imprimir(mate.pi)
`

func parseEveryNode(t *testing.T) *ast.Program {
	t.Helper()
	tokens, err := lexer.Lex("todo.cometa", everyNode)
	if err != nil {
		t.Fatal(err)
	}
	program, err := parser.Parse("todo.cometa", tokens)
	if err != nil {
		t.Fatal(err)
	}
	// A BadStmt only comes out of a syntax error, which would drop other nodes.
	inicio := program.Decls[len(program.Decls)-1].(*ast.FuncDecl)
	inicio.Body = append(inicio.Body, &ast.BadStmt{Pos: inicio.Pos, Name: "roto"})
	return program
}

// reachable lists what a generic walk over the Go values of the tree finds:
// the reference Inspect is compared against.
func reachable(root any) map[any]bool {
	found := map[any]bool{}
	typeRef := reflect.TypeOf(ast.TypeRef{})
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			if v.IsNil() {
				return
			}
			if v.Kind() == reflect.Pointer && inspected(v.Interface()) {
				found[v.Interface()] = true
			}
			walk(v.Elem())
		case reflect.Struct:
			if v.Type() == typeRef && v.CanAddr() {
				found[v.Addr().Interface()] = true
			}
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(root))
	return found
}

func inspected(node any) bool {
	switch node.(type) {
	case ast.Decl, ast.Stmt, ast.Expr, *ast.Field, *ast.MatchArm, *ast.TypeRef:
		return true
	}
	return false
}

func TestInspectVisitsEveryNode(t *testing.T) {
	program := parseEveryNode(t)
	want := reachable(program)
	got := map[any]bool{}
	ast.Inspect(program, func(node any) bool {
		if !inspected(node) {
			t.Errorf("Inspect entregó un %T", node)
		}
		if got[node] {
			t.Errorf("Inspect visitó dos veces un %T", node)
		}
		got[node] = true
		return true
	})
	for node := range want {
		if !got[node] {
			t.Errorf("Inspect no visitó un %T (%+v)", node, node)
		}
	}
	for node := range got {
		if !want[node] {
			t.Errorf("Inspect visitó un %T que no está en el árbol", node)
		}
	}

	// The comparison above only proves something for the node types the sample
	// contains, so the sample must contain all of them.
	present := map[string]bool{}
	for node := range want {
		present[reflect.TypeOf(node).Elem().Name()] = true
	}
	for _, name := range nodeTypes(t) {
		if !present[name] {
			t.Errorf("el programa de ejemplo no contiene ningún %s: añádelo a everyNode y a Inspect", name)
		}
	}
}

// nodeTypes reads ast.go for the types that are declarations, statements or
// expressions: the ones with a declNode, stmtNode or exprNode method.
func nodeTypes(t *testing.T) []string {
	t.Helper()
	file, err := goparser.ParseFile(gotoken.NewFileSet(), "ast.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, decl := range file.Decls {
		method, ok := decl.(*goast.FuncDecl)
		if !ok || method.Recv == nil {
			continue
		}
		switch method.Name.Name {
		case "declNode", "stmtNode", "exprNode":
			names = append(names, method.Recv.List[0].Type.(*goast.StarExpr).X.(*goast.Ident).Name)
		}
	}
	sort.Strings(names)
	if len(names) < 30 {
		t.Fatalf("solo se encontraron %d tipos de nodo en ast.go", len(names))
	}
	return names
}

func TestInspectSkipsChildrenAndAcceptsEveryRoot(t *testing.T) {
	program := parseEveryNode(t)
	calls := 0
	ast.Inspect(program, func(node any) bool {
		if _, ok := node.(*ast.CallExpr); ok {
			calls++
		}
		_, function := node.(*ast.FuncDecl)
		return !function
	})
	if calls != 0 {
		t.Fatalf("se visitaron %d llamadas dentro de funciones que debían saltarse", calls)
	}

	method := program.Decls[6].(*ast.TypeDecl).Methods[0]
	count := func(root any) int {
		n := 0
		ast.Inspect(root, func(any) bool { n++; return true })
		return n
	}
	if body, whole := count(method.Body), count(method); body == 0 || body >= whole {
		t.Fatalf("raíces: cuerpo=%d, método=%d", body, whole)
	}
	if count(method.Body[0]) == 0 || count(method.Params[0].Default) != 1 || count(method.ReturnType) != 3 {
		t.Fatalf("raíces: sentencia=%d, expresión=%d, tipo=%d", count(method.Body[0]), count(method.Params[0].Default), count(method.ReturnType))
	}
	var none ast.Expr
	if count(nil) != 0 || count(none) != 0 || count((*ast.Program)(nil)) != 0 {
		t.Fatal("una raíz vacía no debe visitar nada")
	}
}
