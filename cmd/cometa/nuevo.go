package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var projectName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_-]*$`)

const consoleTemplate = `// Tu primer programa en Cometa.
// Ejecútalo con: cometa ejecutar principal.cometa
// Las sangrías se escriben con tabuladores.

fn saludar(nombre cadena) cadena
	"¡Hola, ${nombre}!"

fn inicio()
	imprimir(saludar("mundo"))
`

const gameTemplate = `// Tu primer juego con Pincel: mueve el cuadrado con las flechas o WASD.
// Ejecútalo con: cometa ejecutar principal.cometa
usar std/mate
usar std/pincel
usar std/pincel/graficos
usar std/pincel/entrada

const ancho entero = 320
const alto entero = 180

tipo Partida
	pos mate.Vec2

	// actualizar se llama unas 60 veces por segundo; dt son los segundos transcurridos.
	pub fn actualizar(dt decimal)
		var direccion = mate.Vec2 {}
		si entrada.tecla_mantenida(.Derecha) || entrada.tecla_mantenida(.D)
			direccion.x = 1
		si entrada.tecla_mantenida(.Izquierda) || entrada.tecla_mantenida(.A)
			direccion.x = -1
		si entrada.tecla_mantenida(.Abajo) || entrada.tecla_mantenida(.S)
			direccion.y = 1
		si entrada.tecla_mantenida(.Arriba) || entrada.tecla_mantenida(.W)
			direccion.y = -1
		@pos = @pos + direccion.normalizado() * 100 * dt
		@pos.x = mate.limitar(@pos.x, 0, ancho - 16)
		@pos.y = mate.limitar(@pos.y, 0, alto - 16)

	// pintar dibuja el cuadro actual.
	pub fn pintar()
		graficos.limpiar(.Negro)
		graficos.rectangulo_v(@pos, {16, 16}, .Amarillo)

fn inicio()
	var partida = Partida {pos: {152, 82}}
	pincel.ejecutar(partida, ancho, alto, titulo = "Mi juego", escala = 3) capturar |error|
		imprimir(error)
`

// newProject creates a directory with a runnable starter program.
func newProject(name string, game bool) error {
	if !projectName.MatchString(name) {
		return fmt.Errorf("el nombre %q no es válido: usa letras, números, «_» o «-»", name)
	}
	if entries, err := os.ReadDir(name); err == nil && len(entries) > 0 {
		return fmt.Errorf("la carpeta %q ya existe y no está vacía", name)
	}
	if err := os.MkdirAll(name, 0o755); err != nil {
		return fmt.Errorf("no se pudo crear la carpeta %q: %w", name, err)
	}
	template := consoleTemplate
	if game {
		template = gameTemplate
	}
	file := filepath.Join(name, "principal.cometa")
	if err := os.WriteFile(file, []byte(template), 0o644); err != nil {
		return fmt.Errorf("no se pudo escribir %s: %w", file, err)
	}
	fmt.Printf("Proyecto creado en %s\n\nSiguiente paso:\n  cd %s\n  cometa ejecutar principal.cometa\n", name, name)
	if game {
		fmt.Println("\nLa primera vez se descargan y compilan las dependencias del juego; puede tardar unos minutos.")
	}
	return nil
}

// runNew parses `cometa nuevo <nombre> [--juego]`.
func runNew(args []string) error {
	var name string
	game := false
	for _, arg := range args {
		switch {
		case arg == "--juego":
			game = true
		case len(arg) > 0 && arg[0] == '-':
			return fmt.Errorf("argumento inesperado %q\nuso: cometa nuevo <nombre> [--juego]", arg)
		case name == "":
			name = arg
		default:
			return fmt.Errorf("argumento inesperado %q\nuso: cometa nuevo <nombre> [--juego]", arg)
		}
	}
	if name == "" {
		return fmt.Errorf("uso: cometa nuevo <nombre> [--juego]")
	}
	return newProject(name, game)
}
