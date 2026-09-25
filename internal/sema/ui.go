package sema

import (
 "hacha/internal/ast"
 "hacha/internal/stdlib"
)

// Check known lifecycle roots and their helper/default/implicit-scope calls.
// Unknown dynamic calls are also guarded by the native runtime.
func (c *checker) checkUIPhases(program *ast.Program) error {
	used:=false;for _,f:=range c.model.Game.Calls {if f.Namespace=="ui" {used=true;break}};if !used{return nil}
 var inspect func(any,string,map[*ast.FuncDecl]bool) error
 inspect=func(node any,phase string,seen map[*ast.FuncDecl]bool) error {
  var failure error
  follow:=func(f *ast.FuncDecl) {if f!=nil&&!seen[f] {seen[f]=true;if err:=inspect(f,phase,seen);err!=nil {failure=err}}}
  WalkSyntax(node,func(node any){
   if failure!=nil {return}
   switch n:=node.(type) {
   case *ast.ScopeStmt:
    typ:=c.model.ExprTypes[n.Value]
    if typ.Name==stdlib.Symbol("AmbitoUI")&&phase!="actualizar" {failure=c.fail(n.Pos,"ámbitos UI solo se permiten desde actualizar");return}
    for _,name:=range []string{"entrar","salir"} {follow(c.model.Methods(typ)[name].Decl)}
   case *ast.CallExpr:
    if f,ok:=c.model.Game.Calls[n];ok&&f.Namespace=="ui" {
     switch f.Name {
     case "pintar":if phase!="pintar" {failure=c.fail(n.Pos,"ui.pintar solo se permite desde pintar")}
     case "fila","columna","panel","texto","imagen","boton","casilla","deslizador":if phase!="actualizar" {failure=c.fail(n.Pos,"ui.%s solo se permite desde actualizar",f.Name)}
     }
    }
    info:=c.model.Calls[n].Signature;follow(info.Decl)
    if member,ok:=n.Callee.(*ast.MemberExpr);ok&&c.model.ExprTypes[member.Object].Kind==Interface {for _,t:=range c.model.Types {follow(t.Methods[member.Name].Decl)}}
   }
  })
  return failure
 }
 for _,decl:=range program.Decls {
  if global,ok:=decl.(*ast.GlobalDecl);ok {if err:=inspect(global.Value,"inicialización",map[*ast.FuncDecl]bool{});err!=nil{return err}}
  functions:=[]*ast.FuncDecl{}
  switch d:=decl.(type) {case *ast.FuncDecl:functions=append(functions,d);case *ast.TypeDecl:functions=append(functions,d.Methods...)}
  for _,f:=range functions {if f.Name=="inicio"||f.Name=="actualizar"||f.Name=="pintar" {if err:=inspect(f,f.Name,map[*ast.FuncDecl]bool{f:true});err!=nil{return err}}}
 }
 return nil
}
