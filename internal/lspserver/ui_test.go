package lspserver

import (
 "context"
 "strings"
 "testing"
 "github.com/owenrumney/go-lsp/lsp"
 "github.com/crsolver/cometa/internal/stdlib"
)

func TestUIToolingInScope(t *testing.T) {
 h:=NewHandler();uri:=lsp.DocumentURI("file:///ui.cometa")
 source,pos:=markerPosition("usar std/pincel/ui como u\nvar contexto = u.crear()\nfn actualizar(dt decimal)\n\tcon u.cuadro(contexto)\n\t\tcon u.columna(\"menu\")\n\t\t\tsi u.§boton(\"a\", \"Aceptar\")\n\t\t\t\timprimir(1)\n")
 if _,err:=h.documents.Open(&lsp.DidOpenTextDocumentParams{TextDocument:lsp.TextDocumentItem{URI:uri,Text:source}});err!=nil {t.Fatal(err)}
 params:=lsp.TextDocumentPositionParams{TextDocument:lsp.TextDocumentIdentifier{URI:uri},Position:pos}
 list,err:=h.Completion(context.Background(),&lsp.CompletionParams{TextDocumentPositionParams:params});if err!=nil {t.Fatal(err)}
 found:=false;for _,item:=range list.Items {if item.Label=="boton(...)" {found=true}};if !found {t.Fatal("missing button completion")}
 hover,err:=h.Hover(context.Background(),&lsp.HoverParams{TextDocumentPositionParams:params});if err!=nil||hover==nil||!strings.Contains(hover.Contents.Value(),"boton("){t.Fatalf("hover: %+v %v",hover,err)}
 locations,err:=h.Definition(context.Background(),&lsp.DefinitionParams{TextDocumentPositionParams:params});if err!=nil||len(locations)!=1||locations[0].URI!="cometa-std:///std/pincel/ui.cometa" {t.Fatalf("definition: %+v %v",locations,err)}
 reference,_:=stdlib.Source("std/pincel/ui");if !strings.Contains(strings.Split(reference,"\n")[locations[0].Range.Start.Line],"fn boton("){t.Fatal("wrong definition line")}
}
