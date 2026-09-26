package compiler

import (
 "os"
 "strings"
 "testing"
)

func TestUIExample(t *testing.T) {
 source,err:=os.ReadFile("testdata/programas/ui.cometa");if err!=nil {t.Fatal(err)}
 generated,err:=Compile("ui.cometa",source);if err!=nil {t.Fatal(err)}
 for _,want:=range []string{"_hguiPintar","_hgContexto",".Entrar()",".Salir()","iVBOR"} {if !strings.Contains(string(generated),want){t.Fatal("missing",want)}}
}

func TestUIRendering(t *testing.T) {
 if testing.Short() {t.Skip("desktop rendering")}
 data,err:=os.ReadFile("testdata/programas/ui.cometa");if err!=nil {t.Fatal(err)}
 source:=strings.Replace(string(data),"fn inicio()","fn lanzar()",1)
 generated,err:=Compile("ui.cometa",[]byte(source));if err!=nil {t.Fatal(err)}
 // The capture runtime already imports os and image/png by name.
 code:=strings.Replace(string(generated),"import (","import (\n\"golang.org/x/image/font/gofont/goregular\"",1)
 code+=`
type uiCheck struct{}
func (*uiCheck) Layout(w,h int)(int,int){return 320,180}
func (*uiCheck) Draw(screen *ebiten.Image){}
func (*uiCheck) Update() error {
 c:=_hguiCrear();game:=&Opciones{Contexto:c,Volumen:.5,Musica:true}
 _hgupdating=true;_hgtick++;game.Actualizar(1.0/60);_hgupdating=false
 screen:=ebiten.NewImage(320,180);_hgscreen=screen
 _hgcamara=_hgCamara2D{Pos:_hgVec2{100,100},Zoom:3,Rotacion:1};saved:=_hgcamara
 _hguiPintar(c);if _hgcamara!=saved||_hgscreen!=screen {panic("graphics state leaked")}
 first:=make([]byte,320*180*4);screen.ReadPixels(first)
 screen.Clear();_hguiPintar(c);second:=make([]byte,len(first));screen.ReadPixels(second)
 if !bytes.Equal(first,second){panic("draw is not replayable")}
 if first[3]==0 {panic("panel missing")};if first[(179*320+319)*4+3]!=0 {panic("UI escaped viewport")}
 if path:=os.Getenv("COMETA_UI_SCREENSHOT");path!="" {f,e:=os.Create(path);if e!=nil{panic(e)};if e=png.Encode(f,screen);e!=nil{panic(e)};f.Close()}
 fontSource,e:=text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF));if e!=nil {panic(e)}
 _hgupdating=true;_hgtick++;root:=_hguiCuadro(c);root.Entrar();theme:=_hguiTema(_hguiFuente(&_hgFuente{source:fontSource},20));theme.Entrar();_hguiTexto("Texto TTF: áéñ",_hguiFijo(100),_hguiContenido(0,1e9));theme.Salir();root.Salir();_hgupdating=false
 if len(c.previous[1].lines)<2 {panic("TTF did not wrap")}
 screen.Clear();_hguiPintar(c);pixels:=make([]byte,len(first));screen.ReadPixels(pixels);found:=false;for i:=3;i<len(pixels);i+=4{if pixels[i]>0{found=true;break}};if !found {panic("TTF text absent")}
 fmt.Println("ok");return ebiten.Termination
}
func main(){ebiten.SetWindowVisible(false);ebiten.SetRunnableOnUnfocused(true);if err:=ebiten.RunGame(&uiCheck{});err!=nil{panic(err)}}
`
 runGeneratedGo(t,[]byte(code),"ok\n")
}
