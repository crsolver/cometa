package compiler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/crsolver/cometa/internal/ast"
	"github.com/crsolver/cometa/internal/diagnostic"
	"github.com/crsolver/cometa/internal/sema"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// assetUse is one resource constructor of a program, such as
// recursos.imagen("heroe.png"), with its path resolved against the module
// that wrote it.
type assetUse struct {
	call     *ast.CallExpr
	kind     string // catalog name of the constructor: imagen, fuente, sonido...
	path     string // as written in the source
	source   string // file that contains the call
	resolved string
	err      error // the path could not be resolved
}

// assetUses lists the resources of a checked program in source order.
func assetUses(filename string, model *sema.Model) []assetUse {
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
	uses := make([]assetUse, len(calls))
	for i, call := range calls {
		use := assetUse{call: call, kind: model.Game.Calls[call].Name}
		use.path, _ = strconv.Unquote(call.Args[0].(*ast.LiteralExpr).Value)
		use.source = call.Pos.Filename
		if use.source == "" {
			use.source = filename
		}
		use.resolved, use.err = CanonicalPath(filepath.Join(filepath.Dir(use.source), filepath.FromSlash(use.path)))
		uses[i] = use
	}
	return uses
}

func (u assetUse) fail(err error) error {
	return &diagnostic.Error{Stage: diagnostic.Sema, Filename: u.source, Pos: u.call.Args[0].Position(), Message: fmt.Sprintf("recurso %q: %v", u.path, err)}
}

// loadAssets reads, validates and stores the embedded resources of a program
// through loader, so in-memory projects can provide them as well as source
// files. Only code generation needs it: it decodes every image, font and sound.
func loadAssets(filename string, model *sema.Model, loader SourceLoader) error {
	model.Assets = map[*ast.CallExpr]sema.Asset{}
	for _, use := range assetUses(filename, model) {
		err := use.err
		var data []byte
		if err == nil {
			data, err = loader(use.resolved)
		}
		if err == nil {
			err = validateAsset(use.kind, use.path, data)
		}
		if err != nil {
			return use.fail(err)
		}
		model.Assets[use.call] = sema.Asset{Path: use.path, Data: base64.StdEncoding.EncodeToString(data)}
	}
	return nil
}

// AssetCache remembers resource files that were already validated, by size
// and modification time. An editor analyzes a project on every keystroke; with
// the cache each image or sound is decoded once, not once per keystroke.
type AssetCache struct {
	mu      sync.Mutex
	entries map[string]assetEntry
}

type assetEntry struct {
	size    int64
	modTime time.Time
	err     error
}

// CheckAssets reports the first resource of an analyzed project that is
// missing or invalid, without keeping any data. Files on disk are validated
// through cache; anything else (an in-memory project) is validated directly.
func (p *Project) CheckAssets(cache *AssetCache) error {
	if p.Root == nil || p.Model == nil {
		return nil
	}
	for _, use := range assetUses(p.Root.Path, p.Model) {
		if use.err != nil {
			return use.fail(use.err)
		}
		if err := cache.validate(use, p.loader); err != nil {
			return use.fail(err)
		}
	}
	return nil
}

func (c *AssetCache) validate(use assetUse, loader SourceLoader) error {
	info, statErr := os.Stat(use.resolved)
	if statErr == nil {
		c.mu.Lock()
		entry, ok := c.entries[use.kind+"\x00"+use.resolved]
		c.mu.Unlock()
		if ok && entry.size == info.Size() && entry.modTime.Equal(info.ModTime()) {
			return entry.err
		}
	}
	data, err := loader(use.resolved)
	if err == nil {
		err = validateAsset(use.kind, use.path, data)
	}
	if statErr == nil {
		c.mu.Lock()
		if c.entries == nil {
			c.entries = map[string]assetEntry{}
		}
		c.entries[use.kind+"\x00"+use.resolved] = assetEntry{size: info.Size(), modTime: info.ModTime(), err: err}
		c.mu.Unlock()
	}
	return err
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
	case "texto", "cargar_mapa":
		if !utf8.Valid(data) {
			return fmt.Errorf("se requiere texto en UTF-8")
		}
		return nil
	}
	return fmt.Errorf("tipo de recurso desconocido")
}
