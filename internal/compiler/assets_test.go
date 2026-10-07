package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An editor analyzes the project on every keystroke, so analysis must not
// decode resources, and checking them must decode each file only once.
func TestAnalysisLeavesAssetsToTheCache(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	image, err := os.ReadFile(filepath.Join("..", "..", "examples", "pincel", "assets", "jugador.png"))
	if err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(dir, "jugador.png")
	if err := os.WriteFile(asset, image, 0o600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "juego.cometa")
	source := "usar std/pincel/recursos\nvar sprite = recursos.imagen(\"jugador.png\")\nfn inicio()\n\timprimir(sprite)\n"
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	reads := 0
	loader := func(path string) ([]byte, error) {
		if strings.HasSuffix(path, ".png") {
			reads++
		}
		return os.ReadFile(path)
	}

	p, err := AnalyzeProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if reads != 0 {
		t.Fatalf("AnalyzeProject leyó el recurso %d veces", reads)
	}
	var cache AssetCache
	for i := 0; i < 3; i++ {
		if err := p.CheckAssets(&cache); err != nil {
			t.Fatal(err)
		}
	}
	if reads != 1 {
		t.Fatalf("CheckAssets leyó el recurso %d veces, se esperaba 1", reads)
	}

	// A changed file is validated again, and its error is cached too.
	if err := os.WriteFile(asset, []byte("esto no es una imagen"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		err := p.CheckAssets(&cache)
		if err == nil || !strings.Contains(err.Error(), `recurso "jugador.png"`) || !strings.Contains(err.Error(), "juego.cometa:2:") {
			t.Fatalf("se esperaba un error del recurso con su posición, se obtuvo %v", err)
		}
	}
	if reads != 2 {
		t.Fatalf("tras cambiar el archivo se leyó %d veces, se esperaba 2", reads)
	}
	if _, err := CompileProject(entry, loader); err == nil || !strings.Contains(err.Error(), `recurso "jugador.png"`) {
		t.Fatalf("CompileProject debe rechazar el recurso inválido, se obtuvo %v", err)
	}

	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if err := p.CheckAssets(&cache); err == nil || !strings.Contains(err.Error(), `recurso "jugador.png"`) {
		t.Fatalf("se esperaba un error por el recurso ausente, se obtuvo %v", err)
	}
}
