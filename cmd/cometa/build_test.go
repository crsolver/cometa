package main

import (
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildGameExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop executable build")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "game.cometa")
	output := filepath.Join(dir, "game")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	if err := os.WriteFile(input, []byte("usar std/pincel\nusar std/pincel/graficos\ntipo Partida\n\tpub fn actualizar(dt decimal) imprimir(dt)\n\tpub fn pintar() graficos.limpiar(.Negro)\nfn inicio()\n\tpincel.ejecutar(Partida {}) atrapar |e| imprimir(e)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"construir", input, "-o", output}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		t.Fatalf("missing executable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); !os.IsNotExist(err) {
		t.Fatal("build modified source project")
	}
}

func TestGameCLIArguments(t *testing.T) {
	for _, args := range [][]string{{"ejecutar"}, {"construir"}, {"ejecutar", "x.cometa", "-o", "x.exe"}, {"construir", "x.txt"}, {"construir", "x.cometa", "--unknown"}, {"compilar", "x.cometa", "--escala", "2"}, {"captura", "x.cometa", "--cuadros", "0"}, {"captura", "x.cometa", "--escala", "65"}, {"captura", "x.cometa", "--escala"}, {"captura", "x.cometa", "--cuadros", "10,x"}, {"captura", "x.cometa", "--cuadros", "10,-1"}, {"captura", "x.cometa", "--entrada"}, {"ejecutar", "x.cometa", "--entrada", "g.txt"}} {
		if err := run(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	if frames, err := parseFrames("900,100,300,100"); err != nil || len(frames) != 3 || frames[0] != 100 || frames[2] != 900 {
		t.Errorf("parseFrames = %v, %v", frames, err)
	}
	if paths := capturePaths("a/shot.png", []int{5, 30}); paths[0] != "a/shot_5.png" || paths[1] != "a/shot_30.png" {
		t.Errorf("capturePaths = %v", paths)
	}
}

func TestReadInputScript(t *testing.T) {
	dir := t.TempDir()
	write := func(text string) string {
		path := filepath.Join(dir, "guion.txt")
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	events, err := readScript(write("# prueba\n90 -D\n60 +D +Espacio  # dos teclas\n\n100 Enter raton 1.5 2 RatonIzquierdo\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []scriptEvent{
		{frame: 60, kind: 0, code: "int(ebiten.KeyD)"},
		{frame: 60, kind: 0, code: "int(ebiten.KeySpace)"},
		{frame: 90, kind: 1, code: "int(ebiten.KeyD)"},
		{frame: 100, kind: 2, code: "int(ebiten.KeyEnter)"},
		{frame: 100, kind: 3, x: 1.5, y: 2},
		{frame: 100, kind: 2, code: "-1-int(ebiten.MouseButtonLeft)"},
	}
	if len(events) != len(want) {
		t.Fatalf("events = %v", events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("event %d = %v, want %v", i, events[i], want[i])
		}
	}
	for text, message := range map[string]string{
		"10 Entr\n":     `guion.txt:1: tecla desconocida "Entr"; ¿quisiste decir "Enter"?`,
		"\n10\n":        "guion.txt:2: falta una acción",
		"D 10\n":        "guion.txt:1: cada línea empieza con el número de cuadro",
		"10 raton 4\n":  "«raton» necesita dos números",
		"# solo nota\n": "no tiene acciones",
	} {
		if _, err := readScript(write(text)); err == nil || !strings.Contains(err.Error(), message) {
			t.Errorf("readScript(%q) error = %v, want %q", text, err, message)
		}
	}
}

// A scripted run reaches later scenes without changing the game: held keys,
// one-tick taps, the pointer and several shots in a single run.
func TestCaptureWithInputScript(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "game.cometa")
	script := filepath.Join(dir, "guion.txt")
	game := `usar std/pincel
usar std/pincel/graficos
usar std/pincel/entrada
tipo Partida
	x entero
	fin bool
	marca decimal = -1
	pub fn actualizar(dt decimal)
		si entrada.tecla_mantenida(.D)
			@x += 1
		si entrada.tecla_presionada(.Enter)
			@fin = !@fin
		si entrada.raton_presionado(.Izquierdo)
			@marca = entrada.posicion_raton().x
	pub fn pintar()
		si @fin
			graficos.limpiar(.Azul)
		sino
			graficos.limpiar(.Negro)
		graficos.rectangulo(@x, 0, 1, 1, .Rojo)
		si @marca >= 0
			graficos.rectangulo(@marca, 0, 1, 1, .Verde)
fn inicio()
	pincel.ejecutar(Partida {}, 4, 1) atrapar |e| imprimir(e)
`
	if err := os.WriteFile(input, []byte(game), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("2 +D\n4 -D\n5 Enter\n6 raton 3 0 RatonIzquierdo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "shot.png")
	if err := run([]string{"captura", input, "-o", output, "--cuadros", "7,1", "--entrada", script}); err != nil {
		t.Fatal(err)
	}
	pixels := func(path string) []string {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		img, err := png.Decode(file)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for x := 0; x < 4; x++ {
			r, g, b, _ := img.At(x, 0).RGBA()
			switch {
			case r>>8 > 200 && g>>8 < 100:
				out = append(out, "rojo")
			case g>>8 > 100 && r>>8 < 100 && b>>8 < 150:
				out = append(out, "verde")
			case b>>8 > 100:
				out = append(out, "azul")
			default:
				out = append(out, "negro")
			}
		}
		return out
	}
	if got := strings.Join(pixels(filepath.Join(dir, "shot_1.png")), " "); got != "rojo negro negro negro" {
		t.Errorf("tick 1 = %s", got)
	}
	if got := strings.Join(pixels(filepath.Join(dir, "shot_7.png")), " "); got != "azul azul rojo verde" {
		t.Errorf("tick 7 = %s", got)
	}
}

func TestCaptureGameScreen(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "game.cometa")
	output := filepath.Join(dir, "shot.png")
	game := "usar std/pincel\nusar std/pincel/graficos\nusar std/pincel/lienzo\nvar punto = lienzo.nuevo(1, 1, .Rojo)\ntipo Partida\n\tpub fn actualizar(dt decimal) retornar\n\tpub fn pintar()\n\t\tgraficos.limpiar(.Negro)\n\t\tgraficos.imagen(punto, 1, 0)\nfn inicio()\n\tpincel.ejecutar(Partida {}, 3, 2) atrapar |e| imprimir(e)\n"
	if err := os.WriteFile(input, []byte(game), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"captura", input, "-o", output, "--escala", "2", "--cuadros", "3"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 6 || img.Bounds().Dy() != 4 {
		t.Fatalf("capture bounds = %v", img.Bounds())
	}
	for _, point := range [][3]int{{0, 0, 0}, {2, 0, 255}, {3, 1, 255}, {4, 3, 0}} {
		if r, _, _, _ := img.At(point[0], point[1]).RGBA(); int(r>>8) != point[2] {
			t.Fatalf("pixel %v red = %d", point, r>>8)
		}
	}
	plain := filepath.Join(dir, "plain.cometa")
	if err := os.WriteFile(plain, []byte("fn inicio() imprimir(1)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"captura", plain}); err == nil || !strings.Contains(err.Error(), "pincel.ejecutar") {
		t.Fatalf("captured a program without a game: %v", err)
	}
}
