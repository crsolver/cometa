package compiler

import (
	"strings"
	"testing"
)

func TestGameAPIRemovalsAndStartupEffects(t *testing.T) {
	for _, source := range []string{
		"fn iniciar() vectores.longitud(Vec2 {})\n",
		"fn iniciar() colisiones.rectangulos(Rect {}, Rect {})\n",
		"fn iniciar() imprimir(mat.pi)\n",
		"fn iniciar()\n\tvar config = ConfigJuego {}\n",
		"fn iniciar()\n\tvar n = Rect {}.longitud()\n",
		"fn iniciar()\n\tvar n = Vec2 {}.distancia_a(1)\n",
		"fn helper() juego.configuracion()\n",
		"fn helper() num\n\tjuego.configuracion()\n\t1\nvar x = helper()\nfn iniciar() helper()\n",
		"fn helper() juego.configuracion()\nfn iniciar() helper()\nfn actualizar(dt num) helper()\nfn pintar() imprimir(0)\n",
		"fn helper() juego.configuracion()\nfn iniciar() helper()\nfn actualizar(dt num) imprimir(dt)\nfn pintar() helper()\n",
		"tipo T\n\tfn ajustar() juego.configuracion()\n",
		"fn helper() num\n\tjuego.configuracion()\n\t1\nfn f(x num = helper()) imprimir(x)\nfn iniciar() f()\nfn actualizar(dt num) f()\nfn pintar() imprimir(0)\n",
		"interfaz I\n\tfn ajustar()\ntipo T\n\tfn ajustar() juego.configuracion()\nfn configurar(i I) i.ajustar()\nfn iniciar() configurar(T {})\nfn actualizar(dt num) configurar(T {})\nfn pintar() imprimir(0)\n",
		"fn actualizar(dt num) imprimir(dt)\nfn pintar() graficos.rectangulo(Vec2 {}, Vec2 {}, .Rojo)\n",
	} {
		t.Run(source, func(t *testing.T) {
			if !strings.Contains(source, "fn actualizar") {
				source += minimalGame
			}
			if _, err := Compile("game.hacha", []byte(source)); err == nil {
				t.Fatal("accepted invalid API use")
			} else if strings.Contains(source, "juego.configuracion") && !strings.Contains(err.Error(), "configuración solo se permite desde iniciar") {
				t.Fatalf("expected startup effect diagnostic, got %v", err)
			}
		})
	}
	for _, source := range []string{
		"",
		"fn iniciar() juego.configuracion()\n",
		"fn iniciar() juego.configuracion(titulo = \"Prueba\")\n",
		"fn helper() juego.configuracion(640,360)\nfn iniciar()\n\thelper()\n\tjuego.configuracion()\n",
		"fn configurar(x num) num x + 1\nfn iniciar() imprimir(configurar(1))\n",
		"tipo T\n\tfn ajustar() juego.configuracion()\nfn iniciar()\n\tT {}.ajustar()\n",
	} {
		if _, err := Compile("game.hacha", []byte(source+minimalGame)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGameValueMethodsRuntime(t *testing.T) {
	source := `interfaz Medida
	fn longitud() num
var orden num = 0
fn receptor() Vec2
	orden = orden * 10 + 1
	Vec2 {x: 3, y: 4}
fn argumento() num
	orden = orden * 10 + 2
	5
fn probar()
	var v = Vec2 {x: 3, y: 4}
	imprimir(v.longitud())
	imprimir(mate.redondear(v.normalizado().x * 10))
	imprimir(v.distancia_a(otro = {x: 6, y: 8}))
	imprimir(v.producto_punto({x: 2}))
	imprimir(mate.redondear(Vec2 {x: 1}.rotado(mate.pi / 2).y))
	imprimir(v.colision_circulo(radio_otro = 1, otro = {x: 9, y: 4}, radio = 5))
	var rect = Rect {tamano: {x: 10, y: 10}}
	imprimir(rect.contiene({}))
	imprimir(rect.contiene({x: 10}))
	imprimir(rect.interseca({pos: {x: 10}, tamano: {x: 1, y: 1}}))
	imprimir(rect.interseca({pos: {x: 9}, tamano: {x: 1, y: 1}}))
	var medida Medida = v
	imprimir(medida.longitud())
	imprimir(receptor().colision_circulo(radio_otro = argumento(), otro = {}, radio = 1))
	imprimir(orden)
fn actualizar(dt num) imprimir(dt)
fn pintar() imprimir(0)
`
	generated, err := Compile("game.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runnable := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1) + "\nfunc main() { Probar() }\n"
	runGeneratedGo(t, []byte(runnable), "5\n6\n5\n6\n1\ntrue\ntrue\nfalse\nfalse\ntrue\n5\ntrue\n12\n")
}

func TestGameConfigurationAndTransformsRuntime(t *testing.T) {
	generated, err := Compile("game.hacha", []byte("fn iniciar()\n\tjuego.configuracion(640, 360)\n\tjuego.configuracion(titulo = \"Final\")\n"+minimalGame))
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	if strings.Index(text, "Iniciar()\n\t_hgstarting = false") < 0 || strings.Index(text, "_hgvalidarConfig(_hgconfig)") < strings.LastIndex(text, "Iniciar()") {
		t.Fatal("incorrect startup order")
	}
	runnable := strings.Replace(text, "func main() {", "func unusedMain() {", 1) + `
func near(got,want float64){if math.Abs(got-want)>1e-8{panic(fmt.Sprint(got," != ",want))}}
func main(){
 if _hgconfig.Ancho!=320||_hgconfig.Alto!=180||_hgconfig.Tps!=60||_hgconfig.Escala!=1 {panic("bad defaults")}
 _hgstarting=true;Iniciar();_hgstarting=false
 if _hgconfig.Ancho!=320||_hgconfig.Alto!=180||_hgconfig.Titulo!="Final" {panic("setter must replace complete config")}
 _hgvalidarConfig(_hgconfig)
 func(){defer func(){if recover()==nil{panic("missing phase guard")}}();_hgconfiguracion(320,180,"",1,false,false,60)}()
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
