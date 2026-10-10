package rt

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"log"
	"strings"
)

// DUNGEON.mode by datagoblin (CC0). Original 8x8 atlas artwork.
type _hgIcono int64
type _hgAtlas int64

var _hgretroImages [2]*_hgImagen
var _hgretroData = [2]string{"DUNGEON_ATLAS_DATA", "ASCII_ATLAS_DATA"}

// Each row corresponds to exactly sixteen cells in dungeon-437.png.
const _hgretroCharacters = " ☺☻♥♦♣♠•◘○◙♂♀♪♫☼" +
	"►◄↕‼¶§▬↨↑↓→←∟↔▲▼" +
	"…!\"#$%&'()*+,-./" +
	"0123456789:;<=>?" +
	"@ABCDEFGHIJKLMNO" +
	"PQRSTUVWXYZ[\\]^_" +
	"`abcdefghijklmno" +
	"pqrstuvwxyz{|}~⌂" +
	"ÇüéâäàåçêëèïîìÄÅ" +
	"ÉæÆôöòûùÿÖÜ¢£¥₧ƒ" +
	"áíóúñÑªº¿⌐¬½¼¡«»" +
	"░▒▓│┤╡╢╖╕╣║╗╝╜╛┐" +
	"└┴┬├─┼╞╟╚╔╩╦╠═╬╧" +
	"╨╤╥╙╘╒╓╫╪┘┌█▄▌▐▀" +
	"αßΓπΣσµτΦΘΩδ∞φε∩" +
	"≡±≥≤⌠⌡÷≈°∙·√ⁿ²■□"

var _hgretroCodes = func() map[rune]int64 {
	codes := map[rune]int64{}
	for i, r := range []rune(_hgretroCharacters) {
		codes[r] = int64(i)
	}
	// The font has no glyph for these accented letters: draw the plain letter instead of "?".
	for base, accented := range map[rune]string{'A': "ÁÀÂÃ", 'E': "ÈÊË", 'I': "ÍÌÎÏ", 'O': "ÓÒÔÕ", 'U': "ÚÙÛ", 'Y': "Ý", 'a': "ã", 'o': "õ", 'y': "ý"} {
		for _, r := range accented {
			if _, ok := codes[r]; !ok {
				codes[r] = codes[base]
			}
		}
	}
	return codes
}()

func _hgretroImage(atlas _hgAtlas) *ebiten.Image { return _hgretroImagen(atlas).gpu() }

// The mask is kept as CPU pixels and uploaded on first draw, so an atlas can be requested before the game starts.
func _hgretroImagen(atlas _hgAtlas) *_hgImagen {
	if atlas < 0 || atlas > 1 {
		panic("atlas retro inválido")
	}
	if _hgretroImages[atlas] == nil {
		data, err := base64.StdEncoding.DecodeString(_hgretroData[atlas])
		if err != nil {
			panic(err)
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			panic(err)
		}
		mask := image.NewNRGBA(img.Bounds())
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, _, _, _ := img.At(x, y).RGBA()
				mask.SetNRGBA(x, y, color.NRGBA{255, 255, 255, uint8(r >> 8)})
			}
		}
		_hgretroImages[atlas] = &_hgImagen{pixels: mask}
	}
	return _hgretroImages[atlas]
}

// The atlas as a sheet of 8x8 frames: frame i is the glyph that retro.glifo(i) draws.
var _hgretroHojas [2]*_hgHoja

func _hgretroHoja(atlas _hgAtlas) *_hgHoja {
	img := _hgretroImagen(atlas)
	if _hgretroHojas[atlas] == nil {
		_hgretroHojas[atlas] = _hghoja(img, 8, 8)
	}
	return _hgretroHojas[atlas]
}

// A map saved by tools/mapa-editor: layers bottom to top, each with its icons and (optionally) one color per cell.
type _hgMapa struct {
	nombres        []string
	capas, colores []*_hgRejilla
}

func _hgretroCargarMapa(encoded, path string) (m *_hgMapa) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("mapa %s: %v", path, r)
		}
	}()
	texto := string(_hgbytes(encoded, path))
	m = &_hgMapa{}
	tiene := map[string]bool{}
	for _, linea := range strings.Split(texto, "\n") {
		if linea = strings.TrimSpace(strings.TrimPrefix(linea, "\xEF\xBB\xBF")); strings.HasPrefix(linea, "#") {
			tiene[strings.TrimSpace(linea[1:])] = true
		}
	}
	for _, linea := range strings.Split(texto, "\n") {
		linea = strings.TrimSpace(strings.TrimPrefix(linea, "\xEF\xBB\xBF"))
		if !strings.HasPrefix(linea, "#") {
			continue
		}
		nombre := strings.TrimSpace(linea[1:])
		if base := strings.TrimSuffix(nombre, ".color"); base != nombre && tiene[base] {
			continue
		}
		m.nombres = append(m.nombres, nombre)
		m.capas = append(m.capas, _hgrejillaDesdeCsv(texto, nombre))
		var colores *_hgRejilla
		if tiene[nombre+".color"] {
			colores = _hgrejillaDesdeCsv(texto, nombre+".color")
		}
		m.colores = append(m.colores, colores)
	}
	if len(m.capas) == 0 {
		m.nombres = []string{"mapa"}
		m.capas = []*_hgRejilla{_hgrejillaDesdeCsv(texto, "")}
		m.colores = []*_hgRejilla{nil}
	}
	return m
}
func (m *_hgMapa) Columnas() int64 { return m.capas[0].cols }
func (m *_hgMapa) Filas() int64    { return m.capas[0].rows }
func (m *_hgMapa) Capas() *[]string {
	nombres := append([]string{}, m.nombres...)
	return &nombres
}
func (m *_hgMapa) indice(nombre string) int {
	for i, n := range m.nombres {
		if n == nombre {
			return i
		}
	}
	panic(fmt.Sprintf("el mapa no tiene una capa %q (capas: %s)", nombre, strings.Join(m.nombres, ", ")))
}
func (m *_hgMapa) Capa(nombre string) *_hgRejilla { return m.capas[m.indice(nombre)] }

