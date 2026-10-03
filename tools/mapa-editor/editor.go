// Package mapaeditor carries the map editor page so `cometa mapa` can open it
// without the repository at hand.
package mapaeditor

import _ "embed"

//go:embed index.html
var HTML []byte
