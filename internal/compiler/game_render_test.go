package compiler

import (
	"strings"
	"testing"
)

// Run one hidden desktop tick: pixel readback needs Ebitengine's graphics loop.
func TestGameDrawingVariantsRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	game := "var orden decimal = 0\nfn siguiente() decimal\n\torden = orden + 1\n\torden\nfn actualizar(dt decimal) imprimir(dt)\nfn pintar()\n\tgraficos.rectangulo(color = .Rojo, alto = siguiente(), ancho = siguiente(), y = siguiente(), x = siguiente(), rotacion = 0)\n"
	generated, err := Compile("game.cometa", []byte(pincelImports+game))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1)
	source += `
type renderCheck struct{}
func (*renderCheck) Layout(w,h int)(int,int){return 64,64}
func (*renderCheck) Draw(screen *ebiten.Image){}
func pixels(draw func()) []byte {
 _hgscreen.Clear();draw()
 out:=make([]byte,64*64*4);_hgscreen.ReadPixels(out);return out
}
func equalDraw(a,b func()){
 pa,pb:=pixels(a),pixels(b)
 if !bytes.Equal(pa,pb){panic("drawing variants differ")}
 if bytes.Equal(pa,make([]byte,len(pa))){panic("drawing was empty")}
}
func (*renderCheck) Update()error {
 _hgscreen=ebiten.NewImage(64,64)
 defer func(){_hgscreen.Dispose();_hgscreen=nil}()
 p,s,o:=_hgVec2{28,26},_hgVec2{12,8},_hgVec2{6,4}
 rect:=_hgRect{p,s};c:=_hgColor{255,0,0,255}
 raw:=ebiten.NewImage(8,8);raw.Fill(color.White)
 img:=&_hgImagen{image:raw}
 font:=_hgfuentePredeterminada()
 for _,angle:=range []float64{0,math.Pi/2}{
  for _,camera:=range []_hgCamara2D{{Zoom:1},{Pos:_hgVec2{3,4},Origen:_hgVec2{2,1},Zoom:1.2,Rotacion:0.1}}{
   _hgcamara=camera
   equalDraw(func(){_hgrectanguloXY(28,26,12,8,c,o,angle)},func(){_hgrectangulo(p,s,c,o,angle)})
   equalDraw(func(){_hgrectanguloRect(rect,c,o,angle)},func(){_hgrectangulo(p,s,c,o,angle)})
   equalDraw(func(){_hgcirculoXY(28,26,6,c)},func(){_hgcirculo(p,6,c)})
   equalDraw(func(){_hglineaXY(28,26,12,8,c,2)},func(){_hglinea(p,s,c,2)})
   equalDraw(func(){_hgimagenXY(img,28,26,o,_hgVec2{1,1},angle,c,_hgColor{})},func(){_hgimagen(img,p,o,_hgVec2{1,1},angle,c,_hgColor{})})
   equalDraw(func(){_hgimagenRect(img,rect,o,angle,c,_hgColor{})},func(){_hgrectanguloRect(rect,c,o,angle)})
   crop:=_hgRect{_hgVec2{2,2},_hgVec2{4,4}}
   equalDraw(func(){_hgregionRect(img,crop,rect,o,angle,c,_hgColor{})},func(){_hgimagenRect(img,rect,o,angle,c,_hgColor{})})
   equalDraw(func(){_hgregionXY(img,crop,28,26,o,_hgVec2{2,2},angle,c,_hgColor{})},func(){_hgregion(img,crop,p,o,_hgVec2{2,2},angle,c,_hgColor{})})
   equalDraw(func(){_hgtextoXY("Hi",font,28,26,12,c,o,angle)},func(){_hgtexto("Hi",font,p,12,c,o,angle)})
  }
 }
 _hgrestablecerCamara()
 equalDraw(func(){_hgtextoDepuracionXY("Hi",10,10)},func(){_hgtextoDepuracion("Hi",_hgVec2{10,10})})
 // Verify a nonzero crop offset selects the source, not the whole image.
 raw.Fill(color.Black);raw.SubImage(image.Rect(2,2,6,6)).(*ebiten.Image).Fill(color.White)
 equalDraw(func(){_hgregionRect(img,_hgRect{_hgVec2{2,2},_hgVec2{4,4}},rect,o,0,c,_hgColor{})},func(){_hgrectanguloRect(rect,c,o,0)})
 // Rectangle origin maps to the requested position under a quarter turn.
 pixels(func(){_hgrectanguloRect(rect,c,o,math.Pi/2)})
 if _,_,_,a:=_hgscreen.At(28,26).RGBA();a==0 {panic("rotated center missing")}
 // Named drawing arguments must execute once in source order.
 pixels(func(){Pintar()})
 if Orden!=4 {panic("drawing arguments evaluated more than once")}
 if r,_,_,a:=_hgscreen.At(4,3).RGBA();r==0||a==0 {panic("named drawing argument order changed")}
 if _,_,_,a:=_hgscreen.At(3,4).RGBA();a!=0 {panic("named drawing arguments mapped incorrectly")}
 fmt.Println("ok")
 return ebiten.Termination
}
func main(){
 ebiten.SetWindowVisible(false)
 ebiten.SetRunnableOnUnfocused(true)
 if err:=ebiten.RunGame(&renderCheck{});err!=nil {panic(err)}
}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
