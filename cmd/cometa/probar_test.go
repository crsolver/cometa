package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "pruebas.cometa")
}

func TestProbarReportsResults(t *testing.T) {
	entry := writeProject(t, map[string]string{
		"solucion.cometa": "pub fn doble(n entero) entero\n\tn * 2\n",
		"pruebas.cometa": `usar solucion
usar std/pruebas

fn prueba_ok()
	pruebas.igual(4, solucion.doble(2))

fn prueba_mal()
	pruebas.igual(7, solucion.doble(3), "doble de 3")

fn prueba_crash()
	var v = [1, 2]
	var i = 5
	imprimir(v[i])
`,
	})
	report, code := probar(entry, "", time.Minute, true)
	if code != exitTestsFailed || report.Status != "fallo" || report.Total != 3 || report.Passed != 1 || report.Failed != 2 {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
	status := map[string]testResult{}
	for _, r := range report.Tests {
		status[r.Name] = r
	}
	if status["prueba_ok"].Status != "ok" {
		t.Fatalf("prueba_ok: %+v", status["prueba_ok"])
	}
	if r := status["prueba_mal"]; r.Status != "fallo" || r.Message != "doble de 3: se esperaba 7 pero se obtuvo 6" || r.Line != 8 {
		t.Fatalf("prueba_mal: %+v", r)
	}
	if r := status["prueba_crash"]; r.Status != "error" || r.Line != 13 {
		t.Fatalf("prueba_crash: %+v", r)
	}
}

func TestProbarTimeoutAndCompileError(t *testing.T) {
	entry := writeProject(t, map[string]string{"pruebas.cometa": "fn prueba_a()\n\tmientras verdadero\n\t\tcontinuar\n\nfn prueba_b()\n\timprimir(1)\n"})
	report, code := probar(entry, "", 2*time.Second, true)
	if code != exitTimeout || report.Tests[0].Status != "tiempo_agotado" || report.Tests[1].Status != "omitida" {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
	entry = writeProject(t, map[string]string{"pruebas.cometa": "usar std/pruebas\nfn prueba_x()\n\tpruebas.igual(1, \"a\")\n"})
	report, code = probar(entry, "", time.Minute, true)
	if code != exitCompile || report.Status != "error_compilacion" || len(report.Diagnostics) != 1 || report.Diagnostics[0].File != "pruebas.cometa" || report.Diagnostics[0].Line != 3 {
		t.Fatalf("unexpected report %+v code %d", report, code)
	}
}
