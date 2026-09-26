package compiler

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHuertoExampleCompiles(t *testing.T) {
	if _, err := CompileProject(filepath.Join("testdata", "programas", "huerto.cometa"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestHuertoSimulation(t *testing.T) {
	path := filepath.Join("testdata", "programas", "huerto", "granja.cometa")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := Compile(path, source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("simulation should not require a graphics context")
	}
	harness := `
func require(ok bool, message string) { if !ok { panic(message) } }
func idx(x, y int64) int64 { return y*Columnas + x }
func find(m *Granja, kind int64) int64 {
 for i, c := range m.Celdas { if c == kind { return int64(i) } }
 return -1
}
func main() {
 for _, seed := range []int64{1, 2, 42, 2026, 999999} {
  a := &Granja{}; a.Generar(seed); b := &Granja{}; b.Generar(seed)
  require(len(a.Celdas) == int(Columnas*Filas), "map size")
  for i := range a.Celdas { require(a.Celdas[i] == b.Celdas[i], "seed not deterministic") }
  require(!a.Choca(a.Jugador), "spawn obstructed")
  require(a.Celda(8, 6) == Puerta && a.Celda(10, 8) == Caja && a.Celda(12, 11) == Puesto, "landmarks")
  require(a.Celda(-1, 3) == Bosque && a.Solido(40, 0), "outside the map is solid")
  // Every landmark and most of the field are reachable on foot from the spawn.
  start := idx(8, 8); seen := map[int64]bool{start: true}; queue := []int64{start}
  for h := 0; h < len(queue); h++ {
   i := queue[h]
   for _, d := range []int64{-1, 1, -Columnas, Columnas} {
    j := i + d
    if j >= 0 && j < int64(len(a.Celdas)) && !seen[j] && !a.Solido(j%Columnas, j/Columnas) { seen[j] = true; queue = append(queue, j) }
   }
  }
  require(seen[idx(8, 7)] && seen[idx(10, 9)] && seen[idx(12, 12)], "door, bin or stand unreachable")
  soil, shore := 0, false
  for i, c := range a.Celdas {
   x, y := int64(i)%Columnas, int64(i)/Columnas
   if x > 14 && x < 33 && y > 14 && y < 25 { require(c == Tierra || c == Roca, "field interior blocked") }
   if c == Tierra && seen[int64(i)] { soil++ }
   if c == Agua { for _, d := range []int64{-1, 1, -Columnas, Columnas} { if seen[int64(i)+d] { shore = true } } }
  }
  require(soil > 120, "field unreachable")
  require(shore, "pond unreachable")
 }

 m := &Granja{}; m.Generar(42)
 require(m.Frente() == idx(8, 9), "facing down")
 // Walking into the house stops at the wall, in front of the door.
 for i := 0; i < 120; i++ { m.Mover(_hgVec2{0, -1}, 1.0/60) }
 require(m.Mirando == 1 && m.Jugador.Y >= 117 && !m.Choca(m.Jugador), "walked through the house")
 require(m.Frente() == idx(8, 6) && m.Aplicar(Azada, m.Frente()) == Dormir, "door does not offer sleep")
 m.Mover(_hgVec2{}, 1.0/60); require(!m.Andando, "idle")

 c := find(m, Tierra)
 require(m.Aplicar(Regadera, c) == Nada && m.Aplicar(Semillas_nabo, c) == Nada, "untilled soil accepted water or seeds")
 require(m.Aplicar(Azada, c) == Labrado && m.Celdas[c] == Labrada, "tilling")
 require(m.Aplicar(Semillas_nabo+1, c) == Sin_semillas, "planting without stock")
 require(m.Aplicar(Semillas_nabo, c) == Plantado && m.Semillas[0] == 5, "planting")
 require(m.Aplicar(Semillas_nabo, c) == Nada, "double planting")
 require(m.Aplicar(Hoz, c) == Nada && len(m.Cultivos) == 1, "harvested an unripe crop")
 // A dry night does not grow the crop (day 2 never rains).
 m.Dormir(); require(m.Dia == 2 && !m.Lluvia && m.Cultivos[c].Dias == 0, "unwatered crop grew")
 for day := int64(0); day < Duracion(0); day++ {
  require(!m.Cultivos[c].Listo(), "ripe too early")
  if !m.Mojada[c] { require(m.Aplicar(Regadera, c) == Regado && m.Mojada[c], "watering") }
  require(m.Aplicar(Regadera, c) == Nada, "watering twice")
  m.Dormir()
 }
 require(m.Cultivos[c].Listo(), "watered crop did not ripen")
 // Any tool harvests a ripe crop; the soil stays tilled.
 require(m.Aplicar(Azada, c) == Cosechado && m.Cosecha[0] == 1 && len(m.Cultivos) == 0 && m.Celdas[c] == Labrada, "harvest")
 money := m.Dinero
 require(m.Aplicar(Hoz, idx(10, 8)) == Vendido && m.Dinero == money+Venta(0) && m.Ultima_venta == Venta(0) && m.Cosecha[0] == 0, "selling")
 require(m.Vender() == Nada_que_vender && m.Dinero == money+Venta(0), "selling nothing")
 // The stand sells the selected seed kind, or turnips with other tools.
 m.Dinero = Precio(1) - 1
 require(m.Aplicar(Semillas_nabo+1, idx(12, 11)) == Sin_dinero && m.Semillas[1] == 0, "bought without money")
 m.Dinero = Precio(1)
 require(m.Aplicar(Semillas_nabo+1, idx(12, 11)) == Comprado && m.Dinero == 0 && m.Semillas[1] == 1, "buying carrots")
 m.Dinero = Precio(0)
 require(m.Aplicar(Regadera, idx(12, 11)) == Comprado && m.Semillas[0] == 6, "buying turnips")

 // Water runs out and refills at the pond.
 m.Aplicar(Azada, find(m, Tierra)); d := find(m, Labrada); pond := find(m, Agua)
 m.Agua = 0; m.Mojada[d] = false; require(m.Aplicar(Regadera, d) == Sin_agua && !m.Mojada[d], "watered without water")
 require(m.Aplicar(Regadera, pond) == Recargado && m.Agua == Agua_maxima, "refill")
 require(m.Aplicar(Regadera, pond) == Nada, "refilling a full can")
 require(m.Aplicar(Semillas_nabo, pond) == Nada && m.Aplicar(Azada, idx(0, 0)) == Nada && m.Aplicar(Azada, -1) == Nada, "tools on solid cells")
 if r := find(m, Roca); r >= 0 {
  require(m.Aplicar(Azada, r) == Roca_rota && m.Celdas[r] == Tierra, "breaking rocks")
 }
 require(len(m.Pendientes) > 0, "edits not queued for repaint")

 // Stages advance monotonically up to ripeness.
 for clase := int64(0); clase < 3; clase++ {
  last := int64(0)
  for dias := int64(0); dias <= Duracion(clase); dias++ {
   e := (&Cultivo{Clase: clase, Dias: dias}).Etapa()
   require(e >= last && e <= 3 && (e == 3) == (dias >= Duracion(clase)), "stage order")
   last = e
  }
 }

 // Rain waters tilled soil and makes crops grow without the can.
 m.Pendientes = nil
 rainy := 0
 for i := 0; i < 40; i++ {
  m.Dormir()
  if m.Lluvia { rainy++; require(m.Mojada[d], "rain did not water tilled soil") } else { require(!m.Mojada[d], "soil stayed wet") }
 }
 require(rainy > 3 && rainy < 30, "rain frequency")
 require(m.Agua == Agua_maxima && m.Hora == 6, "morning reset")

 // The clock runs out at 02:00.
 m.Hora = 6; require(!m.Avanzar(1), "exhausted too soon")
 m.Hora = Hora_limite - 0.01; require(m.Avanzar(Segundos_por_hora), "never exhausted")
}
`
	runGeneratedGo(t, append(generated, []byte(harness)...), "")
}

// Run the real game for a few frames in a hidden window through the capture
// hook: sprite texts, the baked ground and every draw function execute.
func TestHuertoRenderingRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	generated, err := CompileProject(filepath.Join("testdata", "programas", "huerto.cometa"), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "huerto.png")
	source := string(generated) + fmt.Sprintf("\nfunc init() { _hgcapturaRuta = %q; _hgcapturaCuadros = 20; _hgcapturaEscala = 1 }\n", out)
	runGeneratedGo(t, []byte(source), "")
	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if size := img.Bounds().Size(); size.X != 320 || size.Y != 180 {
		t.Fatalf("capture size = %v", size)
	}
	unique := map[[3]uint32]bool{}
	for y := 0; y < 180; y++ {
		for x := 0; x < 320; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			unique[[3]uint32{r, g, b}] = true
		}
	}
	if len(unique) < 40 {
		t.Fatalf("scene lacks detail: %d colors", len(unique))
	}
}
