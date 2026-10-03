package stdlib

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The map editor (tools/mapa-editor) is a single HTML file that carries its own
// copy of the dungeon atlas, the icon names and the color palette. This test
// keeps that copy equal to the library; run it with COMETA_ACTUALIZAR_EDITOR=1
// to rewrite the data lines after changing the atlas, an icon or a color.
func TestMapEditorData(t *testing.T) {
	const path = "../../tools/mapa-editor/index.html"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	number := regexp.MustCompile(`\d+`)
	icons := map[string]int{}
	for name, value := range Constants[Symbol("Icono")] {
		icons[name], _ = strconv.Atoi(number.FindString(value))
	}
	colors := map[string][]int{}
	for name, value := range Constants[Symbol("Color")] {
		var channels []int
		for _, s := range number.FindAllString(value, -1) {
			n, _ := strconv.Atoi(s)
			channels = append(channels, n)
		}
		if len(channels) == 4 && channels[3] == 255 {
			colors[name] = channels[:3]
		}
	}
	encode := func(v any) string {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	want := map[string]string{
		"ATLAS":   strconv.Quote("data:image/png;base64," + base64.StdEncoding.EncodeToString(dungeonAtlas)),
		"ICONOS":  encode(icons),
		"COLORES": encode(colors),
	}
	lines := strings.Split(string(source), "\n")
	found := map[string]bool{}
	changed := false
	for i, line := range lines {
		for name, value := range want {
			prefix := "const " + name + " = "
			if !strings.HasPrefix(line, prefix) {
				continue
			}
			found[name] = true
			if expected := prefix + value + ";"; strings.TrimRight(line, "\r") != expected {
				changed = true
				lines[i] = expected
			}
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("falta la línea «const %s = …;» en %s", name, path)
		}
	}
	if !changed || t.Failed() {
		return
	}
	if os.Getenv("COMETA_ACTUALIZAR_EDITOR") == "" {
		t.Fatalf("%s no coincide con internal/stdlib; ejecuta la prueba con COMETA_ACTUALIZAR_EDITOR=1", path)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}
