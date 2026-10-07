package compiler

import (
	"strings"
	"testing"
)

// Draws lienzo-built sprite sheets and grids in one hidden Ebitengine tick and
// checks screen pixels: mirroring, origins, empty tiles, animated tiles and the
// camera must keep each quadrant of the frame where it belongs.
func TestSpriteSheetAndGridPixels(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	program := `usar std/pincel
usar std/pincel/graficos
usar std/pincel/rejilla
usar std/pincel/lienzo
usar std/mate

// Two 4x4 frames: frame 0 has red/blue on top and green/yellow below; frame 1 is magenta.
fn sprites() graficos.Hoja
	var img = lienzo.desde_texto(["RRBBMMMM", "RRBBMMMM", "GGYYMMMM", "GGYYMMMM"], ["R": .Rojo, "B": .Azul, "G": .Verde, "Y": .Amarillo, "M": .Magenta])
	graficos.hoja(img, 4, 4)

// Cells hold frame 1, frame 0 and an empty tile.
fn mapa() rejilla.Rejilla
	rejilla.desde_texto(["ab."], ["a": 1, "b": 0, ".": -1])

// The inner clip is limited to the outer one; leaving each block restores the previous target.
fn recortes(h graficos.Hoja, salir bool)
	con graficos.recortar(mate.Rect {pos: {11, 11}, tamano: {2, 2}})
		con graficos.recortar(mate.Rect {pos: {12, 12}, tamano: {20, 20}})
			graficos.cuadro(h, 0, 10, 10)
			si salir
				retornar
		graficos.cuadro(h, 0, 30, 30)
		graficos.cuadro(h, 1, 8, 8)
	graficos.cuadro(h, 0, 40, 40)

// Inside the block everything lands on the image, without the caller's camera; the nested clip is in image pixels.
fn pintada(img graficos.Imagen, h graficos.Hoja, salir bool)
	con graficos.pintar_en(img)
		graficos.limpiar(.Azul)
		con graficos.recortar(mate.Rect {pos: {0, 0}, tamano: {4, 4}})
			graficos.cuadro(h, 1, 2, 2)
			si salir
				retornar
	graficos.cuadro(h, 0, 40, 40)

fn inicio()
	imprimir(0)
`
	generated, err := Compile("sprites.cometa", []byte(program))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1)
	source += `
type spriteCheck struct{}
func (*spriteCheck) Layout(w,h int)(int,int){return 64,64}
func (*spriteCheck) Draw(screen *ebiten.Image){}
var (
 rojo=_hgColor{255,0,0,255};azul=_hgColor{0,0,255,255};verde=_hgColor{0,255,0,255}
 amarillo=_hgColor{255,255,0,255};magenta=_hgColor{255,0,255,255};nada=_hgColor{}
)
var sinColores struct { tag int; payload1 *_hgRejilla }
type expect struct{x,y int;c _hgColor}
func check(name string,draw func(),want ...expect){
 _hgscreen.Clear();_hgrestablecerCamara();draw();_hgquitarRecorte()
 px:=make([]byte,64*64*4);_hgscreen.ReadPixels(px)
 for _,w:=range want{
  i:=(w.y*64+w.x)*4
  got:=_hgColor{px[i],px[i+1],px[i+2],px[i+3]}
  if got!=w.c{panic(fmt.Sprintf("%s: pixel (%d,%d) = %v, want %v",name,w.x,w.y,got,w.c))}
 }
}
func mustPanic(name,want string,f func()){
 defer func(){if r:=recover();fmt.Sprint(r)!=want{panic(fmt.Sprintf("%s: panic = %v, want %q",name,r,want))}}()
 f()
}
func (*spriteCheck) Update() error {
 _hgscreen=ebiten.NewImage(64,64)
 defer func(){_hgscreen.Dispose();_hgscreen=nil}()
 h:=Sprites();m:=Mapa()
 one,zero:=_hgVec2{1,1},_hgVec2{}
 at:=_hgVec2{10,10}
 check("plain",func(){_hgcuadro(h,0,at,zero,one,0,_hgColor{255,255,255,255},false,false,nada)},
  expect{10,10,rojo},expect{13,10,azul},expect{10,13,verde},expect{13,13,amarillo},expect{9,10,nada},expect{14,10,nada},expect{10,14,nada})
 check("second frame",func(){_hgcuadro(h,1,at,zero,one,0,_hgColor{255,255,255,255},false,false,nada)},expect{10,10,magenta},expect{13,13,magenta})
 check("out of range frame",func(){_hgcuadro(h,2,at,zero,one,0,_hgColor{255,255,255,255},false,false,nada);_hgcuadro(h,-1,at,zero,one,0,_hgColor{255,255,255,255},false,false,nada)},expect{10,10,nada},expect{13,13,nada})
 // Mirroring flips the frame in place: the box stays at [10,14).
 check("mirror h",func(){_hgcuadro(h,0,at,zero,one,0,_hgColor{255,255,255,255},true,false,nada)},
  expect{10,10,azul},expect{13,10,rojo},expect{10,13,amarillo},expect{9,10,nada},expect{14,10,nada})
 check("mirror v",func(){_hgcuadro(h,0,at,zero,one,0,_hgColor{255,255,255,255},false,true,nada)},
  expect{10,10,verde},expect{10,13,rojo},expect{13,10,amarillo},expect{10,9,nada},expect{10,14,nada})
 check("mirror both",func(){_hgcuadro(h,0,at,zero,one,0,_hgColor{255,255,255,255},true,true,nada)},expect{10,10,amarillo},expect{13,13,rojo})
 // The origin is a source pixel; scaled by 2 the frame spans [6,14).
 check("origin scaled",func(){_hgcuadro(h,0,at,_hgVec2{2,2},_hgVec2{2,2},0,_hgColor{255,255,255,255},false,false,nada)},
  expect{6,6,rojo},expect{13,6,azul},expect{13,13,amarillo},expect{5,6,nada},expect{14,6,nada})
 // A mirrored frame keeps its origin point fixed: with origin x=1 the box stays at [9,13).
 check("origin mirrored",func(){_hgcuadro(h,0,at,_hgVec2{1,0},one,0,_hgColor{255,255,255,255},true,false,nada)},
  expect{9,10,azul},expect{12,10,rojo},expect{8,10,nada},expect{13,10,nada})
 check("tint",func(){_hgcuadro(h,1,at,zero,one,0,_hgColor{0,255,255,255},false,false,nada)},expect{10,10,azul})
 // A white fill paints the silhouette, keeps transparency and wins over the tint.
 blanco:=_hgColor{255,255,255,255}
 check("flash",func(){_hgcuadro(h,0,at,zero,one,0,_hgColor{0,255,255,255},false,false,blanco)},
  expect{10,10,blanco},expect{13,13,blanco},expect{9,10,nada},expect{14,14,nada})
 check("flash image",func(){_hgimagenXY(h.img,10,10,zero,one,0,_hgColor{255,255,255,255},blanco)},expect{10,10,blanco},expect{17,13,blanco},expect{18,10,nada})
 check("half flash",func(){_hgcuadro(h,0,at,zero,one,0,_hgColor{255,255,255,255},false,false,_hgColor{255,255,255,128})})
 if px:=_hgscreen.At(10,10).(color.RGBA);px.R!=255||px.G<125||px.G>131||px.B<125||px.B>131 {panic(fmt.Sprintf("half flash: pixel = %v",px))}
 // Clipping keeps screen coordinates and only lets the clipped area through, until it is removed.
 check("clip",func(){_hgrecortar(_hgRect{_hgVec2{11,11},_hgVec2{2,2}});_hgcuadro(h,0,at,zero,one,0,blanco,false,false,nada)},
  expect{11,11,rojo},expect{12,12,amarillo},expect{10,10,nada},expect{13,13,nada})
 check("clip removed",func(){_hgrecortar(_hgRect{_hgVec2{0,0},_hgVec2{1,1}});_hgquitarRecorte();_hgcuadro(h,0,at,zero,one,0,blanco,false,false,nada)},expect{10,10,rojo},expect{13,13,amarillo})
 check("clip outside",func(){_hgrecortar(_hgRect{_hgVec2{100,100},_hgVec2{5,5}});_hgcuadro(h,0,at,zero,one,0,blanco,false,false,nada);_hgquitarRecorte()},expect{10,10,nada})
 // con blocks: frame 0 only shows through the intersection (12,12); once the inner block ends, frame 1 reaches the rest of the outer clip.
 check("clip scopes",func(){Recortes(h,false)},expect{11,11,magenta},expect{12,12,amarillo},expect{10,10,nada},expect{13,13,nada},expect{30,30,nada},expect{40,40,rojo})
 // An early return leaves both blocks, so later drawing is unclipped.
 check("clip scopes return",func(){Recortes(h,true);_hgcuadro(h,0,_hgVec2{40,40},zero,one,0,blanco,false,false,nada)},
  expect{12,12,amarillo},expect{11,11,nada},expect{13,13,nada},expect{40,40,rojo})
 // pintar_en: the screen is untouched inside the block and the camera comes back after it.
 lz:=_hglienzoNuevo(8,8,_hgColor{})
 drawn:=func(){_hgimagenXY(lz,20,20,zero,one,0,blanco,nada)}
 check("target",func(){_hgusarCamara(_hgCamara2D{Pos:_hgVec2{-5,0},Zoom:1});Pintada(lz,h,false)},expect{0,0,nada},expect{2,2,nada},expect{45,40,rojo},expect{40,40,nada})
 check("target drawn",drawn,expect{20,20,azul},expect{22,22,magenta},expect{23,23,magenta},expect{24,24,azul},expect{27,27,azul},expect{28,28,nada})
 check("target return",func(){Pintada(lz,h,true);_hgcuadro(h,0,_hgVec2{40,40},zero,one,0,blanco,false,false,nada)},expect{0,0,nada},expect{40,40,rojo})
 if _hgdestinoActual!=nil {panic("target left open after return")}
 // lienzo reads what was painted, and editing afterwards keeps it.
 if c:=_hglienzoLeerPixel(lz,3,3).payload1;c!=magenta {panic(fmt.Sprintf("target read back = %v",c))}
 _hglienzoPixel(lz,0,0,rojo)
 check("target edited",drawn,expect{20,20,rojo},expect{21,21,azul},expect{22,22,magenta})
 // Outside pintar the block supplies the target and leaves none behind.
 pantalla:=_hgscreen;_hgscreen=nil
 fuera:=_hgpintarEn(lz);fuera.Entrar();_hglimpiar(verde);fuera.Salir()
 if _hgscreen!=nil {panic("target left a screen outside pintar")}
 _hgscreen=pantalla
 check("target outside pintar",drawn,expect{20,20,verde},expect{27,27,verde})
 mustPanic("self draw","no se puede dibujar una imagen sobre sí misma",func(){d:=_hgpintarEn(lz);d.Entrar();defer d.Salir();drawn()})
 mustPanic("free target","no se puede liberar una imagen mientras se pinta en ella",func(){d:=_hgpintarEn(lz);d.Entrar();defer d.Salir();_hgliberar(lz)})
 // Freeing drops the GPU copy: unread paint is lost and the CPU pixels are uploaded again.
 _hgliberar(lz);_hgliberar(lz)
 check("freed",drawn,expect{20,20,rojo},expect{21,21,azul},expect{22,22,magenta})
 otra:=_hgpintarEn(lz);otra.Entrar();_hglimpiar(verde);otra.Salir()
 if c:=_hglienzoLeerPixel(lz,0,0).payload1;c!=verde {panic(fmt.Sprintf("paint read back = %v",c))}
 _hgliberar(lz)
 check("freed after read",drawn,expect{20,20,verde})
 grid:=_hgVec2{20,30}
 check("grid",func(){_hgrejillaDibujar(m,h,grid,one,_hgColor{255,255,255,255},nil,8,sinColores)},
  expect{20,30,magenta},expect{23,33,magenta},expect{24,30,rojo},expect{27,33,amarillo},expect{28,30,nada},expect{19,30,nada})
 check("grid scaled",func(){_hgrejillaDibujar(m,h,_hgVec2{0,0},_hgVec2{2,2},_hgColor{255,255,255,255},nil,8,sinColores)},
  expect{7,7,magenta},expect{8,0,rojo},expect{15,7,amarillo},expect{16,0,nada})
 // A color grid tints each cell with its own 0xRRGGBB; negative cells keep the tint.
 colores:=_hgrejillaNueva(3,1,-1);colores.Poner(0,0,0x0000FF)
 conColores:=struct { tag int; payload1 *_hgRejilla }{1,colores}
 check("cell colors",func(){_hgrejillaDibujar(m,h,grid,one,_hgColor{255,255,255,255},nil,8,conColores)},expect{20,30,azul},expect{24,30,rojo},expect{27,33,amarillo})
 // Animated cells: value 1 cycles frames [0, 1] at 8 per second (60 TPS).
 anim:=map[int64][]int64{1:{0,1}}
 _hgtick=0
 check("animated first",func(){_hgrejillaDibujar(m,h,grid,one,_hgColor{255,255,255,255},anim,8,sinColores)},expect{20,30,rojo},expect{24,30,rojo})
 _hgtick=8
 check("animated second",func(){_hgrejillaDibujar(m,h,grid,one,_hgColor{255,255,255,255},anim,8,sinColores)},expect{20,30,magenta},expect{24,30,rojo})
 _hgtick=15
 check("animated wraps",func(){_hgrejillaDibujar(m,h,grid,one,_hgColor{255,255,255,255},anim,8,sinColores)},expect{20,30,rojo})
 // Nearest sampling keeps fractional positions crisp: every source pixel still covers whole screen pixels,
 // so pixel art needs no manual rounding (including exact halves and fractional camera offsets).
 check("fractional down",func(){_hgcuadro(h,0,_hgVec2{10.4,10.4},zero,one,0,blanco,false,false,nada)},expect{10,10,rojo},expect{13,13,amarillo},expect{14,14,nada})
 check("fractional half",func(){_hgcuadro(h,0,_hgVec2{10.5,10.5},zero,one,0,blanco,false,false,nada)},expect{10,10,nada},expect{11,11,rojo},expect{12,11,rojo},expect{13,11,azul},expect{14,14,amarillo},expect{15,15,nada})
 check("fractional scaled",func(){_hgcuadro(h,0,_hgVec2{10.6,10.6},zero,_hgVec2{2,2},0,blanco,false,false,nada)},expect{11,11,rojo},expect{14,14,rojo},expect{15,15,amarillo},expect{10,10,nada},expect{18,18,amarillo},expect{19,19,nada})
 check("fractional camera",func(){_hgusarCamara(_hgCamara2D{Pos:_hgVec2{-0.7,0},Zoom:1});_hgcuadro(h,0,at,zero,one,0,blanco,false,false,nada)},expect{11,10,rojo},expect{10,10,nada})
 // The camera moves sprites and grids alike.
 cam:=_hgCamara2D{Pos:_hgVec2{-5,0},Zoom:1}
 check("camera sprite",func(){_hgusarCamara(cam);_hgcuadro(h,0,at,zero,one,0,_hgColor{255,255,255,255},false,false,nada)},expect{15,10,rojo},expect{10,10,nada})
 check("camera grid",func(){_hgusarCamara(cam);_hgrejillaDibujar(m,h,grid,one,_hgColor{255,255,255,255},nil,8,sinColores)},expect{25,30,magenta},expect{29,30,rojo},expect{20,30,nada})
 fmt.Println("ok")
 return ebiten.Termination
}
func main(){
 ebiten.SetWindowVisible(false)
 ebiten.SetRunnableOnUnfocused(true)
 if err:=ebiten.RunGame(&spriteCheck{});err!=nil {panic(err)}
}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
