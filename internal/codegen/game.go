package codegen

import (
	_ "embed"
	"fmt"
	"hacha/internal/ast"
	"hacha/internal/gameapi"
	"strconv"
	"strings"
)

// Preserve public names when possible, but allow the common Jugador/jugador pair.
func (g *generator) globalName(name string) string {
	candidate := exported(name)
	occupied := map[string]bool{}
	for n := range g.model.Types {
		occupied[exported(n)] = true
	}
	for n := range g.model.Enums {
		occupied[exported(n)] = true
	}
	for n := range g.model.Interfaces {
		occupied[exported(n)] = true
	}
	for n := range g.model.Functions {
		occupied[exported(n)] = true
	}
	for n := range g.model.Globals {
		if n != name {
			occupied[exported(n)] = true
		}
	}
	if !occupied[candidate] {
		return candidate
	}
	candidate = fmt.Sprintf("HachaGlobal_%x", []byte(name))
	for occupied[candidate] {
		candidate += "_"
	}
	return candidate
}

//go:embed game_runtime.txt
var gameRuntime string

const gameImports = `import (
 "bytes"
 "encoding/base64"
 "fmt"
 "image"
 "image/color"
 _ "image/jpeg"
 _ "image/png"
 "io"
 "log"
 "math"
 "math/rand/v2"
 "strings"
 "github.com/hajimehoshi/ebiten/v2"
 "github.com/hajimehoshi/ebiten/v2/audio"
 "github.com/hajimehoshi/ebiten/v2/audio/mp3"
 "github.com/hajimehoshi/ebiten/v2/audio/vorbis"
 "github.com/hajimehoshi/ebiten/v2/audio/wav"
 "github.com/hajimehoshi/ebiten/v2/ebitenutil"
 "github.com/hajimehoshi/ebiten/v2/inpututil"
 "github.com/hajimehoshi/ebiten/v2/text/v2"
 "github.com/hajimehoshi/ebiten/v2/vector"
)`

func (g *generator) gameCall(call *ast.CallExpr, f gameapi.Function, indent int) string {
	if f.Resource {
		asset := g.model.Assets[call]
		return "_hg" + f.GoName + "(" + strconv.Quote(asset.Data) + "," + strconv.Quote(asset.Path) + ")"
	}
	info := g.model.Calls[call]
	args := make([]string, len(info.Signature.Params))
	for i, arg := range call.Args {
		args[info.Parameters[i]] = g.flowExpr(arg, indent)
	}
	for i, p := range info.Signature.Decl.Params {
		if args[i] == "" {
			args[i] = g.flowExpr(p.Default, indent)
		}
	}
	return "_hg" + f.GoName + "(" + strings.Join(args, ",") + ")"
}

func (g *generator) emitGameEntry() {
	g.line(0, "func main() {")
	g.line(1, "_hgstarting = true")
	if g.model.Functions["iniciar"].Decl != nil {
		g.line(1, "Iniciar()")
	}
	g.line(1, "_hgstarting = false")
	g.line(1, "_hgvalidarConfig(_hgconfig)")
	g.line(1, "ebiten.SetWindowSize(int(_hgconfig.Ancho * _hgconfig.Escala), int(_hgconfig.Alto * _hgconfig.Escala))")
	g.line(1, "ebiten.SetWindowTitle(_hgconfig.Titulo)")
	g.line(1, "ebiten.SetFullscreen(_hgconfig.Pantalla_completa)")
	g.line(1, "if _hgconfig.Redimensionable { ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled) }")
	g.line(1, "ebiten.SetTPS(int(_hgconfig.Tps))")
	g.line(1, "if err := ebiten.RunGame(&_hgGame{}); err != nil { log.Fatal(err) }")
	g.line(0, "}")
}
