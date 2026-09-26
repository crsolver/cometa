//go:build ignore

// Generates the original public-domain example assets. Run from the repo root.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	must(os.MkdirAll("examples/pincel/assets", 0755))
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := color.NRGBA{60, 185, 255, 255}
			if x == 0 || x == 15 || y == 0 || y == 15 {
				c = color.NRGBA{20, 80, 150, 255}
			}
			if y >= 4 && y <= 6 && (x == 4 || x == 5 || x == 10 || x == 11) {
				c = color.NRGBA{255, 255, 255, 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var p bytes.Buffer
	must(png.Encode(&p, img))
	must(os.WriteFile("examples/pincel/assets/jugador.png", p.Bytes(), 0644))
	const samples = 4800
	var wave bytes.Buffer
	wave.WriteString("RIFF")
	must(binary.Write(&wave, binary.LittleEndian, uint32(36+samples*2)))
	wave.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(48000), uint32(96000), uint16(2), uint16(16)} {
		must(binary.Write(&wave, binary.LittleEndian, v))
	}
	wave.WriteString("data")
	must(binary.Write(&wave, binary.LittleEndian, uint32(samples*2)))
	for i := 0; i < samples; i++ {
		t := float64(i) / 48000
		v := math.Sin(2*math.Pi*(660*t+2200*t*t)) * (1 - float64(i)/samples)
		must(binary.Write(&wave, binary.LittleEndian, int16(v*12000)))
	}
	must(os.WriteFile("examples/pincel/assets/recoger.wav", wave.Bytes(), 0644))
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
