package compiler

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLienzoAPI(t *testing.T) {
	got, err := Compile("lienzo.cometa", []byte("usar std/pincel/lienzo\nvar s = lienzo.desde_texto([\"a.\"], [\"a\": .Rojo])\nfn inicio()\n\tlienzo.pixel(s, 1, 0, .Azul)\n\timprimir(lienzo.ancho(s))\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"_hglienzoDesdeTexto", "_hglienzoPixel", "func (i *_hgImagen) gpu()"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(string(got), "_hgretro") || strings.Contains(string(got), "func _hgejecutar") {
		t.Fatal("lienzo pulled unrelated runtime")
	}
	for _, source := range []string{
		"fn inicio() lienzo.nuevo(4, 4)\n",
		"usar std/pincel/lienzo\nfn inicio() imprimir(lienzo.ancho(lienzo.nuevo(4.5, 4)))\n",
		"usar std/pincel/lienzo\nfn inicio() imprimir(lienzo.ancho(lienzo.desde_texto(\"ab\", [\"a\": .Rojo])))\n",
		"usar std/pincel/lienzo\nfn inicio() imprimir(lienzo.ancho(lienzo.desde_texto([\"ab\"], [1: .Rojo])))\n",
		"usar std/pincel/lienzo\nfn inicio() lienzo.guardar(lienzo.nuevo(1, 1), \"x.png\")\n",
	} {
		if _, err := Compile("invalid.cometa", []byte(source)); err == nil {
			t.Fatal("accepted invalid lienzo call", source)
		}
	}
	if _, err := CompileProject(filepath.Join("testdata", "programas", "pixelart.cometa"), nil); err != nil {
		t.Fatal(err)
	}
}

// Canvas programs run without a window; results are compared pixel by pixel.
func TestLienzoRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("generated program")
	}
	dir := filepath.ToSlash(t.TempDir())
	source := `usar std/pincel/lienzo
usar std/pincel/color

fn guardar(img graficos.Imagen, nombre cadena, escala entero = 1)
	lienzo.guardar(img, "` + dir + `/" + nombre, escala = escala) atrapar |e| imprimir(e)

fn inicio()
	var marco = lienzo.nuevo(7, 7)
	lienzo.rect(marco, 0, 0, 7, 7, .Rojo, relleno = falso)
	lienzo.linea(marco, 1, 1, 5, 5, .Azul)
	lienzo.rellenar(marco, 5, 1, .Verde)
	guardar(marco, "marco.png")

	var disco = lienzo.nuevo(7, 7)
	lienzo.circulo(disco, 3, 3, 3, .Blanco, relleno = falso)
	guardar(disco, "circulo.png")

	var par = lienzo.desde_texto(["ab"], ["a": .Rojo, "b": .Azul])
	var espejo = lienzo.nuevo(2, 1, .Negro)
	lienzo.pegar(espejo, par, 0, 0, espejo_h = verdadero)
	guardar(espejo, "espejo.png", escala = 2)

	var fondo = lienzo.nuevo(1, 1, color.rgba(0, 0, 255))
	lienzo.pegar(fondo, lienzo.desde_texto(["x"], ["x": color.rgba(255, 0, 0, 128)]), 0, 0)
	lienzo.pegar(fondo, lienzo.desde_texto(["."], [:]), 0, 0)
	guardar(fondo, "mezcla.png")

	var hoja = lienzo.hoja_desde_texto([["a."], [".a"]], ["a": .Verde])
	imprimir(hoja.longitud())
	imprimir(lienzo.leer_pixel(hoja[1], 1, 0) o .Negro)
	si lienzo.leer_pixel(hoja[1], 2, 0) |c|
		imprimir(c)
	sino
		imprimir("fuera")
	var copia = lienzo.copiar(par)
	lienzo.limpiar(copia)
	imprimir(lienzo.leer_pixel(par, 0, 0) o .Negro)
	imprimir(lienzo.leer_pixel(copia, 0, 0) o .Negro)
`
	generated, err := Compile("lienzo.cometa", []byte("usar std/pincel/graficos\n"+source))
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "2\nNRGBA {r: 0, g: 255, b: 0, a: 255}\nfuera\nNRGBA {r: 255, g: 0, b: 0, a: 255}\nNRGBA {r: 0, g: 0, b: 0, a: 0}\n")
	palette := map[rune]color.NRGBA{'.': {}, 'R': {255, 0, 0, 255}, 'B': {0, 0, 255, 255}, 'G': {0, 255, 0, 255}, 'W': {255, 255, 255, 255}, 'm': {128, 0, 127, 255}}
	expectPNG(t, dir+"/marco.png", palette, "RRRRRRR", "RBGGGGR", "R.BGGGR", "R..BGGR", "R...BGR", "R....BR", "RRRRRRR")
	expectPNG(t, dir+"/circulo.png", palette, "..WWW..", ".W...W.", "W.....W", "W.....W", "W.....W", ".W...W.", "..WWW..")
	expectPNG(t, dir+"/espejo.png", palette, "BBRR", "BBRR")
	expectPNG(t, dir+"/mezcla.png", palette, "m")
}

