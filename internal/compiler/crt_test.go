package compiler

import (
	"strings"
	"testing"
)

func TestRetroStartupArguments(t *testing.T) {
	for _, args := range []string{
		"g",
		"g, retro = verdadero",
		"g, retro = falso, pixelado = verdadero",
		"g, pixelado = falso, retro = verdadero",
		`g, 320, 180, "Retro", 4, falso, falso, 60, verdadero`,
		`g, 320, 180, "Retro", 4, falso, falso, 60, falso, verdadero`,
	} {
		source := "usar std/pincel\nfn iniciar(g pincel.Juego)\n\tpincel.ejecutar(" + args + ") capturar |e| imprimir(e)\n"
		if _, err := Compile("crt.cometa", []byte(source)); err != nil {
			t.Fatalf("%s: %v", args, err)
		}
	}
	for _, value := range []string{"1", `"verdadero"`} {
		source := "usar std/pincel\nfn iniciar(g pincel.Juego)\n\tpincel.ejecutar(g, retro = " + value + ") capturar |e| imprimir(e)\n"
		if _, err := Compile("crt.cometa", []byte(source)); err == nil {
			t.Fatalf("accepted retro = %s", value)
		}
	}
	generated, err := Compile("math.cometa", []byte("usar std/mate\nfn inicio() imprimir(mate.Vec2 {1, 2})\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "_hgCRT") || strings.Contains(string(generated), "ebiten") {
		t.Fatal("CRT dependencies leaked into math-only program")
	}
}

func TestCRTPresentationRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	generated, err := Compile("crt.cometa", []byte("usar std/pincel\nusar std/pincel/retro\n"))
	if err != nil {
		t.Fatal(err)
	}
	// juego's capture runtime already imports os and image/png by name.
	source := string(generated) + `
type crtCheck struct{}
func (*crtCheck) Layout(w,h int)(int,int){return 64,48}
func (*crtCheck) Draw(screen *ebiten.Image){}
func readCRT(img *ebiten.Image) []byte {
 b:=make([]byte,img.Bounds().Dx()*img.Bounds().Dy()*4);img.ReadPixels(b);return b
}
func (*crtCheck) Update() error {
 shader,err:=ebiten.NewShader([]byte(_hgCRTSource));if err!=nil{panic(err)}
 defer shader.Dispose()
 wrapper:=&_hgCRTGame{shader:shader}
 raw:=ebiten.NewImage(64,48);defer raw.Dispose()
 raw.Fill(color.White)
 raw.SubImage(image.Rect(8,8,24,24)).(*ebiten.Image).Fill(color.NRGBA{255,0,0,255})
 raw.SubImage(image.Rect(24,8,40,24)).(*ebiten.Image).Fill(color.NRGBA{0,255,0,255})
 raw.SubImage(image.Rect(40,8,56,24)).(*ebiten.Image).Fill(color.NRGBA{0,0,255,255})
 _hgscreen=raw
 _hgretroTexto("¡Niño!",8,32,1,_hgColor{0,0,0,255})
 _hgscreen=nil
 original:=readCRT(raw)
 preview:=image.NewRGBA(image.Rect(0,0,800,800))
 row:=0
 for _,scale:=range []float64{1,2,4,2.5,6} {
  w,h:=int(64*scale)+20,int(48*scale)+20
  normal,effect:=ebiten.NewImage(w,h),ebiten.NewImage(w,h)
  var m ebiten.GeoM;m.Scale(scale,scale);m.Translate(10,10)
  ebiten.SetScreenFilterEnabled(false)
  ebiten.DefaultDrawFinalScreen(normal,raw,m)
  wrapper.DrawFinalScreen(effect,raw,m)
  before,after:=readCRT(normal),readCRT(effect)
  changed:=false
  for i:=0;i<len(before);i+=4 {
   if before[i+3]!=after[i+3]{panic("CRT shifted geometry or letterboxing")}
   for c:=0;c<3;c++ {
    a,b:=int(before[i+c]),int(after[i+c])
    if b>a+1 || b<int(float64(a)*0.74)-1{panic("CRT exceeds subtle darkening limits")}
    if a==0 && b!=0 {panic("CRT blurred source pixels")}
    changed=changed||a!=b
   }
   if scale==1 && before[i]==255 && before[i+1]==255 && before[i+2]==255 {
    if after[i]!=after[i+1] || after[i]!=after[i+2]{panic("phosphor mask did not fade at 1x")}
   }
  }
  if !changed{panic("CRT did not affect presentation")}
  if !bytes.Equal(original,readCRT(raw)){panic("CRT modified logical screen")}
  // The disabled presentation still follows the existing Ebitengine filter.
  for _,pixelated:=range []bool{false,true} {
   normal.Clear();effect.Clear();ebiten.SetScreenFilterEnabled(!pixelated)
   ebiten.DefaultDrawFinalScreen(normal,raw,m)
   op:=&ebiten.DrawImageOptions{GeoM:m}
   if !pixelated && math.Floor(scale)!=scale {op.Filter=ebiten.FilterPixelated}
   effect.DrawImage(raw,op)
   a,b:=readCRT(normal),readCRT(effect)
   for i:=range a {if math.Abs(float64(a[i])-float64(b[i]))>1 {panic(fmt.Sprintf("disabled presentation differs at scale %v, pixelated %v, byte %d: %d vs %d",scale,pixelated,i,a[i],b[i]))}}
  }
  normal.Clear();effect.Clear();ebiten.SetScreenFilterEnabled(false)
  ebiten.DefaultDrawFinalScreen(normal,raw,m);wrapper.DrawFinalScreen(effect,raw,m)
  for y:=0;y<h;y++ {for x:=0;x<w;x++ {
   preview.Set(x,row+y,normal.At(x,y));preview.Set(400+x,row+y,effect.At(x,y))
  }}
  row+=h
  normal.Dispose();effect.Dispose()
 }
 if path:=os.Getenv("COMETA_CRT_PREVIEW");path!="" {
  f,err:=os.Create(path);if err!=nil{panic(err)}
  if err:=png.Encode(f,preview);err!=nil{panic(err)};if err:=f.Close();err!=nil{panic(err)}
 }
 fmt.Println("ok")
 return ebiten.Termination
}
func main(){ebiten.SetWindowVisible(false);ebiten.SetRunnableOnUnfocused(true);if err:=ebiten.RunGame(&crtCheck{});err!=nil{panic(err)}}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
