package compiler

import (
	"strings"
	"testing"
)

func TestPositionalStructLiteralsRuntime(t *testing.T) {
	runHacha(t, `tipo Base
	n num
tipo Punto
	Base
	x num
	y num
tipo Caja<T>
	valor T
tipo Par
	izquierda Punto
	derecha Punto
fn valor(n num) num
	imprimir(n)
	n
fn inicio()
	var parcial = Punto {Base {7}, 2}
	imprimir(parcial.n)
	imprimir(parcial.x)
	imprimir(parcial.y)
	var contextual Punto = {{8}, 3, 4}
	imprimir(contextual.n)
	var caja = Caja<num> {9}
	imprimir(caja.valor)
	var par = Par {{Base {1}, 2, 3}, {Base {4}, 5, 6}}
	imprimir(par.derecha.y)
	var orden = Punto {Base {valor(1)}, valor(2), valor(3)}
	imprimir(orden.y)
`, "7\n2\n0\n8\n9\n6\n1\n2\n3\n3\n")
}

func TestPositionalBuiltInStructLiterals(t *testing.T) {
	source := `fn actualizar(dt num)
	var v = Vec2 {1, 2}
	imprimir(v.x + v.y + dt)
fn pintar()
	var r = Rect {{3, 4}, {5, 6}}
	graficos.rectangulo_rect(r, .Rojo)
`
	generated, err := Compile("posicionales_juego.hacha", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"_hgVec2{X:", "Y:", "_hgRect{Pos:", "Tamano:"} {
		if !strings.Contains(string(generated), want) {
			t.Fatalf("generated Go does not contain %q:\n%s", want, generated)
		}
	}
}

func TestPositionalStructLiteralDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{"mixed after positional", "tipo P\n\tx num\n\ty num\nfn inicio()\n\tvar p = P {1, y: 2}\n", "no se pueden mezclar"},
		{"mixed after named", "tipo P\n\tx num\n\ty num\nfn inicio()\n\tvar p = P {x: 1, 2}\n", "no se pueden mezclar"},
		{"too many", "tipo P\n\tx num\nfn inicio()\n\tvar p = P {1, 2}\n", "más valores posicionales"},
		{"wrong type", "tipo P\n\tx num\nfn inicio()\n\tvar p = P {\"x\"}\n", "campo \"x\" debe ser num"},
		{"cannot infer", "fn inicio()\n\tvar p = {1}\n", "no se puede inferir"},
		{"opaque", "fn inicio()\n\tvar imagen = Imagen {1}\n", "tipo opaco"},
		{"required trailing", "tipo P\n\tx num\n\testado num!\nfn inicio()\n\tvar p = P {1}\n", "campo estado requiere inicialización"},
		{"generic required empty", "tipo Caja<T>\n\tvalor T\nfn inicio()\n\tvar c = Caja<num> {}\n", "campo valor requiere inicialización"},
		{"unknown named unchanged", "tipo P\n\tx num\nfn inicio()\n\tvar p = P {y: 1}\n", "campo \"y\" no existe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("posicionales.hacha", []byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}
