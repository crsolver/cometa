package stdlib

import (
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "testing"
)

// Compile the actual embedded engine against tiny data-only graphics types.
// This catches runtime Go errors and exercises layout without a window/backend.
func TestUICore(t *testing.T) {
 dir:=t.TempDir()
 source:=`package main
import("math";"strings";"strconv";"testing")
type _hgVec2 struct { X,Y float64 }
type _hgRect struct { Pos,Tamano _hgVec2 }
type _hgColor struct { R,G,B,A uint8 }
type _hgFuente struct{}
type _hgImagen struct{}
func _hgUIBegin(c *_hgContexto) {}
`+uiCoreSource+`
func frame(c *_hgContexto,in _hgUIInput) {c.begin(_hgVec2{320,180},in,func(s string,t _hgTema)float64{return float64(len([]rune(s)))*8});_hgUIActive=c}
func end(c *_hgContexto) {c.finish();_hgUIActive=nil}
func fixed(v float64)*_hgMedida{return _hguiFijo(v)}
func fit()*_hgMedida{return _hguiContenido(0,1e9)}
func grow()*_hgMedida{return _hguiExpandir(0,1e9)}
func near(t *testing.T,a,b float64){t.Helper();if math.Abs(a-b)>0.0001 {t.Fatalf("got %v want %v",a,b)}}
func panics(t *testing.T,f func()){t.Helper();defer func(){if recover()==nil {t.Error("expected panic")}}();f()}
func TestSizing(t *testing.T){
 c:=_hguiCrear();frame(c,_hgUIInput{})
 row:=_hguiFila("row",fixed(300),fixed(60),10,_hgBordes{10,10,5,5},0,1,false);row.Entrar()
 _hguiBoton("a","a",fixed(40),fixed(20),true)
 _hguiBoton("b","b",_hguiExpandir(0,50),fixed(20),true)
 _hguiBoton("c","c",grow(),fixed(20),true)
 row.Salir();end(c)
 near(t,c.previous[2].rect.Pos.X,10);near(t,c.previous[2].rect.Pos.Y,20)
 near(t,c.previous[3].rect.Tamano.X,50);near(t,c.previous[4].rect.Tamano.X,170)
 frame(c,_hgUIInput{});p:=_hguiColumna("p",fixed(100),fixed(100),0,_hgBordes{},0,0,false);p.Entrar();_hguiTexto("uno dos tres cuatro",grow(),fit(),_hgColor{});p.Salir();end(c)
 if len(c.previous[2].lines)<2 {t.Fatal("text did not wrap")};near(t,c.previous[2].rect.Tamano.Y,float64(len(c.previous[2].lines))*8)
 frame(c,_hgUIInput{});p=_hguiColumna("p",fit(),fit(),0,_hgBordes{},0,0,false);p.Entrar();_hguiTexto("a",_hguiPorcentaje(.5),fit(),_hgColor{});p.Salir();panics(t,func(){end(c)});_hgUIActive=nil
 frame(c,_hgUIInput{});row2:=_hguiFila("row2",fixed(300),fixed(20),0,_hgBordes{},0,0,false);row2.Entrar();_hguiBoton("a","a",fixed(40),fixed(20),true);_hguiEspacio(grow(),fixed(20));_hguiBoton("b","b",fixed(40),fixed(20),true);row2.Salir();end(c)
 near(t,c.previous[3].rect.Tamano.X,220)
}
func TestBarra(t *testing.T){
 c:=_hguiCrear();frame(c,_hgUIInput{});_hguiBarra(5,0,10,fixed(100),fixed(10));end(c)
 near(t,c.previous[1].value,.5)
 frame(c,_hgUIInput{});_hguiBarra(-5,0,10,fixed(100),fixed(10));end(c);near(t,c.previous[1].value,0)
 frame(c,_hgUIInput{});_hguiBarra(50,0,10,fixed(100),fixed(10));end(c);near(t,c.previous[1].value,1)
 panics(t,func(){frame(c,_hgUIInput{});_hguiBarra(5,10,0,fixed(100),fixed(10));end(c)});_hgUIActive=nil
}
func TestSeparador(t *testing.T){
 c:=_hguiCrear();frame(c,_hgUIInput{});col:=_hguiColumna("col",fixed(100),fit(),0,_hgBordes{},0,0,false);col.Entrar();_hguiSeparador(grow(),fixed(1));col.Salir();end(c)
 near(t,c.previous[1].rect.Tamano.X,100);near(t,c.previous[1].rect.Tamano.Y,1)
}
func TestTextInput(t *testing.T){
 c:=_hguiCrear();v:=""
 build:=func(in _hgUIInput){frame(c,in);v=_hguiCampoTexto("f",v,fixed(100),fixed(20),true);end(c)}
 build(_hgUIInput{})
 build(_hgUIInput{tab:true})
 build(_hgUIInput{chars:[]rune("ab")});if v!="ab" {t.Fatalf("typing failed: %q",v)}
 build(_hgUIInput{chars:[]rune("c")});if v!="abc" {t.Fatalf("append failed: %q",v)}
 build(_hgUIInput{left:true});build(_hgUIInput{chars:[]rune("X")});if v!="abXc" {t.Fatalf("insert at cursor failed: %q",v)}
 build(_hgUIInput{backspace:true});if v!="abc" {t.Fatalf("backspace failed: %q",v)}
 build(_hgUIInput{home:true});build(_hgUIInput{delete:true});if v!="bc" {t.Fatalf("delete failed: %q",v)}
 build(_hgUIInput{end:true});build(_hgUIInput{backspace:true});if v!="b" {t.Fatalf("backspace at end failed: %q",v)}
 build(_hgUIInput{left:true});build(_hgUIInput{left:true});build(_hgUIInput{left:true});build(_hgUIInput{backspace:true});if v!="b" {t.Fatalf("backspace at start should be no-op: %q",v)}
 c2:=_hguiCrear();v2:="z"
 frame(c2,_hgUIInput{});v2=_hguiCampoTexto("g",v2,fixed(100),fixed(20),true);end(c2)
 frame(c2,_hgUIInput{chars:[]rune("Y")});v2=_hguiCampoTexto("g",v2,fixed(100),fixed(20),true);end(c2)
 if v2!="z" {t.Fatalf("unfocused field should ignore input: %q",v2)}
}
func TestInteraction(t *testing.T){
 c:=_hguiCrear();build:=func(in _hgUIInput,order bool)(bool,bool){frame(c,in);var a,b bool;if order {b=_hguiBoton("b","B",fixed(80),fixed(20),true)};a=_hguiBoton("a","A",fixed(80),fixed(20),true);if !order {b=_hguiBoton("b","B",fixed(80),fixed(20),true)};end(c);return a,b}
 if a,_:=build(_hgUIInput{pointer:_hgVec2{5,5},press:true},false);a {t.Fatal("new widget activated")}
 build(_hgUIInput{pointer:_hgVec2{5,5},press:true,down:true},false)
 a,b:=build(_hgUIInput{pointer:_hgVec2{5,5},release:true},true);if !a||b {t.Fatal("identity lost on reorder")}
 a,b=build(_hgUIInput{},true);if a||b {t.Fatal("repeated activation")}
 build(_hgUIInput{tab:true},true);a,b=build(_hgUIInput{activate:true},true);if a==b {t.Fatal("keyboard activation")}
 frame(c,_hgUIInput{});end(c);if c.focus!=""||c.active!="" {t.Fatal("removed widget retained state")}
 frame(c,_hgUIInput{});_hguiBoton("x","X",fit(),fit(),true);panics(t,func(){_hguiBoton("x","X",fit(),fit(),true)});_hgUIActive=nil
}
func TestScrollAndClip(t *testing.T){
 c:=_hguiCrear();build:=func(in _hgUIInput,height float64){frame(c,in);p:=_hguiPanel("p",fixed(100),fixed(40),0,_hgBordes{},0,0,true);p.Entrar();_hguiBoton("big","B",fixed(80),fixed(height),true);p.Salir();end(c)}
 build(_hgUIInput{},100);build(_hgUIInput{pointer:_hgVec2{5,5},wheel:_hgVec2{0,-1}},100)
 near(t,c.previous[2].rect.Pos.Y,-24);near(t,c.previous[2].clip.Tamano.Y,40)
 build(_hgUIInput{pointer:_hgVec2{5,50},press:true},100);if c.active!=""{t.Fatal("clipped target captured input")}
 build(_hgUIInput{},10);near(t,c.previous[2].rect.Pos.Y,0)
}
func TestSliderCapture(t *testing.T){
 c:=_hguiCrear();v:=0.0;build:=func(in _hgUIInput){frame(c,in);v=_hguiDeslizador("s",v,0,1,fixed(100),fixed(20),true);end(c)}
 build(_hgUIInput{});build(_hgUIInput{pointer:_hgVec2{25,5},press:true,down:true});near(t,v,.25)
 build(_hgUIInput{pointer:_hgVec2{400,5},down:true});near(t,v,1)
 build(_hgUIInput{pointer:_hgVec2{400,5},release:true});if c.active!=""{t.Fatal("capture not released")}
 build(_hgUIInput{left:true});near(t,v,.99)
}
func TestTextColor(t *testing.T){
 c:=_hguiCrear();frame(c,_hgUIInput{});_hguiTexto("a",fit(),fit(),_hgColor{});_hguiTexto("b",fit(),fit(),_hgColor{R:255,A:255});end(c)
 if c.previous[1].tint.A!=0 {t.Fatal("default text color should be transparent (inherit theme)")}
 if c.previous[2].tint.R!=255||c.previous[2].tint.A!=255 {t.Fatal("explicit text color not stored")}
}
func TestThemeSnapshot(t *testing.T){c:=_hguiCrear();frame(c,_hgUIInput{});theme:=_hguiRetro(2);s:=_hguiTema(theme);theme.Espacio=99;s.Entrar();_hguiTexto("a",fit(),fit(),_hgColor{});s.Salir();end(c);near(t,c.previous[1].theme.Espacio,4)}
`
 file:=filepath.Join(dir,"ui_test.go");if err:=os.WriteFile(file,[]byte(source),0600);err!=nil {t.Fatal(err)}
 exe:=filepath.Join(runtime.GOROOT(),"bin","go");if runtime.GOOS=="windows" {exe+=".exe"}
 cmd:=exec.Command(exe,"test",file,"-count=1");out,err:=cmd.CombinedOutput();if err!=nil {t.Fatalf("UI core: %v\n%s",err,out)}
}
