package compiler

import (
	"strings"
	"testing"
)

// Mixing, channels and the 0xRRGGBB form used by the color grids are plain
// computations, so they can be checked from a console program.
func TestColorHelpersRun(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{"main.cometa": `usar std/pincel/color
usar std/pincel/rejilla

fn ver(c color.Color) cadena
	"${c.r},${c.g},${c.b},${c.a}"

tipo Destello
	tinte color.Color = .Blanco

	fn apagado() color.Color @tinte.con_alfa(0)

fn inicio()
	var ambar = color.rgba(255, 176, 0)
	imprimir(ver(ambar))
	imprimir(ver(color.mezclar(.Negro, ambar, 0.5)))
	imprimir(ver(color.mezclar(.Negro, .Blanco, -3)))
	imprimir(ver(color.mezclar(.Negro, .Blanco, 9)))
	imprimir(ver(ambar.con_alfa(128)))
	imprimir(ver(ambar.con_alfa(999)))
	imprimir(ver(ambar))
	imprimir(ver(Destello {}.apagado()))
	imprimir(color.a_entero(ambar) == 0xFFB000)
	imprimir(ver(color.desde_entero(0x00FF80)))
	imprimir(color.a_entero(color.desde_entero(0x123456)) == 0x123456)
	imprimir(color.Color.Rojo.r + ambar.g)
	var colores = rejilla.nueva(2, 1, -1)
	imprimir(colores.poner(0, 0, color.a_entero(.Rojo)))
	imprimir(ver(color.desde_entero(colores.obtener(0, 0) o 0)))
`})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"255,176,0,255", "128,88,0,255", "0,0,0,255", "255,255,255,255",
		"255,176,0,128", "255,176,0,255", "255,176,0,255", "255,255,255,0",
		"verdadero", "0,255,128,255", "verdadero", "431", "verdadero", "255,0,0,255",
	}, "\n") + "\n"
	runGeneratedGo(t, generated, want)

	for _, tc := range []struct{ name, body, want string }{
		{"asignar canal", "\tvar c = color.rgba(1, 2, 3)\n\tc.r = 9\n", "los canales de un color solo se leen"},
		{"canal inexistente", "\tvar c = color.rgba(1, 2, 3)\n\timprimir(c.x)\n", "no existe"},
		{"literal", "\tvar c = color.Color {}\n\timprimir(c.r)\n", "opaco"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte("usar std/pincel/color\n\nfn inicio()\n"+tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}
