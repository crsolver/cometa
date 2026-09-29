package compiler

import (
	"strings"
	"testing"
)

func TestRemovedGameConfiguration(t *testing.T) {
	_, err := Compile("game.cometa", []byte("usar std/pincel\nfn inicio() pincel.configuracion()\n"))
	if err == nil || !strings.Contains(err.Error(), "configuracion") {
		t.Fatalf("removed API: %v", err)
	}
}

func TestGameValueMethodsRuntime(t *testing.T) {
	source := `interfaz Medida
	fn longitud() decimal
var orden decimal = 0
fn receptor() mate.Vec2
	orden = orden * 10 + 1
	mate.Vec2 {x: 3, y: 4}
fn argumento() decimal
	orden = orden * 10 + 2
	5
fn probar()
	var v = mate.Vec2 {x: 3, y: 4}
	imprimir(v.longitud())
	imprimir(mate.redondear(v.normalizado().x * 10))
	imprimir(v.distancia_a(otro = {x: 6, y: 8}))
	imprimir(v.producto_punto({x: 2}))
	imprimir(mate.redondear(mate.Vec2 {x: 1}.rotado(mate.pi / 2).y))
	imprimir(v.colision_circulo(radio_otro = 1, otro = {x: 9, y: 4}, radio = 5))
	var rect = mate.Rect {tamano: {x: 10, y: 10}}
	imprimir(rect.contiene({}))
	imprimir(rect.contiene({x: 10}))
	imprimir(rect.interseca({pos: {x: 10}, tamano: {x: 1, y: 1}}))
	imprimir(rect.interseca({pos: {x: 9}, tamano: {x: 1, y: 1}}))
	var medida Medida = v
	imprimir(medida.longitud())
	imprimir(receptor().colision_circulo(radio_otro = argumento(), otro = {}, radio = 1))
	imprimir(orden)
fn actualizar(dt decimal) imprimir(dt)
fn pintar() imprimir(0)
`
	generated, err := Compile("game.cometa", []byte(pincelImports+source))
	if err != nil {
		t.Fatal(err)
	}
	runnable := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1) + "\nfunc main() { Probar() }\n"
	runGeneratedGo(t, []byte(runnable), "5\n6\n5\n6\n1\nverdadero\nverdadero\nfalso\nfalso\nverdadero\n5\nverdadero\n12\n")
}

func TestGameConfigurationAndTransformsRuntime(t *testing.T) {
	generated, err := Compile("game.cometa", []byte(pincelImports+minimalGame))
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	runnable := strings.Replace(text, "func main() {", "func unusedMain() {", 1) + `
func near(got,want float64){if math.Abs(got-want)>1e-8{panic(fmt.Sprint(got," != ",want))}}
func main(){
 if _hgconfig.Ancho!=320||_hgconfig.Alto!=180||_hgconfig.Tps!=60||_hgconfig.Escala!=1 {panic("bad defaults")}
 if err:=_hgvalidarConfig(_hgconfig);err!=nil{panic(err)}
 func(){defer func(){if recover()==nil{panic("missing phase guard")}}();_hglimpiar(_hgColor{})}()
 p,o,s:=_hgVec2{20,30},_hgVec2{2,3},_hgVec2{4,5}
 m:=_hgtransform(p,o,s,math.Pi/2,true)
 x,y:=m.Apply(1,1);near(x,18);near(y,32)
 m=_hgtransform(p,o,s,math.Pi/2,false)
 x,y=m.Apply(1,1);near(x,30);near(y,26)
 _hgcamara=_hgCamara2D{Pos:_hgVec2{10,20},Origen:_hgVec2{1,2},Zoom:2,Rotacion:math.Pi/2}
 m=_hgtransform(p,o,s,math.Pi/2,true)
 x,y=m.Apply(1,1);near(x,25);near(y,-14)
 fmt.Println("ok")
}
`
	runGeneratedGo(t, []byte(runnable), "ok\n")
}
