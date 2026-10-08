package compiler

import "testing"

// The editor analyzes on every pause in typing and `cometa ejecutar` compiles
// before each run, so both costs are worth watching:
//
//	go test ./internal/compiler -run '^$' -bench . -benchmem

const benchConsole = "fn inicio()\n\timprimir(1)\n"

const benchGame = `usar std/pincel
usar std/pincel/graficos
usar std/mate

tipo Juego
	pos mate.Vec2

	pub fn actualizar(dt decimal)
		@pos = @pos + mate.Vec2 {x: 1, y: 0} * dt

	pub fn pintar()
		graficos.limpiar(.Negro)
		graficos.rectangulo(@pos.x, @pos.y, 8, 8, .Rojo)
		graficos.circulo(1, 2, 3, .Azul)
		graficos.linea(1, 2, 3, 4, .Verde)

fn inicio()
	pincel.ejecutar(Juego {}) atrapar |e| imprimir(e)
`

func BenchmarkAnalyzeConsole(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := AnalyzeSource("a.cometa", []byte(benchConsole)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAnalyzeGame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := AnalyzeSource("a.cometa", []byte(benchGame)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileGame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Compile("a.cometa", []byte(benchGame)); err != nil {
			b.Fatal(err)
		}
	}
}
