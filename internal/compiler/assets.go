package compiler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/sema"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// loadAssets reads embedded resources through loader, so in-memory projects
// can provide them as well as source files.
func loadAssets(filename string, model *sema.Model, loader SourceLoader) error {
	model.Assets = map[*ast.CallExpr]sema.Asset{}
	var calls []*ast.CallExpr
	for call, f := range model.Game.Calls {
		if f.Resource {
			calls = append(calls, call)
		}
	}
	sort.Slice(calls, func(i, j int) bool {
		a, b := calls[i].Pos, calls[j].Pos
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
	for _, call := range calls {
		f := model.Game.Calls[call]
		path, _ := strconv.Unquote(call.Args[0].(*ast.LiteralExpr).Value)
		source := call.Pos.Filename
		if source == "" {
			source = filename
		}
		resolved, err := CanonicalPath(filepath.Join(filepath.Dir(source), filepath.FromSlash(path)))
		var data []byte
		if err == nil {
			data, err = loader(resolved)
		}
		if err == nil {
			err = validateAsset(f.Name, path, data)
		}
		if err != nil {
			return &sema.Error{Filename: source, Pos: call.Args[0].Position(), Message: fmt.Sprintf("recurso %q: %v", path, err)}
		}
		model.Assets[call] = sema.Asset{Path: path, Data: base64.StdEncoding.EncodeToString(data)}
	}
	return nil
}

func validateAsset(kind, path string, data []byte) error {
	ext := strings.ToLower(filepath.Ext(path))
	r := bytes.NewReader(data)
	switch kind {
	case "imagen":
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			return fmt.Errorf("se requiere PNG o JPEG")
		}
		_, _, err := image.Decode(r)
		return err
	case "fuente":
		if ext != ".ttf" && ext != ".otf" {
			return fmt.Errorf("se requiere TTF u OTF")
		}
		l, err := opentype.NewLoader(r)
		if err != nil {
			return err
		}
		_, err = font.NewFont(l)
		return err
	case "sonido":
		var stream io.Reader
		var err error
		switch ext {
		case ".wav":
			stream, err = wav.DecodeWithSampleRate(48000, r)
		case ".ogg":
			stream, err = vorbis.DecodeWithSampleRate(48000, r)
		case ".mp3":
			stream, err = mp3.DecodeWithSampleRate(48000, r)
		default:
			return fmt.Errorf("se requiere WAV, Ogg o MP3")
		}
		if err != nil {
			return err
		}
		n, err := io.Copy(io.Discard, stream)
		if err == nil && n == 0 {
			return fmt.Errorf("sonido vacío")
		}
		return err
	}
	return fmt.Errorf("tipo de recurso desconocido")
}
