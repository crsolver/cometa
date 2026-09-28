package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetroAPI(t *testing.T) {
	source := "usar std/pincel/retro como r\nfn pintar()\n\tr.texto(\"¡Niño!\", 0, 0)\n\tr.icono(.Corazon, color = .Rojo, y = 8, x = 0)\n\tr.glifo(255, 8, 8, atlas = .ASCII)\n"
	got, err := Compile("retro.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"_hgretroTexto", "_hgretroIcono", "_hgAtlas(1)", "iVBOR"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, source := range []string{
		"fn pintar() retro.texto(\"hola\", 0, 0)\n",
		"usar std/pincel/retro\nfn pintar() retro.icono(.NoExiste, 0, 0)\n",
		"usar std/pincel/retro\nfn pintar() retro.glifo(0, 0, 0, escala = 1.5)\n",
		"usar std/pincel/retro\nfn pintar() retro.icono(.ASCII, 0, 0)\n",
	} {
		if _, err := Compile("invalid.cometa", []byte(source)); err == nil {
			t.Fatal("accepted invalid retro call", source)
		}
	}
	for _, file := range []string{"dungeon2.cometa", "retro.cometa", "escape_retro.cometa"} {
		path := filepath.Join("testdata", "programas", file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(path, data); err != nil {
			t.Fatal(err)
		}
	}
	got, err = Compile("plain.cometa", []byte("usar std/pincel/graficos\nfn pintar() graficos.limpiar(.Negro)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "_hgretro") || strings.Contains(string(got), "iVBOR") {
		t.Fatal("unrelated module embeds retro assets")
	}
}

func TestRetroRenderingRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	generated, err := Compile("retro.cometa", []byte("usar std/pincel/retro\nfn pintar() retro.texto(\"hola\", 0, 0)\n"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	if !strings.Contains(source, `"fmt"`) {
		source = strings.Replace(source, "import (", "import (\n\"fmt\"", 1)
	}
	source += `
type retroCheck struct{}
func (*retroCheck) Layout(w,h int)(int,int){return 48,48}
func (*retroCheck) Draw(screen *ebiten.Image){}
func mustPanic(f func()) {defer func(){if recover()==nil{panic("expected validation error")}}();f()}
func (*retroCheck) Update() error {
 _hgscreen=ebiten.NewImage(48,48)
 defer func(){_hgscreen.Dispose();_hgscreen=nil}()
 _hgcamara=_hgCamara2D{Zoom:1}
 white:=_hgColor{255,255,255,255}
 if len([]rune(_hgretroCharacters))!=256 {panic("invalid mapping length")}
 for r,want:=range map[rune]int64{'ñ':164,'Ñ':165,'á':160,'é':130,'í':161,'ó':162,'ú':163,'ü':129,'¿':168,'¡':173,'"':34,92:92,'?':63} {
  if _hgretroCodes[r]!=want {panic(fmt.Sprintf("bad character mapping %c",r))}
 }
 var layout [][3]int64
 _hgretroLayout("A \tñ\r\n🙂",func(i,x,y int64){layout=append(layout,[3]int64{i,x,y})})
 if fmt.Sprint(layout)!="[[65 0 0] [164 4 0] [63 0 1]]" {panic(fmt.Sprint(layout))}
 // Compare every glyph with the original PNG at both native and enlarged size.
 // Blue behind each cell proves that black source pixels become transparent.
 for atlas:=_hgAtlas(0);atlas<2;atlas++ {
  data,_:=base64.StdEncoding.DecodeString(_hgretroData[atlas])
  original,_,err:=image.Decode(bytes.NewReader(data));if err!=nil{panic(err)}
  for index:=int64(0);index<256;index++ {
   for _,scale:=range []int64{1,4} {
    _hgscreen.Fill(color.NRGBA{0,0,255,255})
    _hgretroGlifo(index,3,5,scale,_hgColor{255,0,0,255},atlas)
    pixels:=make([]byte,48*48*4);_hgscreen.ReadPixels(pixels)
    for y:=0;y<48;y++ {for x:=0;x<48;x++ {
     want:=color.NRGBA{0,0,255,255}
     if x>=3&&y>=5&&x<3+8*int(scale)&&y<5+8*int(scale) {
      sx,sy:=int(index%16)*8+(x-3)/int(scale),int(index/16)*8+(y-5)/int(scale)
      r,_,_,_:=original.At(sx,sy).RGBA();if r!=0{want=color.NRGBA{255,0,0,255}}
     }
     at:=(y*48+x)*4
     if pixels[at]!=want.R||pixels[at+1]!=want.G||pixels[at+2]!=want.B||pixels[at+3]!=want.A{panic(fmt.Sprintf("pixel mismatch atlas %d glyph %d scale %d at %d,%d",atlas,index,scale,x,y))}
    }}
   }
  }
 }
 for _,index:=range []int64{-1,256}{mustPanic(func(){_hgretroGlifo(index,0,0,1,white,0)})}
 for _,scale:=range []int64{0,-1}{mustPanic(func(){_hgretroTexto("",0,0,scale,white)})}
 mustPanic(func(){_hgretroGlifo(0,0,0,1,white,2)})
 screen:=_hgscreen;_hgscreen=nil
 mustPanic(func(){_hgretroTexto("",0,0,1,white)})
 _hgscreen=screen
 fmt.Println("ok")
 return ebiten.Termination
}
func main(){ebiten.SetWindowVisible(false);ebiten.SetRunnableOnUnfocused(true);if err:=ebiten.RunGame(&retroCheck{});err!=nil{panic(err)}}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
