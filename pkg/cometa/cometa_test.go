package cometa_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/crsolver/cometa/pkg/cometa"
)

func project(files map[string]string) map[string][]byte {
	out := map[string][]byte{}
	for name, source := range files {
		out[name] = []byte(source)
	}
	return out
}

func diagnostics(t *testing.T, err error) []cometa.Diagnostic {
	t.Helper()
	var e *cometa.Error
	if !errors.As(err, &e) || len(e.Diagnostics) == 0 {
		t.Fatalf("expected *cometa.Error with diagnostics, got %T %v", err, err)
	}
	return e.Diagnostics
}

func TestCompileMemoryProject(t *testing.T) {
	files := project(map[string]string{
		"principal.cometa":  "usar mundo/mapa\nfn inicio()\n\timprimir(mapa.ancho())\n",
		"mundo/mapa.cometa": "pub fn ancho() entero 16\n",
	})
	if err := cometa.Check("principal.cometa", cometa.Options{Files: files}); err != nil {
		t.Fatal(err)
	}
	generated, err := cometa.Compile("principal.cometa", cometa.Options{Files: files})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(generated, []byte("package main")) && !bytes.Contains(generated, []byte("\npackage main")) {
		t.Fatalf("not a Go main package:\n%s", generated)
	}
}

func TestMemoryDiagnosticsUseCallerKeys(t *testing.T) {
	files := project(map[string]string{
		"Principal.cometa":  "usar mundo/mapa\nfn inicio()\n\timprimir(mapa.ancho())\n",
		"mundo/mapa.cometa": "pub fn ancho() entero\n\tdevolver \"dieciséis\"\n",
	})
	err := cometa.Check("Principal.cometa", cometa.Options{Files: files})
	got := diagnostics(t, err)
	if got[0].File != "mundo/mapa.cometa" || got[0].Line != 1 || got[0].Stage != "sema" {
		t.Fatalf("unexpected diagnostic %+v", got[0])
	}
	if strings.Contains(err.Error(), "cometa-memoria") {
		t.Fatalf("virtual directory leaked: %v", err)
	}
	data, _ := json.Marshal(got[0])
	if !strings.Contains(string(data), `"archivo":"mundo/mapa.cometa"`) || !strings.Contains(string(data), `"linea":1`) {
		t.Fatalf("unexpected JSON %s", data)
	}
}

func TestMemoryMissingFiles(t *testing.T) {
	files := project(map[string]string{"principal.cometa": "usar falta\nfn inicio()\n\timprimir(falta.x)\n"})
	got := diagnostics(t, cometa.Check("principal.cometa", cometa.Options{Files: files}))
	if got[0].File != "principal.cometa" || got[0].Line != 1 || !strings.Contains(got[0].Message, "falta.cometa no existe") {
		t.Fatalf("unexpected diagnostic %+v", got[0])
	}
	got = diagnostics(t, cometa.Check("otro.cometa", cometa.Options{Files: files}))
	if got[0].File != "" || !strings.Contains(got[0].Message, "otro.cometa") {
		t.Fatalf("unexpected diagnostic %+v", got[0])
	}
	got = diagnostics(t, cometa.Check("principal.cometa", cometa.Options{Files: project(map[string]string{"principal.cometa": "", "../fuera.cometa": ""})}))
	if !strings.Contains(got[0].Message, "sale del proyecto") {
		t.Fatalf("unexpected diagnostic %+v", got[0])
	}
}

func TestMemoryLineDirectivesAreRelative(t *testing.T) {
	files := project(map[string]string{"principal.cometa": "fn inicio()\n\timprimir(1)\n"})
	generated, err := cometa.Compile("principal.cometa", cometa.Options{Files: files, LineDirectives: true})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte("//line principal.cometa:")) {
		t.Fatalf("missing relative //line directive:\n%s", generated)
	}
	if bytes.Contains(generated, []byte("cometa-memoria")) {
		t.Fatal("virtual directory leaked into generated code")
	}
}

func TestMemoryResources(t *testing.T) {
	var image bytes.Buffer
	if err := png.Encode(&image, newImage()); err != nil {
		t.Fatal(err)
	}
	files := project(map[string]string{
		"principal.cometa":   "usar arte/modelo\nfn inicio()\n\timprimir(1)\n",
		"arte/modelo.cometa": "usar std/pincel/recursos\npub var sprite = recursos.imagen(\"heroe.png\")\n",
	})
	files["arte/heroe.png"] = image.Bytes()
	generated, err := cometa.Compile("principal.cometa", cometa.Options{Files: files})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte("iVBORw0KGgo")) {
		t.Fatal("image bytes not embedded")
	}
	delete(files, "arte/heroe.png")
	got := diagnostics(t, cometa.Check("principal.cometa", cometa.Options{Files: files}))
	if got[0].File != "arte/modelo.cometa" || !strings.Contains(got[0].Message, "heroe.png") {
		t.Fatalf("unexpected diagnostic %+v", got[0])
	}
}

func TestCompileFromDiskAndLoader(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "principal.cometa")
	if err := os.WriteFile(entry, []byte("fn inicio()\n\timprimir(1)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cometa.Compile(entry, cometa.Options{}); err != nil {
		t.Fatal(err)
	}
	overlay := cometa.Options{Loader: func(path string) ([]byte, error) {
		return []byte("fn inicio()\n\timprimir(x)\n"), nil
	}}
	got := diagnostics(t, cometa.Check(entry, overlay))
	if got[0].Line != 2 || !strings.HasSuffix(got[0].File, "principal.cometa") {
		t.Fatalf("unexpected diagnostic %+v", got[0])
	}
}

func TestConcurrentCompiles(t *testing.T) {
	files := project(map[string]string{
		"principal.cometa":  "usar std/mate\nusar mundo/mapa\nfn inicio()\n\timprimir(mate.absoluto(mapa.ancho()))\n",
		"mundo/mapa.cometa": "pub fn ancho() entero -16\n",
	})
	want, err := cometa.Compile("principal.cometa", cometa.Options{Files: files})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := cometa.Compile("principal.cometa", cometa.Options{Files: files})
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("concurrent compile differs: %v", err)
			}
		}()
	}
	wg.Wait()
}

func newImage() image.Image { return image.NewRGBA(image.Rect(0, 0, 2, 2)) }
