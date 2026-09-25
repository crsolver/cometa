package stdlib

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRetroAtlasesAndIcons(t *testing.T) {
	for _, data := range [][]byte{dungeonAtlas, asciiAtlas} {
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 128 {
			t.Fatal("atlas must contain 16 by 16 cells of 8 pixels")
		}
	}
	for name, want := range map[string]string{
		"Corazon": "_hgIcono(3)", "CorazonVacio": "_hgIcono(19)",
		"Espada": "_hgIcono(156)", "Escudo": "_hgIcono(157)",
		"Llave": "_hgIcono(175)", "Calavera": "_hgIcono(237)",
		"Derecha": "_hgIcono(12)", "Izquierda": "_hgIcono(13)",
		"Arriba": "_hgIcono(14)", "Abajo": "_hgIcono(15)",
	} {
		if got := Constants[Symbol("Icono")][name]; got != want {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
}
