package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func memoryProject(t *testing.T, files map[string]string) (string, SourceLoader) {
	t.Helper()
	root := t.TempDir()
	sources := map[string][]byte{}
	for name, source := range files {
		path, err := CanonicalPath(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		sources[path] = []byte(source)
	}
	return filepath.Join(root, "main.cometa"), func(path string) ([]byte, error) {
		if source, ok := sources[path]; ok {
			return source, nil
		}
		return nil, os.ErrNotExist
	}
}

func TestProjectQualifiedDeclarations(t *testing.T) {
	t.Parallel()
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa": `usar modelos como m
usar util
pub fn recibir<T m.Describible>(valor T) cadena valor.describir()
pub fn crear() m.Usuario m.Usuario {nombre: "Ana"}
fn inicio()
	var usuario = crear()
	var caja = m.Caja<m.Usuario> {valor: usuario}
	imprimir(recibir(caja.valor))
	imprimir(m.saludar(saludo = "hola "))
	imprimir(util.identidad<entero>(3))
	var evento = m.Evento.Texto("texto")
	casos evento |dato|
		m.Evento.Texto => imprimir(dato)
		m.Evento.Vacio => imprimir("vacío")
	var i m.Describible = usuario
	var opt = i como m.Usuario
	var otro m.Usuario? = opt
`,
		"modelos.cometa": `pub interfaz Describible
	fn describir() cadena
pub tipo Usuario
	pub nombre cadena
	pub fn describir() cadena @nombre
pub tipo Caja<T>
	pub valor T
pub enum Evento
	Texto cadena
	Vacio
pub fn saludar(nombre cadena = "mundo", saludo cadena = "hola ") cadena saludo
`,
		"util.cometa": "pub fn identidad<T>(valor T) T valor\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "func main()") || !strings.Contains(string(generated), "CometaModulo") {
		t.Fatal(string(generated))
	}
	runGeneratedGo(t, generated, "Ana\nhola \n3\ntexto\n")
}

func TestProjectEmbeddingAndDefaultsRuntime(t *testing.T) {
	t.Parallel()
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa": `usar modelos como m
pub tipo Grupo
	pub m.Usuario
	pub fn renombrar(nombre cadena)
		@Usuario.nombre = nombre
pub fn mostrar(valor m.Describible) cadena valor.describir()
fn inicio()
	var grupo = Grupo {Usuario: m.Usuario {nombre: "Ana"}}
	imprimir(mostrar(grupo))
	grupo.renombrar("Eva")
	imprimir(grupo.describir())
	imprimir(grupo.saludar())
	var vacio = Grupo {}
	vacio.Usuario.nombre = "Luis"
	imprimir(vacio.describir())
`,
		"modelos.cometa": `pub interfaz Describible
	fn describir() cadena
pub tipo Usuario
	pub nombre cadena
	pub fn describir() cadena @nombre
	pub fn saludar(nombre cadena = @nombre) cadena nombre
`,
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "Ana\nEva\nEva\nLuis\n")
}

func TestProjectStableOutputAndNoEntry(t *testing.T) {
	t.Parallel()
	files := map[string]string{"main.cometa": "usar a\npub fn crear() a.Usuario a.Usuario {}\n", "a.cometa": "pub tipo Usuario\n\tpub x entero\n"}
	first, loadFirst := memoryProject(t, files)
	second, loadSecond := memoryProject(t, files)
	a, err := CompileProject(first, loadFirst)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CompileProject(second, loadSecond)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) || strings.Contains(string(a), "func main()") {
		t.Fatalf("non-portable output:\n%s\n%s", a, b)
	}
}

func TestProjectPhysicalIdentity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.cometa")
	if err := os.WriteFile(lib, []byte("pub fn f() imprimir(1)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lib, filepath.Join(dir, "alias.cometa")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	entry := filepath.Join(dir, "main.cometa")
	if err := os.WriteFile(entry, []byte("usar lib\nusar alias\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompileProject(entry, nil); err == nil || !strings.Contains(err.Error(), "duplicada") {
		t.Fatal(err)
	}
}

func TestProjectFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, root, a, b, want string }{
		{"missing", "usar ausente\n", "", "", "no se pudo importar"},
		{"cycle", "usar a\n", "usar main\n", "", "ciclo"},
		{"self", "usar main\n", "", "", "ciclo"},
		{"duplicate", "usar a\nusar ./a como otro\n", "", "", "duplicada"},
		{"alias", "usar a\npub fn a()\n\timprimir(1)\n", "", "", "conflicto"},
		{"entry", "usar a\n", "fn inicio() imprimir(1)\n", "", "inicio"},
		{"reexport", "usar a\nfn inicio() a.b.f()\n", "usar b\n", "pub fn f() imprimir(1)\n", "no tiene"},
		{"leak", "usar a\nfn inicio() f()\n", "pub fn f() imprimir(1)\n", "", "no existe"},
		{"nominal", "usar a\nusar b\npub fn f(valor a.Usuario)\n\timprimir(valor)\nfn inicio() f(b.Usuario {})\n", "pub tipo Usuario\n\tpub x entero\n", "pub tipo Usuario\n\tpub x entero\n", "Usuario"},
		{"dependency error", "usar a\n", "pub fn f() entero \"error\"\n", "", "entero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry, loader := memoryProject(t, map[string]string{"main.cometa": tc.root, "a.cometa": tc.a, "b.cometa": tc.b})
			_, err := CompileProject(entry, loader)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if tc.name == "dependency error" && !strings.Contains(err.Error(), "a.cometa:1:") {
				t.Fatal(err)
			}
		})
	}
}

func TestProjectDiamondAndShadowing(t *testing.T) {
	t.Parallel()
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":  "usar a\nusar sub/b como b\nfn inicio()\n\tvar a = b.crear()\n\timprimir(a.nombre)\n",
		"a.cometa":     "usar comun\npub fn crear() comun.Usuario comun.Usuario {}\n",
		"sub/b.cometa": "usar ../comun\npub fn crear() comun.Usuario comun.Usuario {}\n",
		"comun.cometa": "pub tipo Usuario\n\tpub nombre cadena\n",
	})
	project, err := AnalyzeProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Modules) != 4 {
		t.Fatal(len(project.Modules))
	}
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(generated), "Usuario struct") != 1 {
		t.Fatal(string(generated))
	}
}

// Windows file names ignore case, so modules are identified by a lower-cased
// path, but what people read (diagnostics, line directives) must show the
// file as it is named on disk, however the command line spelled it.
func TestDiagnosticsKeepTheCaseOfFileNames(t *testing.T) {
	t.Parallel()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, source string) string {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	entry := write("Proyecto/Juego.cometa", "usar Mundo/Mapa\nfn inicio()\n\timprimir(Mapa.ancho())\n\timprimir(nada)\n")
	module := write("Proyecto/Mundo/Mapa.cometa", "pub fn ancho() entero\n\tfalta\n")
	typed := entry
	if filepath.Separator == '\\' {
		// What a user may type on a case-insensitive file system.
		typed = filepath.Join(dir, "proyecto", "JUEGO.cometa")
	}

	_, err = CompileProject(typed, nil)
	if err == nil {
		t.Fatal("se esperaban errores")
	}
	for _, want := range []string{entry + ":4:11: ", module + ":2:2: "} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("falta %q en:\n%v", want, err)
		}
	}

	// Tooling keeps the canonical path, which is the key of Project.Modules.
	p, analysisErr := AnalyzeProject(typed, nil)
	key, _ := CanonicalPath(entry)
	if analysisErr == nil || !strings.Contains(analysisErr.Error(), key+":4:11: ") || p.Modules[key] == nil || p.DisplayPath(key) != entry {
		t.Fatalf("AnalyzeProject debe conservar la ruta canónica %q: %v", key, analysisErr)
	}

	write("Proyecto/Juego.cometa", "usar Mundo/Mapa\nfn inicio()\n\timprimir(Mapa.ancho())\n")
	write("Proyecto/Mundo/Mapa.cometa", "pub fn ancho() entero\n\t3\n")
	generated, err := CompileProjectForRun(typed, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"//line " + entry + ":3\n", "//line " + module + ":2\n"} {
		if !strings.Contains(string(generated), want) {
			t.Errorf("falta %q en el Go generado", want)
		}
	}
}
