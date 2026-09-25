package compiler

import (
	"strings"
	"testing"
)

func TestGameObjectFromImportedModule(t *testing.T) {
	entry, loader := memoryProject(t, map[string]string{
		"main.cometa":    "usar std/pincel/juego\nusar partida\nfn inicio()\n\tjuego.ejecutar(partida.Partida {}, titulo = \"Modular\") capturar |e| imprimir(e)\n",
		"partida.cometa": "usar std/pincel/graficos\npub tipo Partida\n\tpub fn actualizar(dt decimal) imprimir(dt)\n\tpub fn pintar() graficos.limpiar(.Negro)\npub fn iniciar() imprimir(0)\n",
	})
	if _, err := CompileProject(entry, loader); err != nil {
		t.Fatal(err)
	}
}

func TestGameInvalidConfigurationRuntime(t *testing.T) {
	generated, err := Compile("game.cometa", []byte(pincelImports+minimalGame))
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated) + `
type invalidGame struct{}
func (*invalidGame) _cometa_public_Actualizar() {}
func (*invalidGame) _cometa_public_Pintar() {}
func (*invalidGame) Actualizar(float64){}
func (*invalidGame) Pintar(){}
func main(){
 invalid:=[]_hgConfigJuego{
 {0,180,"",1,false,false,60},{320,-1,"",1,false,false,60},
 {320,180,"",0.001,false,false,60},{320,180,"",0,false,false,60},
 {320,180,"",math.NaN(),false,false,60},{320,180,"",math.Inf(1),false,false,60},
 {320,180,"",1,false,false,0},{320,180,"",1,false,false,-1},
 {32769,180,"",1,false,false,60},{320,180,"",200,false,false,60},
 }
 for _,c:=range invalid {
  result:=_hgejecutar(&invalidGame{},c.Ancho,c.Alto,c.Titulo,c.Escala,c.Redimensionable,c.Pantalla_completa,c.Tps,false,false)
  if result.tag!=2||result.payload2==""{panic("accepted invalid configuration")}
 }
 if _hgrunStarted{panic("invalid configuration started the game")}
 _hgrunStarted=true
 if _hgejecutar(&invalidGame{},320,180,"",1,false,false,60,false,false).tag!=2{panic("accepted repeated run")}
 fmt.Println("ok")
}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}

func TestGameInterfaceAndStartupErrors(t *testing.T) {
	for _, source := range []string{
		"tipo T\n\tpub fn actualizar(dt decimal) imprimir(dt)\nfn inicio()\n\tjuego.ejecutar(T {}) capturar |e| imprimir(e)\n",
		"tipo T\n\tpub fn actualizar(dt cadena) imprimir(dt)\n\tpub fn pintar() imprimir(0)\nfn inicio()\n\tjuego.ejecutar(T {}) capturar |e| imprimir(e)\n",
		"tipo T\n\tpub fn actualizar(dt decimal) imprimir(dt)\n\tpub fn pintar() imprimir(0)\nfn inicio() juego.ejecutar(T {})\n",
	} {
		if _, err := Compile("game.cometa", []byte("usar std/pincel/juego\n"+source)); err == nil {
			t.Fatal("accepted invalid game")
		}
	}
	generated, err := Compile("normal.cometa", []byte("fn actualizar(dt cadena) imprimir(dt)\nfn pintar() imprimir(0)\nfn iniciar() imprimir(0)\nfn inicio() pintar()\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("callback names activated game")
	}
	runGeneratedGo(t, generated, "0\n")
}
