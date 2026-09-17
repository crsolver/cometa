package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"strings"
)

type _hgVec2 struct{ X, Y float64 }
type _hgRect struct{ Pos, Tamano _hgVec2 }
type _hgCamara2D struct {
	Pos, Origen    _hgVec2
	Zoom, Rotacion float64
}
type _hgConfigJuego struct {
	Ancho, Alto                        float64
	Titulo                             string
	Escala                             float64
	Redimensionable, Pantalla_completa bool
	Tps                                float64
}
type _hgColor = color.NRGBA
type _hgTecla ebiten.Key
type _hgBotonRaton ebiten.MouseButton
type _hgImagen struct{ image *ebiten.Image }
type _hgFuente struct{ source *text.GoTextFaceSource }
type _hgSonido struct{ pcm []byte }
type _hgReproduccion struct {
	player       *audio.Player
	sound        *_hgSonido
	loop, paused bool
	volume       float64
}
type _hgGame struct{}

var _hgconfig = _hgConfigJuego{320, 180, "Hacha", 1, false, false, 60}
var _hgstarting bool
var _hgscreen *ebiten.Image
var _hgcamara = _hgCamara2D{Zoom: 1}
var _hgaudio *audio.Context
var _hgplayers = map[*_hgReproduccion]bool{}
var _hgwhite *ebiten.Image

