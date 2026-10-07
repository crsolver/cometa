//go:build cometa_runtime_deps

package stdlib

// The bundled runtime (the *_runtime.txt files) is Go source that only ever
// compiles inside generated programs, so nothing in this module imports its
// dependencies. These blank imports record them for `go mod tidy`, which reads
// every build tag: tests run generated games against this module's go.mod.
// TestRuntimeDependenciesAreRecorded keeps the list equal to runtimeImports.
import (
	_ "github.com/hajimehoshi/ebiten/v2"
	_ "github.com/hajimehoshi/ebiten/v2/audio"
	_ "github.com/hajimehoshi/ebiten/v2/audio/mp3"
	_ "github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	_ "github.com/hajimehoshi/ebiten/v2/audio/wav"
	_ "github.com/hajimehoshi/ebiten/v2/colorm"
	_ "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	_ "github.com/hajimehoshi/ebiten/v2/inpututil"
	_ "github.com/hajimehoshi/ebiten/v2/text/v2"
	_ "github.com/hajimehoshi/ebiten/v2/vector"
	_ "golang.org/x/image/font/gofont/goregular"
)