func expectPNG(t *testing.T, path string, palette map[rune]color.NRGBA, rows ...string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, len(rows[0]), len(rows)) {
		t.Fatalf("%s bounds = %v", path, img.Bounds())
	}
	for y, row := range rows {
		for x, r := range row {
			got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if want := palette[r]; got != want && !(got.A == 0 && want.A == 0) {
				t.Fatalf("%s (%d,%d) = %v, want %v", filepath.Base(path), x, y, got, want)
			}
		}
	}
}

func TestLienzoRuntimeErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("generated program")
	}
	for source, want := range map[string]string{
		"lienzo.desde_texto([\"ab\", \"a\"], [\"a\": .Rojo, \"b\": .Azul])": "la fila 2 mide 1 caracteres",
		"lienzo.desde_texto([\"ax\"], [\"a\": .Rojo])":                     "el carácter 'x' de la fila 1 no tiene color",
		"lienzo.desde_texto([\"a\"], [\"ab\": .Rojo])":                     "debe ser un solo carácter",
		"lienzo.hoja_desde_texto([[\"a\"], [\"aa\"]], [\"a\": .Rojo])":     "el cuadro 2 no mide lo mismo",
		"lienzo.nuevo(0, 4)":                                               "el tamaño de lienzo",
	} {
		generated, err := Compile("error.cometa", []byte("usar std/pincel/lienzo\nfn inicio()\n\tvar img = "+source+"\n\timprimir(img)\n"))
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		file := filepath.Join(dir, "main.go")
		if err := os.WriteFile(file, generated, 0600); err != nil {
			t.Fatal(err)
		}
		goName := "go"
		if runtime.GOOS == "windows" {
			goName += ".exe"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		out, err := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", goName), "run", file).CombinedOutput()
		cancel()
		if err == nil || !strings.Contains(string(out), want) {
			t.Fatalf("%s: err=%v, output lacks %q:\n%s", source, err, want, out)
		}
	}
}

// Edited canvases must re-upload before their next draw.
func TestLienzoGPUUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	generated, err := Compile("lienzo.cometa", []byte("usar std/pincel/lienzo\nusar std/pincel/graficos\nfn pintar() graficos.imagen(lienzo.nuevo(1, 1), 0, 0)\n"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated) + `
type lienzoCheck struct{}
func (*lienzoCheck) Layout(w,h int)(int,int){return 4,4}
func (*lienzoCheck) Draw(screen *ebiten.Image){}
func (*lienzoCheck) Update() error {
 _hgscreen=ebiten.NewImage(4,4)
 white:=_hgColor{255,255,255,255}
 img:=_hglienzoNuevo(2,2,_hgColor{255,0,0,255})
 _hgimagenXY(img,0,0,_hgVec2{},_hgVec2{1,1},0,white,_hgColor{})
 if r,_,b,_:=_hgscreen.At(1,1).RGBA();r>>8!=255||b!=0 {panic("initial upload missing")}
 _hglienzoPixel(img,1,1,_hgColor{0,0,255,128})
 _hgscreen.Clear();_hgimagenXY(img,0,0,_hgVec2{},_hgVec2{1,1},0,white,_hgColor{})
 if r,_,b,a:=_hgscreen.At(1,1).RGBA();r!=0||b>>8!=128||a>>8!=128 {panic(fmt.Sprint("edit not uploaded ",r,b,a))}
 if r,_,_,_:=_hgscreen.At(0,0).RGBA();r>>8!=255 {panic("unedited pixel changed")}
 fmt.Println("ok")
 return ebiten.Termination
}
func main(){ebiten.SetWindowVisible(false);ebiten.SetRunnableOnUnfocused(true);if err:=ebiten.RunGame(&lienzoCheck{});err!=nil{panic(err)}}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