func (*_hgGame) Update() error {
	Actualizar(1 / _hgconfig.Tps)
	for r := range _hgplayers {
		if !r.paused && !r.player.IsPlaying() {
			r.player.Close()
			r.player = nil
			delete(_hgplayers, r)
		}
	}
	return nil
}
func (*_hgGame) Draw(screen *ebiten.Image) {
	_hgscreen = screen
	_hgrestablecerCamara()
	defer func() { _hgscreen = nil }()
	Pintar()
}
func (*_hgGame) Layout(w, h int) (int, int) { return int(_hgconfig.Ancho), int(_hgconfig.Alto) }
func _hgconfiguracion(w, h float64, title string, scale float64, resize, full bool, tps float64) {
	if !_hgstarting {
		panic("configuración solo se permite desde iniciar")
	}
	_hgconfig = _hgConfigJuego{w, h, title, scale, resize, full, tps}
}
func _hgvalidarConfig(c _hgConfigJuego) {
	for _, n := range []float64{c.Ancho, c.Alto, c.Escala, c.Tps, c.Ancho * c.Escala, c.Alto * c.Escala} {
		if math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 || n > 32768 {
			log.Fatal("configuración de juego inválida: dimensiones, escala y tps deben ser positivos y finitos (máximo 32768)")
		}
	}
	if c.Ancho != math.Trunc(c.Ancho) || c.Alto != math.Trunc(c.Alto) || c.Tps != math.Trunc(c.Tps) || c.Ancho*c.Escala < 1 || c.Alto*c.Escala < 1 {
		log.Fatal("dimensiones lógicas y tps deben ser enteros")
	}
}
func _hgadd(a, b _hgVec2) _hgVec2           { return _hgVec2{a.X + b.X, a.Y + b.Y} }
func _hgsub(a, b _hgVec2) _hgVec2           { return _hgVec2{a.X - b.X, a.Y - b.Y} }
func _hgscale(a _hgVec2, s float64) _hgVec2 { return _hgVec2{a.X * s, a.Y * s} }
func _hglongitud(v _hgVec2) float64         { return math.Hypot(v.X, v.Y) }
func _hgnormalizar(v _hgVec2) _hgVec2 {
	n := _hglongitud(v)
	if n == 0 {
		return _hgVec2{}
	}
	return _hgscale(v, 1/n)
}
func _hgdistancia(a, b _hgVec2) float64     { return _hglongitud(_hgsub(a, b)) }
func _hgproductoPunto(a, b _hgVec2) float64 { return a.X*b.X + a.Y*b.Y }
func _hgrotar(v _hgVec2, a float64) _hgVec2 {
	s, c := math.Sincos(a)
	return _hgVec2{v.X*c - v.Y*s, v.X*s + v.Y*c}
}
func _hgabs(v float64) float64              { return math.Abs(v) }
func _hgmin(a, b float64) float64           { return math.Min(a, b) }
func _hgmax(a, b float64) float64           { return math.Max(a, b) }
func _hglimitar(v, a, b float64) float64    { return math.Max(a, math.Min(b, v)) }
func _hginterpolar(a, b, t float64) float64 { return a + (b-a)*t }
func _hgpiso(v float64) float64             { return math.Floor(v) }
func _hgtecho(v float64) float64            { return math.Ceil(v) }
func _hgredondear(v float64) float64        { return math.Round(v) }
func _hgraiz(v float64) float64             { return math.Sqrt(v) }
func _hgseno(v float64) float64             { return math.Sin(v) }
func _hgcoseno(v float64) float64           { return math.Cos(v) }
func _hgatan2(y, x float64) float64         { return math.Atan2(y, x) }
func _hgazarReal(a, b float64) float64 {
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) || a > b || math.IsInf(b-a, 0) {
		panic("rango aleatorio inválido")
	}
	return a + rand.Float64()*(b-a)
}
func _hgazarEntero(a, b float64) float64 {
	if a != math.Trunc(a) || b != math.Trunc(b) || b-a > 9007199254740991 {
		panic("rango entero inválido")
	}
	return math.Floor(_hgazarReal(a, b))
}
func _hgrgba(r, g, b, a float64) _hgColor {
	return _hgColor{uint8(_hglimitar(r, 0, 255)), uint8(_hglimitar(g, 0, 255)), uint8(_hglimitar(b, 0, 255)), uint8(_hglimitar(a, 0, 255))}
}
func _hgdraw() {
	if _hgscreen == nil {
		panic("dibujar solo se permite desde pintar")
	}
}
func _hgcameraMatrix() ebiten.GeoM {
	var m ebiten.GeoM
	m.Translate(-_hgcamara.Pos.X, -_hgcamara.Pos.Y)
	m.Rotate(-_hgcamara.Rotacion)
	m.Scale(_hgcamara.Zoom, _hgcamara.Zoom)
	m.Translate(_hgcamara.Origen.X, _hgcamara.Origen.Y)
	return m
}
func _hgpoint(v _hgVec2) _hgVec2 {
	m := _hgcameraMatrix()
	x, y := m.Apply(v.X, v.Y)
	return _hgVec2{x, y}
}
func _hglimpiar(c _hgColor) { _hgdraw(); _hgscreen.Fill(c) }
func _hgrectangulo(p, s _hgVec2, c _hgColor, o _hgVec2, r float64) {
	_hgdraw()
	if _hgcamara.Rotacion == 0 && r == 0 {
		p = _hgpoint(_hgsub(p, o))
		vector.FillRect(_hgscreen, float32(p.X), float32(p.Y), float32(s.X*_hgcamara.Zoom), float32(s.Y*_hgcamara.Zoom), c, false)
		return
	}
	if _hgwhite == nil {
		_hgwhite = ebiten.NewImage(1, 1)
		_hgwhite.Fill(color.White)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM = _hgtransform(p, o, s, r, true)
	op.ColorScale.ScaleWithColor(c)
	_hgscreen.DrawImage(_hgwhite, op)
}
func _hgcirculo(p _hgVec2, r float64, c _hgColor) {
	_hgdraw()
	p = _hgpoint(p)
	vector.FillCircle(_hgscreen, float32(p.X), float32(p.Y), float32(r*_hgcamara.Zoom), c, true)
}
func _hglinea(a, b _hgVec2, c _hgColor, w float64) {
	_hgdraw()
	a = _hgpoint(a)
	b = _hgpoint(b)
	vector.StrokeLine(_hgscreen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), float32(w*_hgcamara.Zoom), c, true)
}
func _hgimageDraw(img *ebiten.Image, p, o, s _hgVec2, r float64, c _hgColor) {
	_hgdraw()
	op := &ebiten.DrawImageOptions{}
	op.GeoM = _hgtransform(p, o, s, r, false)
	op.ColorScale.ScaleWithColor(c)
	_hgscreen.DrawImage(img, op)
}
func _hgimagen(img *_hgImagen, p, o, s _hgVec2, r float64, c _hgColor) {
	_hgimageDraw(img.image, p, o, s, r, c)
}
func _hgregion(img *_hgImagen, region _hgRect, p, o, s _hgVec2, r float64, c _hgColor) {
	area := image.Rect(int(region.Pos.X), int(region.Pos.Y), int(region.Pos.X+region.Tamano.X), int(region.Pos.Y+region.Tamano.Y))
	if area.Empty() {
		return
	}
	_hgimageDraw(img.image.SubImage(area).(*ebiten.Image), p, o, s, r, c)
}
func _hgtexto(value string, font *_hgFuente, p _hgVec2, size float64, c _hgColor, o _hgVec2, r float64) {
	_hgdraw()
	op := &text.DrawOptions{}
	op.GeoM = _hgtransform(p, o, _hgVec2{1, 1}, r, true)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.2
	text.Draw(_hgscreen, value, &text.GoTextFace{Source: font.source, Size: size}, op)
}
func _hgtextoDepuracion(value string, p _hgVec2) {
	_hgdraw()
	ebitenutil.DebugPrintAt(_hgscreen, value, int(p.X), int(p.Y))
}
func _hgtamano() _hgVec2 { return _hgVec2{_hgconfig.Ancho, _hgconfig.Alto} }
func _hgusarCamara(c _hgCamara2D) {
	_hgdraw()
	if c.Zoom <= 0 || math.IsNaN(c.Zoom) || math.IsInf(c.Zoom, 0) {
		panic("zoom debe ser positivo y finito")
	}
	_hgcamara = c
}
func _hgrestablecerCamara()              { _hgcamara = _hgCamara2D{Zoom: 1} }
func _hgteclaMantenida(k _hgTecla) bool  { return ebiten.IsKeyPressed(ebiten.Key(k)) }
func _hgteclaPresionada(k _hgTecla) bool { return inpututil.IsKeyJustPressed(ebiten.Key(k)) }
func _hgteclaSoltada(k _hgTecla) bool    { return inpututil.IsKeyJustReleased(ebiten.Key(k)) }
func _hgratonMantenido(k _hgBotonRaton) bool {
	return ebiten.IsMouseButtonPressed(ebiten.MouseButton(k))
}
func _hgratonPresionado(k _hgBotonRaton) bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(k))
}
func _hgratonSoltado(k _hgBotonRaton) bool {
	return inpututil.IsMouseButtonJustReleased(ebiten.MouseButton(k))
}
func _hgposicionRaton() _hgVec2 {
	x, y := ebiten.CursorPosition()
	return _hgVec2{float64(x), float64(y)}
}
func _hgrueda() _hgVec2 { x, y := ebiten.Wheel(); return _hgVec2{x, y} }
func _hgreproducir(sound *_hgSonido, volume float64, loop bool) *_hgReproduccion {
	r := &_hgReproduccion{sound: sound, loop: loop, volume: _hglimitar(volume, 0, 1)}
	_hgreanudar(r)
	return r
}
func _hgpausar(r *_hgReproduccion) {
	if r.player != nil {
		r.player.Pause()
		r.paused = true
	}
}
func _hgreanudar(r *_hgReproduccion) {
	if _hgaudio == nil {
		_hgaudio = audio.NewContext(48000)
	}
	if r.player == nil {
		var src io.ReadSeeker = bytes.NewReader(r.sound.pcm)
		if r.loop {
			src = audio.NewInfiniteLoop(src, int64(len(r.sound.pcm)))
		}
		p, err := _hgaudio.NewPlayer(src)
		if err != nil {
			log.Fatal(err)
		}
		r.player = p
		r.player.SetVolume(r.volume)
	}
	r.paused = false
	r.player.Play()
	_hgplayers[r] = true
}
func _hgdetener(r *_hgReproduccion) {
	if r.player != nil {
		r.player.Close()
		r.player = nil
	}
	r.paused = false
	delete(_hgplayers, r)
}
func _hgvolumen(r *_hgReproduccion, v float64) {
	r.volume = _hglimitar(v, 0, 1)
	if r.player != nil {
		r.player.SetVolume(r.volume)
	}
}
func _hgrectangulos(a, b _hgRect) bool {
	return a.Pos.X < b.Pos.X+b.Tamano.X && a.Pos.X+a.Tamano.X > b.Pos.X && a.Pos.Y < b.Pos.Y+b.Tamano.Y && a.Pos.Y+a.Tamano.Y > b.Pos.Y
}
func _hgcirculos(a _hgVec2, ra float64, b _hgVec2, rb float64) bool {
	return _hgdistancia(a, b) <= ra+rb
}
func _hgpuntoRectangulo(p _hgVec2, r _hgRect) bool {
	return p.X >= r.Pos.X && p.Y >= r.Pos.Y && p.X < r.Pos.X+r.Tamano.X && p.Y < r.Pos.Y+r.Tamano.Y
}
func _hgtitulo(t string)         { ebiten.SetWindowTitle(t) }
func _hgpantallaCompleta(v bool) { ebiten.SetFullscreen(v) }
func _hgventanaTamano() _hgVec2  { x, y := ebiten.WindowSize(); return _hgVec2{float64(x), float64(y)} }
func _hgfps() float64            { return ebiten.ActualFPS() }
func _hgtps() float64            { return ebiten.ActualTPS() }
func _hgbytes(encoded, path string) []byte {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Fatalf("recurso %s: %v", path, err)
	}
	return b
}
func _hgcargarImagen(encoded, path string) *_hgImagen {
	img, _, err := image.Decode(bytes.NewReader(_hgbytes(encoded, path)))
	if err != nil {
		log.Fatalf("imagen %s: %v", path, err)
	}
	return &_hgImagen{ebiten.NewImageFromImage(img)}
}
func _hgcargarFuente(encoded, path string) *_hgFuente {
	f, err := text.NewGoTextFaceSource(bytes.NewReader(_hgbytes(encoded, path)))
	if err != nil {
		log.Fatalf("fuente %s: %v", path, err)
	}
	return &_hgFuente{f}
}
func _hgcargarSonido(encoded, path string) *_hgSonido {
	b := bytes.NewReader(_hgbytes(encoded, path))
	var stream io.Reader
	var err error
	switch {
	case strings.HasSuffix(strings.ToLower(path), ".wav"):
		stream, err = wav.DecodeWithSampleRate(48000, b)
	case strings.HasSuffix(strings.ToLower(path), ".ogg"):
		stream, err = vorbis.DecodeWithSampleRate(48000, b)
	case strings.HasSuffix(strings.ToLower(path), ".mp3"):
		stream, err = mp3.DecodeWithSampleRate(48000, b)
	default:
		err = fmt.Errorf("formato de sonido desconocido")
	}
	if err != nil {
		log.Fatalf("sonido %s: %v", path, err)
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		log.Fatalf("sonido %s: %v", path, err)
	}
	return &_hgSonido{pcm}
}

