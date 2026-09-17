package compiler

import (
	"strings"
	"testing"
)

func TestGameConfigurationModuleHelper(t *testing.T) {
	for _, phase := range []string{"iniciar", "actualizar"} {
		main := "usar ajustes\nfn iniciar() ajustes.configurar()\n" + minimalGame
		if phase == "actualizar" {
			main = strings.Replace(main, "imprimir(dt)", "ajustes.configurar()", 1)
		}
		entry, loader := memoryProject(t, map[string]string{
			"main.hacha":    main,
			"ajustes.hacha": "fn configurar() juego.configuracion(titulo = \"Modular\")\n",
		})
		_, err := CompileProject(entry, loader)
		if phase == "iniciar" && err != nil {
			t.Fatal(err)
		}
		if phase == "actualizar" && (err == nil || !strings.Contains(err.Error(), "configuración solo se permite desde iniciar")) {
			t.Fatalf("missing imported helper effect: %v", err)
		}
	}
}

func TestGameInvalidConfigurationRuntime(t *testing.T) {
	generated, err := Compile("game.hacha", []byte(minimalGame))
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1)
	source = strings.Replace(source, "import (", "import (\n \"os\"\n \"os/exec\"\n \"strconv\"", 1)
	source += `
func main(){
 invalid:=[]_hgConfigJuego{
  {0,180,"",1,false,false,60},
  {320,-1,"",1,false,false,60},
  {320.5,180,"",1,false,false,60},
  {320,180,"",0,false,false,60},
  {320,180,"",math.NaN(),false,false,60},
  {320,180,"",math.Inf(1),false,false,60},
  {320,180,"",1,false,false,0},
  {320,180,"",1,false,false,59.5},
  {32769,180,"",1,false,false,60},
  {320,180,"",200,false,false,60},
 }
 if len(os.Args)>1 {
  i,_:=strconv.Atoi(os.Args[1])
  _hgvalidarConfig(invalid[i])
  return
 }
 for i:=range invalid {
  out,err:=exec.Command(os.Args[0],strconv.Itoa(i)).CombinedOutput()
  if err==nil || !(bytes.Contains(out,[]byte("configuración"))||bytes.Contains(out,[]byte("enteros"))) {panic(fmt.Sprintf("accepted invalid configuration %d: %s",i,out))}
 }
 fmt.Println("ok")
}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
