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
	return filepath.Join(root, "main.hacha"), func(path string) ([]byte, error) {
		if source, ok := sources[path]; ok {
			return source, nil
		}
		return nil, os.ErrNotExist
	}
}

func TestProjectQualifiedDeclarations(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.hacha": `usar modelos como m
usar util
fn recibir<T m.Describible>(valor T) cadena valor.describir()
fn crear() m.Usuario m.Usuario {nombre: "Ana"}
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
		"modelos.hacha": `interfaz Describible
	fn describir() cadena
tipo Usuario
	nombre cadena
	fn describir() cadena @nombre
tipo Caja<T>
	valor T
enum Evento
	Texto cadena
	Vacio
fn saludar(nombre cadena = "mundo", saludo cadena = "hola ") cadena saludo
`,
		"util.hacha": "fn identidad<T>(valor T) T valor\n",
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "func main()") || !strings.Contains(string(generated), "HachaModulo") {
		t.Fatal(string(generated))
	}
	runGeneratedGo(t, generated, "Ana\nhola \n3\ntexto\n")
}

func TestProjectEmbeddingAndDefaultsRuntime(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.hacha": `usar modelos como m
tipo Grupo
	m.Usuario
	fn renombrar(nombre cadena)
		@Usuario.nombre = nombre
fn mostrar(valor m.Describible) cadena valor.describir()
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
		"modelos.hacha": `interfaz Describible
	fn describir() cadena
tipo Usuario
	nombre cadena
	fn describir() cadena @nombre
	fn saludar(nombre cadena = @nombre) cadena nombre
`,
	})
	generated, err := CompileProject(entry, loader)
	if err != nil {
		t.Fatal(err)
	}
	runGeneratedGo(t, generated, "Ana\nEva\nEva\nLuis\n")
}

func TestProjectStableOutputAndNoEntry(t *testing.T) {
	files := map[string]string{"main.hacha": "usar a\nfn crear() a.Usuario a.Usuario {}\n", "a.hacha": "tipo Usuario\n\tx entero\n"}
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
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.hacha")
	if err := os.WriteFile(lib, []byte("fn f() imprimir(1)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lib, filepath.Join(dir, "alias.hacha")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	entry := filepath.Join(dir, "main.hacha")
	if err := os.WriteFile(entry, []byte("usar lib\nusar alias\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompileProject(entry, nil); err == nil || !strings.Contains(err.Error(), "duplicada") {
		t.Fatal(err)
	}
}

func TestProjectFailures(t *testing.T) {
	for _, tc := range []struct{ name, root, a, b, want string }{
		{"missing", "usar ausente\n", "", "", "no se pudo importar"},
		{"cycle", "usar a\n", "usar main\n", "", "ciclo"},
		{"self", "usar main\n", "", "", "ciclo"},
		{"duplicate", "usar a\nusar ./a como otro\n", "", "", "duplicada"},
		{"alias", "usar a\nfn a()\n\timprimir(1)\n", "", "", "conflicto"},
		{"entry", "usar a\n", "fn inicio() imprimir(1)\n", "", "inicio"},
		{"reexport", "usar a\nfn inicio() a.b.f()\n", "usar b\n", "fn f() imprimir(1)\n", "desconocida"},
		{"leak", "usar a\nfn inicio() f()\n", "fn f() imprimir(1)\n", "", "desconocida"},
		{"nominal", "usar a\nusar b\nfn f(valor a.Usuario)\n\timprimir(valor)\nfn inicio() f(b.Usuario {})\n", "tipo Usuario\n\tx entero\n", "tipo Usuario\n\tx entero\n", "Usuario"},
		{"dependency error", "usar a\n", "fn f() entero \"error\"\n", "", "entero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry, loader := memoryProject(t, map[string]string{"main.hacha": tc.root, "a.hacha": tc.a, "b.hacha": tc.b})
			_, err := CompileProject(entry, loader)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if tc.name == "dependency error" && !strings.Contains(err.Error(), "a.hacha:1:") {
				t.Fatal(err)
			}
		})
	}
}

func TestProjectDiamondAndShadowing(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.hacha":  "usar a\nusar sub/b como b\nfn inicio()\n\tvar a = b.crear()\n\timprimir(a.nombre)\n",
		"a.hacha":     "usar comun\nfn crear() comun.Usuario comun.Usuario {}\n",
		"sub/b.hacha": "usar ../comun\nfn crear() comun.Usuario comun.Usuario {}\n",
		"comun.hacha": "tipo Usuario\n\tnombre cadena\n",
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
