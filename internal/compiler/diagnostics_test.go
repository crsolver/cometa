package compiler

import (
	"errors"
	"hacha/internal/diagnostic"
	"hacha/internal/sema"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMultipleFrontendDiagnostics(t *testing.T) {
	tests := []struct {
		name, source string
		count        int
		absent       string
	}{
		{"statements", "fn inicio()\n\tvar a entero = verdadero\n\timprimir(a + 1)\n\tvar b bool = 2\n", 2, "no existe"},
		{"mixed", "fn inicio()\n\tvar a =\n\tvar b entero = verdadero\n\timprimir(a)\n\t~\n\tvar c bool = 1\n", 4, "no existe"},
		{"functions", "fn uno()\n\timprimir(a)\nfn dos()\n\timprimir(b)\n", 2, ""},
		{"signatures", "fn uno(a Fantasma)\n\timprimir(a)\nfn dos(b Desconocido)\n\timprimir(b)\nfn inicio()\n\tuno(1)\n\tvar x bool = 2\n", 3, "función"},
		{"operands", "fn inicio()\n\timprimir(a + b)\n", 2, ""},
		{"indentation", "fn inicio()\n    imprimir(1)\nfn otra()\n\tvar a bool = 1\n", 2, "bloque indentado"},
		{"nested", "fn inicio()\n\tsi verdadero\n\t\tvar a bool = 1\n\t\tvar b bool = 2\n\tvar c entero = falso\n", 3, ""},
		{"lexical binding", "fn inicio()\n\tvar a = ~\n\timprimir(a)\n\ta = 1\n\tvar b bool = 1\n", 2, "no existe"},
		{"arguments", "fn f(a entero, b entero)\n\timprimir(a + b)\nfn inicio()\n\tf(verdadero, falso)\n", 2, ""},
		{"lists", "fn inicio()\n\tvar x [entero] = [verdadero, falso]\n", 2, ""},
		{"conditions", "fn inicio()\n\tsi ausente\n\t\tvar x bool = 1\n\tsino\n\t\tvar y bool = 2\n", 3, ""},
		{"value branches", "fn f() entero\n\tsi verdadero\n\t\tausente\n\tsino\n\t\totro\n", 2, "produc"},
		{"match syntax", "enum E\n\tA\n\tB\nfn inicio()\n\tvar e E = .A\n\tcasos e\n\t\t.A imprimir(1)\n\t\t.B =>\n\t\t\tvar b bool = 1\n\tvar c bool = 2\n", 3, "exhaustivo"},
		{"match values", "enum E\n\tA\n\tB\nfn f(e E) entero\n\tcasos e\n\t\t.A => ausente\n\t\t.B => otro\n", 2, "produc"},
		{"cycle and independent", "tipo A\n\ta A\nfn inicio()\n\tvar a A = {}\n\tvar b bool = 1\n", 2, "no existe"},
		{"member signatures", "tipo A\n\tx Fantasma\n\ty Ausente\nfn f(a Otro, b Desconocido)\n\timprimir(a)\n", 4, ""},
		{"member syntax", "tipo A\n\tx =\n\ty =\nfn inicio()\n\tvar a A = {}\n\tvar b bool = 1\n", 3, "no existe"},
		{"interface syntax", "interfaz I\n\tfn a(\n\tfn b(\nfn inicio()\n\tvar b bool = 1\n", 3, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, model, err := Analyze("errores.hacha", []byte(tt.source))
			if program == nil || model == nil {
				t.Fatal("missing partial analysis")
			}
			if got := len(diagnostic.Flatten(err)); got != tt.count {
				t.Fatalf("got %d errors, want %d:\n%v", got, tt.count, err)
			}
			if tt.absent != "" && strings.Contains(err.Error(), tt.absent) {
				t.Fatalf("cascading error: %v", err)
			}
			_, projectErr := AnalyzeProject(filepath.Join(t.TempDir(), "errores.hacha"), func(string) ([]byte, error) { return []byte(tt.source), nil })
			if got := len(diagnostic.Flatten(projectErr)); got != tt.count {
				t.Fatalf("project got %d errors, want %d:\n%v", got, tt.count, projectErr)
			}
			if code, err := Compile("errores.hacha", []byte(tt.source)); err == nil || code != nil {
				t.Fatal("generated invalid program")
			}
		})
	}
}