// A layer saved without colors gets an empty color grid on first request, so cells can still be recolored.
func (m *_hgMapa) Colores(nombre string) *_hgRejilla {
	i := m.indice(nombre)
	if m.colores[i] == nil {
		m.colores[i] = _hgrejillaNueva(m.capas[i].cols, m.capas[i].rows, -1)
	}
	return m.colores[i]
}
func _hgretroMapa(m *_hgMapa, x, y, scale int64, capa string, tinte _hgColor) {
	_hgdraw()
	_hgretroScale(scale)
	solo := -1
	if capa != "" {
		solo = m.indice(capa)
	}
	for i, g := range m.capas {
		if solo >= 0 && i != solo {
			continue
		}
		var colores struct {
			tag      int
			payload1 *_hgRejilla
		}
		if m.colores[i] != nil {
			colores.tag = 1
			colores.payload1 = m.colores[i]
		}
		_hgrejillaDibujar(g, _hgretroHoja(0), _hgVec2{float64(x), float64(y)}, _hgVec2{float64(scale), float64(scale)}, tinte, nil, 0, colores)
	}
}
func _hgretroScale(scale int64) {
	if scale <= 0 {
		panic("escala retro debe ser positiva")
	}
}
func _hgretroGlifo(index, x, y, scale int64, c _hgColor, atlas _hgAtlas, espejoH, espejoV bool) {
	_hgdraw()
	_hgretroScale(scale)
	if index < 0 || index > 255 {
		panic("índice de glifo retro debe estar entre 0 y 255")
	}
	img := _hgretroImage(atlas)
	sx, sy := int(index%16)*8, int(index/16)*8
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	// A mirrored glyph is scaled by -1 around the far edge of its 8x8 cell, so it stays in place.
	o, s := _hgVec2{}, _hgVec2{float64(scale), float64(scale)}
	if espejoH {
		s.X = -s.X
		o.X = 8
	}
	if espejoV {
		s.Y = -s.Y
		o.Y = 8
	}
	op.GeoM = _hgtransform(_hgVec2{float64(x), float64(y)}, o, s, 0, false)
	op.ColorScale.ScaleWithColor(c)
	_hgscreen.DrawImage(img.SubImage(image.Rect(sx, sy, sx+8, sy+8)).(*ebiten.Image), op)
}
func _hgretroIcono(icon _hgIcono, x, y, scale int64, c _hgColor, espejoH, espejoV bool) {
	_hgretroGlifo(int64(icon), x, y, scale, c, 0, espejoH, espejoV)
}
func _hgretroLayout(value string, draw func(index, col, row int64)) {
	var col, row int64
	for _, r := range value {
		switch r {
		case '\r':
			continue
		case '\n':
			col = 0
			row++
			continue
		case '\t':
			col = (col/4 + 1) * 4
			continue
		case ' ':
			col++
			continue
		}
		index, ok := _hgretroCodes[r]
		if !ok {
			index = 63
		}
		draw(index, col, row)
		col++
	}
}
func _hgretroTexto(value string, x, y, scale int64, c _hgColor) {
	_hgdraw()
	_hgretroScale(scale)
	_hgretroLayout(value, func(index, col, row int64) {
		_hgretroGlifo(index, x+col*8*scale, y+row*8*scale, scale, c, 1, false, false)
	})
}

// Width of the widest line up to its last visible glyph, so text can be centered.
func _hgretroAnchoTexto(value string, scale int64) int64 {
	_hgretroScale(scale)
	widest := int64(0)
	_hgretroLayout(value, func(index, col, row int64) {
		if col+1 > widest {
			widest = col + 1
		}
	})
	return widest * 8 * scale
}

// Defaults a fantasy-console-sized window (320x200, escala 4, pixelado) to fit the 8x8 dungeon font cleanly.
func _hgretroEjecutar(instancia _hgJuego, w, h int64, title string, scale float64, resize, full bool, tps int64, pixelado, retro bool) struct {
	tag      int
	payload2 string
} {
	return _hgejecutar(instancia, w, h, title, scale, resize, full, tps, pixelado, retro)
}