func (v _hgVec2) Longitud() float64                   { return _hglongitud(v) }
func (v _hgVec2) Normalizado() _hgVec2                { return _hgnormalizar(v) }
func (v _hgVec2) Distancia_a(otro _hgVec2) float64    { return _hgdistancia(v, otro) }
func (v _hgVec2) Producto_punto(otro _hgVec2) float64 { return _hgproductoPunto(v, otro) }
func (v _hgVec2) Rotado(angulo float64) _hgVec2       { return _hgrotar(v, angulo) }
func (v _hgVec2) Colision_circulo(radio float64, otro _hgVec2, radioOtro float64) bool {
	return _hgcirculos(v, radio, otro, radioOtro)
}
func (r _hgRect) Interseca(otro _hgRect) bool { return _hgrectangulos(r, otro) }
func (r _hgRect) Contiene(punto _hgVec2) bool { return _hgpuntoRectangulo(punto, r) }
func _hgrectanguloXY(x, y, w, h float64, c _hgColor, o _hgVec2, r float64) {
	_hgrectangulo(_hgVec2{x, y}, _hgVec2{w, h}, c, o, r)
}
func _hgrectanguloRect(rect _hgRect, c _hgColor, o _hgVec2, r float64) {
	_hgrectangulo(rect.Pos, rect.Tamano, c, o, r)
}
func _hgcirculoXY(x, y, r float64, c _hgColor) { _hgcirculo(_hgVec2{x, y}, r, c) }
func _hglineaXY(x1, y1, x2, y2 float64, c _hgColor, w float64) {
	_hglinea(_hgVec2{x1, y1}, _hgVec2{x2, y2}, c, w)
}
func _hgimagenXY(img *_hgImagen, x, y float64, o, s _hgVec2, r float64, c _hgColor) {
	_hgimagen(img, _hgVec2{x, y}, o, s, r, c)
}
func _hgregionXY(img *_hgImagen, region _hgRect, x, y float64, o, s _hgVec2, r float64, c _hgColor) {
	_hgregion(img, region, _hgVec2{x, y}, o, s, r, c)
}
func _hgtextoXY(v string, f *_hgFuente, x, y, size float64, c _hgColor, o _hgVec2, r float64) {
	_hgtexto(v, f, _hgVec2{x, y}, size, c, o, r)
}
func _hgtextoDepuracionXY(v string, x, y float64) { _hgtextoDepuracion(v, _hgVec2{x, y}) }