func FuzzAnalyzeRecovery(f *testing.F) {
	for _, seed := range []string{
		"fn inicio()\n\tvar a =\n\t~\n\timprimir(a)\n",
		"enum E\n\tA\n\tB\nfn f(e E) entero\n\tcasos e\n\t\t.A => 1\n\t\t.B => 2\n",
		"tipo Caja<T>\n\tvalor T\nfn inicio()\n\tvar a Caja<entero> = {valor: 1}\n",
		"interfaz I\n\tfn f() entero\nfn g(x I)\n\timprimir(x.f())\n",
		"fn inicio()\n\timprimir(\"hola ${ausente}\")\n",
		"tipo A\n\ta A\nfn inicio()\n\tvar a A = {}\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 8192 {
			t.Skip()
		}
		done := make(chan struct{})
		go func() { Analyze("fuzz.hacha", []byte(source)); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("frontend recovery did not terminate")
		}
	})
}

func TestRecoveryAtEverySourcePrefix(t *testing.T) {
	seeds := []string{
		"fn inicio()\n\tvar a = [1, 2]\n\trepetir n en a\n\t\tsi n == 1\n\t\t\timprimir(\"${n}\")\n\t\tsino\n\t\t\tcontinuar\n",
		"enum E\n\tA\n\tB entero\nfn f(e E) entero\n\tcasos e |n|\n\t\t.A => 1\n\t\t.B => n\n",
		"tipo Caja<T>\n\tvalor T\nfn inicio()\n\tvar a Caja<entero> = {valor: 1}\n",
	}
	for _, seed := range seeds {
		for i := range seed {
			Analyze("prefix.hacha", []byte(seed[:i]))
		}
	}
}

func TestDiagnosticErrorCompatibilityAndLimit(t *testing.T) {
	_, _, err := Analyze("errores.hacha", []byte("fn inicio()\n\tvar a bool = 1\n\tvar b bool = 2\n"))
	var semantic *sema.Error
	if !errors.As(err, &semantic) {
		t.Fatalf("errors.As: %v", err)
	}
	source := "fn inicio()\n" + strings.Repeat("\timprimir(noExiste)\n", 120)
	_, _, err = Analyze("errores.hacha", []byte(source))
	if got := len(diagnostic.Flatten(err)); got != 101 || !strings.Contains(err.Error(), "límite de 100") {
		t.Fatalf("limit: %d %v", got, err)
	}
}

func TestProjectMultipleDiagnostics(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"inicio.hacha": "usar a\nusar b\nusar ausente\nfn inicio()\n\tvar x bool = 1\n\tausente.hacer()\n",
		"a.hacha":      "usar comun\nfn a()\n\tvar a =\n\tvar b bool = 2\n",
		"b.hacha":      "usar comun\nfn b()\n\tvar c entero = falso\n",
		"comun.hacha":  "fn comun()\n\tvar d bool = 4\n",
	}
	p, err := AnalyzeProject(filepath.Join(dir, "inicio.hacha"), func(path string) ([]byte, error) {
		if s, ok := sources[filepath.Base(path)]; ok {
			return []byte(s), nil
		}
		return nil, errors.New("no existe")
	})
	if p == nil || p.Model == nil || len(p.Modules) != 4 {
		t.Fatalf("partial project: %#v", p)
	}
	if got := len(diagnostic.Flatten(err)); got != 6 {
		t.Fatalf("got %d errors, want 6:\n%v", got, err)
	}
	if strings.Contains(err.Error(), "HachaModulo") {
		t.Fatalf("private linkage: %v", err)
	}
}