// Destination origins are measured after sizing, before rotation and translation.
func _hgimageRect(img *ebiten.Image, d _hgRect, o _hgVec2, r float64, c _hgColor) {
	_hgdraw()
	bounds := img.Bounds()
	if bounds.Empty() {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM = _hgtransform(d.Pos, o, _hgVec2{d.Tamano.X / float64(bounds.Dx()), d.Tamano.Y / float64(bounds.Dy())}, r, true)
	op.ColorScale.ScaleWithColor(c)
	_hgscreen.DrawImage(img, op)
}
func _hgimagenRect(img *_hgImagen, d _hgRect, o _hgVec2, r float64, c _hgColor) {
	_hgimageRect(img.image, d, o, r, c)
}
func _hgregionRect(img *_hgImagen, source, d _hgRect, o _hgVec2, r float64, c _hgColor) {
	_hgdraw()
	area := image.Rect(int(source.Pos.X), int(source.Pos.Y), int(source.Pos.X+source.Tamano.X), int(source.Pos.Y+source.Tamano.Y))
	if area.Empty() {
		return
	}
	_hgimageRect(img.image.SubImage(area).(*ebiten.Image), d, o, r, c)
}

// Transform local geometry before applying the active camera. Image origins use
// source pixels; rectangle/destination origins use already-sized local pixels.
func _hgtransform(p, o, s _hgVec2, r float64, destinationOrigin bool) ebiten.GeoM {
	var m ebiten.GeoM
	if !destinationOrigin {
		m.Translate(-o.X, -o.Y)
	}
	m.Scale(s.X, s.Y)
	if destinationOrigin {
		m.Translate(-o.X, -o.Y)
	}
	m.Rotate(r)
	m.Translate(p.X, p.Y)
	m.Concat(_hgcameraMatrix())
	return m
}

type Jugador struct {
	Pos _hgVec2
	Vel _hgVec2
}

var Sprite *_hgImagen = func() *_hgImagen {
	var _hacha1 *_hgImagen = _hgcargarImagen("iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAARElEQVR4nGIRCZjGQApgYWBg0MjMJFL1jenTmUgynoGBgTINh91ACL8I7Z3Egsyx3YUujSlC51AaKRoYSU3egAAAAP//8IoLLGPEagkAAAAASUVORK5CYII=", "assets/jugador.png")
	_ = _hacha1
	return _hacha1
}()

var HachaGlobal_736f6e69646f *_hgSonido = func() *_hgSonido {
	var _hacha2 *_hgSonido = _hgcargarSonido("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", "assets/recoger.wav")
	_ = _hacha2
	return _hacha2
}()

var HachaGlobal_6a756761646f72 *Jugador = func() *Jugador {
	var _hacha3 float64 = 80
	_ = _hacha3
	var _hacha4 float64 = 100
	_ = _hacha4
	var _hacha5 _hgVec2 = _hgVec2{X: _hacha3, Y: _hacha4}
	_ = _hacha5
	var _hacha6 *Jugador = &Jugador{Pos: _hacha5, Vel: _hgVec2{}}
	_ = _hacha6
	return _hacha6
}()

var Objetivo _hgRect = func() _hgRect {
	var _hacha7 float64 = 280
	_ = _hacha7
	var _hacha8 float64 = 160
	_ = _hacha8
	var _hacha9 _hgVec2 = _hgVec2{X: _hacha7, Y: _hacha8}
	_ = _hacha9
	var _hacha10 float64 = 20
	_ = _hacha10
	var _hacha11 float64 = 20
	_ = _hacha11
	var _hacha12 _hgVec2 = _hgVec2{X: _hacha10, Y: _hacha11}
	_ = _hacha12
	var _hacha13 _hgRect = _hgRect{Pos: _hacha9, Tamano: _hacha12}
	_ = _hacha13
	return _hacha13
}()

func (_self *Jugador) Dibujar() {
	var _hacha14 *_hgImagen = Sprite
	_ = _hacha14
	var _hacha15 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha15
	var _hacha16 _hgVec2 = _hacha15.Pos
	_ = _hacha16
	var _hacha17 _hgVec2 = _hgVec2{}
	_ = _hacha17
	var _hacha18 float64 = 1
	_ = _hacha18
	var _hacha19 float64 = 1
	_ = _hacha19
	var _hacha20 _hgVec2 = _hgVec2{X: _hacha18, Y: _hacha19}
	_ = _hacha20
	var _hacha21 float64 = 0
	_ = _hacha21
	var _hacha22 _hgColor = _hgColor{255, 255, 255, 255}
	_ = _hacha22
	_hgimagen(_hacha14, _hacha16, _hacha17, _hacha20, _hacha21, _hacha22)
}

func Iniciar() {
	var _hacha23 float64 = 480
	var _hacha24 float64 = 320
	var _hacha25 string = "Hacha: recoge el objetivo"
	var _hacha26 float64 = 2
	var _hacha27 bool = false
	var _hacha28 bool = false
	var _hacha29 float64 = 60
	_hgconfiguracion(_hacha23, _hacha24, _hacha25, _hacha26, _hacha27, _hacha28, _hacha29)
}

func Actualizar(dt float64) {
	var _hacha30 _hgVec2 = _hgVec2{}
	_ = _hacha30
	var direccion _hgVec2 = _hacha30
	_ = direccion
	var _hacha31 _hgTecla = _hgTecla(ebiten.KeyArrowRight)
	_ = _hacha31
	var _hacha32 bool = _hgteclaMantenida(_hacha31)
	_ = _hacha32
	_hacha33 := _hacha32
	if !_hacha33 {
		var _hacha34 _hgTecla = _hgTecla(ebiten.KeyD)
		_ = _hacha34
		var _hacha35 bool = _hgteclaMantenida(_hacha34)
		_ = _hacha35
		_hacha33 = _hacha35
	}
	var _hacha36 bool = _hacha33
	_ = _hacha36
	if _hacha36 {
		_hacha37 := &(direccion.X)
		_ = _hacha37
		var _hacha38 float64 = 1
		_ = _hacha38
		*_hacha37 = _hacha38
	}
	var _hacha39 _hgTecla = _hgTecla(ebiten.KeyArrowLeft)
	_ = _hacha39
	var _hacha40 bool = _hgteclaMantenida(_hacha39)
	_ = _hacha40
	_hacha41 := _hacha40
	if !_hacha41 {
		var _hacha42 _hgTecla = _hgTecla(ebiten.KeyA)
		_ = _hacha42
		var _hacha43 bool = _hgteclaMantenida(_hacha42)
		_ = _hacha43
		_hacha41 = _hacha43
	}
	var _hacha44 bool = _hacha41
	_ = _hacha44
	if _hacha44 {
		_hacha45 := &(direccion.X)
		_ = _hacha45
		var _hacha46 float64 = 1
		_ = _hacha46
		var _hacha47 float64 = (-_hacha46)
		_ = _hacha47
		*_hacha45 = _hacha47
	}
	var _hacha48 _hgTecla = _hgTecla(ebiten.KeyArrowDown)
	_ = _hacha48
	var _hacha49 bool = _hgteclaMantenida(_hacha48)
	_ = _hacha49
	_hacha50 := _hacha49
	if !_hacha50 {
		var _hacha51 _hgTecla = _hgTecla(ebiten.KeyS)
		_ = _hacha51
		var _hacha52 bool = _hgteclaMantenida(_hacha51)
		_ = _hacha52
		_hacha50 = _hacha52
	}
	var _hacha53 bool = _hacha50
	_ = _hacha53
	if _hacha53 {
		_hacha54 := &(direccion.Y)
		_ = _hacha54
		var _hacha55 float64 = 1
		_ = _hacha55
		*_hacha54 = _hacha55
	}
	var _hacha56 _hgTecla = _hgTecla(ebiten.KeyArrowUp)
	_ = _hacha56
	var _hacha57 bool = _hgteclaMantenida(_hacha56)
	_ = _hacha57
	_hacha58 := _hacha57
	if !_hacha58 {
		var _hacha59 _hgTecla = _hgTecla(ebiten.KeyW)
		_ = _hacha59
		var _hacha60 bool = _hgteclaMantenida(_hacha59)
		_ = _hacha60
		_hacha58 = _hacha60
	}
	var _hacha61 bool = _hacha58
	_ = _hacha61
	if _hacha61 {
		_hacha62 := &(direccion.Y)
		_ = _hacha62
		var _hacha63 float64 = 1
		_ = _hacha63
		var _hacha64 float64 = (-_hacha63)
		_ = _hacha64
		*_hacha62 = _hacha64
	}
	var _hacha66 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha66
	_hacha65 := &(_hacha66.Vel)
	_ = _hacha65
	var _hacha67 _hgVec2 = direccion
	_ = _hacha67
	_hacha68 := _hacha67.Normalizado
	_ = _hacha68
	var _hacha69 _hgVec2 = _hacha68()
	_ = _hacha69
	var _hacha70 float64 = 140
	_ = _hacha70
	var _hacha71 _hgVec2 = _hgscale(_hacha69, _hacha70)
	_ = _hacha71
	*_hacha65 = _hacha71
	var _hacha73 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha73
	_hacha72 := &(_hacha73.Pos)
	_ = _hacha72
	var _hacha74 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha74
	var _hacha75 _hgVec2 = _hacha74.Pos
	_ = _hacha75
	var _hacha76 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha76
	var _hacha77 _hgVec2 = _hacha76.Vel
	_ = _hacha77
	var _hacha78 float64 = dt
	_ = _hacha78
	var _hacha79 _hgVec2 = _hgscale(_hacha77, _hacha78)
	_ = _hacha79
	var _hacha80 _hgVec2 = _hgadd(_hacha75, _hacha79)
	_ = _hacha80
	*_hacha72 = _hacha80
	var _hacha82 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha82
	_hacha81 := &(_hacha82.Pos.X)
	_ = _hacha81
	var _hacha83 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha83
	var _hacha84 _hgVec2 = _hacha83.Pos
	_ = _hacha84
	var _hacha85 float64 = _hacha84.X
	_ = _hacha85
	var _hacha86 float64 = 0
	_ = _hacha86
	var _hacha87 float64 = 464
	_ = _hacha87
	var _hacha88 float64 = _hglimitar(_hacha85, _hacha86, _hacha87)
	_ = _hacha88
	*_hacha81 = _hacha88
	var _hacha90 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha90
	_hacha89 := &(_hacha90.Pos.Y)
	_ = _hacha89
	var _hacha91 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha91
	var _hacha92 _hgVec2 = _hacha91.Pos
	_ = _hacha92
	var _hacha93 float64 = _hacha92.Y
	_ = _hacha93
	var _hacha94 float64 = 40
	_ = _hacha94
	var _hacha95 float64 = 304
	_ = _hacha95
	var _hacha96 float64 = _hglimitar(_hacha93, _hacha94, _hacha95)
	_ = _hacha96
	*_hacha89 = _hacha96
	var _hacha97 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha97
	var _hacha98 _hgVec2 = _hacha97.Pos
	_ = _hacha98
	var _hacha99 float64 = 16
	_ = _hacha99
	var _hacha100 float64 = 16
	_ = _hacha100
	var _hacha101 _hgVec2 = _hgVec2{X: _hacha99, Y: _hacha100}
	_ = _hacha101
	var _hacha102 _hgRect = _hgRect{Pos: _hacha98, Tamano: _hacha101}
	_ = _hacha102
	_hacha103 := _hacha102.Interseca
	_ = _hacha103
	var _hacha104 _hgRect = Objetivo
	_ = _hacha104
	var _hacha105 bool = _hacha103(_hacha104)
	_ = _hacha105
	if _hacha105 {
		var _hacha106 *_hgSonido = HachaGlobal_736f6e69646f
		_ = _hacha106
		var _hacha107 float64 = 0.3
		_ = _hacha107
		var _hacha108 bool = false
		_ = _hacha108
		var _hacha109 *_hgReproduccion = _hgreproducir(_hacha106, _hacha107, _hacha108)
		_ = _hacha109
		_ = _hacha109
		_hacha110 := &(Objetivo.Pos)
		_ = _hacha110
		var _hacha111 float64 = 20
		_ = _hacha111
		var _hacha112 float64 = 440
		_ = _hacha112
		var _hacha113 float64 = _hgazarEntero(_hacha111, _hacha112)
		_ = _hacha113
		var _hacha114 float64 = 60
		_ = _hacha114
		var _hacha115 float64 = 280
		_ = _hacha115
		var _hacha116 float64 = _hgazarEntero(_hacha114, _hacha115)
		_ = _hacha116
		var _hacha117 _hgVec2 = _hgVec2{X: _hacha113, Y: _hacha116}
		_ = _hacha117
		*_hacha110 = _hacha117
	}
}

func Pintar() {
	var _hacha118 _hgColor = _hgColor{0, 0, 0, 255}
	_ = _hacha118
	_hglimpiar(_hacha118)
	var _hacha119 *Jugador = HachaGlobal_6a756761646f72
	_ = _hacha119
	_hacha120 := _hacha119.Dibujar
	_ = _hacha120
	_hacha120()
	var _hacha121 _hgRect = Objetivo
	_ = _hacha121
	var _hacha122 _hgColor = _hgColor{255, 255, 0, 255}
	_ = _hacha122
	var _hacha123 _hgVec2 = _hgVec2{}
	_ = _hacha123
	var _hacha124 float64 = 0
	_ = _hacha124
	_hgrectanguloRect(_hacha121, _hacha122, _hacha123, _hacha124)
	var _hacha125 string = "WASD / flechas: mover. Recoge el objetivo amarillo."
	_ = _hacha125
	var _hacha126 float64 = 8
	_ = _hacha126
	var _hacha127 float64 = 12
	_ = _hacha127
	var _hacha128 _hgVec2 = _hgVec2{X: _hacha126, Y: _hacha127}
	_ = _hacha128
	_hgtextoDepuracion(_hacha125, _hacha128)
}

func main() {
	_hgstarting = true
	Iniciar()
	_hgstarting = false
	_hgvalidarConfig(_hgconfig)
	ebiten.SetWindowSize(int(_hgconfig.Ancho*_hgconfig.Escala), int(_hgconfig.Alto*_hgconfig.Escala))
	ebiten.SetWindowTitle(_hgconfig.Titulo)
	ebiten.SetFullscreen(_hgconfig.Pantalla_completa)
	if _hgconfig.Redimensionable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	ebiten.SetTPS(int(_hgconfig.Tps))
	if err := ebiten.RunGame(&_hgGame{}); err != nil {
		log.Fatal(err)
	}
}
