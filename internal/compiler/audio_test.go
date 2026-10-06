package compiler

import (
	"strings"
	"testing"
)

func TestAudioSynthesisAPI(t *testing.T) {
	source := `usar std/pincel/audio

var salto = audio.tono(330, 0.12, hasta = 660)
var golpe = audio.ruido(0.2, volumen = 0.6)

fn sonar(nivel entero)
	audio.reproducir(salto)
	audio.reproducir(golpe, 0.5, tono = 1.0 + nivel * 0.1)
	var grave = audio.tono(110, 0.3, .Triangular)
	var onda audio.Onda = si nivel > 3 audio.Onda.Sierra sino .Seno
	audio.reproducir(audio.tono(440, 0.1, onda), bucle = falso, tono = 2)
	audio.detener(audio.reproducir(grave, bucle = verdadero))
`
	got, err := Compile("audio.cometa", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"_hgaudioTono(", "_hgaudioRuido(", "_hgOnda(2)", "_hgaudioRemuestrear("} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("falta %s en el Go generado", want)
		}
	}
	for _, tc := range []struct{ name, body, want string }{
		{"onda inválida", "\taudio.reproducir(audio.tono(440, 1, .Redonda))\n", "constante inválida"},
		{"sin duración", "\taudio.reproducir(audio.tono(440))\n", `falta el parámetro "duracion"`},
		{"tono cadena", "\taudio.reproducir(audio.ruido(1), tono = \"alto\")\n", "decimal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile("caso.cometa", []byte("usar std/pincel/audio\n\nfn sonar()\n"+tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}

// The synthesized PCM is checked sample by sample: length, pitch (counted as
// rising zero crossings), level, the anti-click fade and the resampling used
// by the tono parameter of audio.reproducir. No audio device is opened.
func TestAudioSynthesisRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("builds Ebitengine")
	}
	generated, err := Compile("audio.cometa", []byte("usar std/pincel/audio\nfn sonar()\n\taudio.reproducir(audio.tono(440, 0.5), tono = 2)\n\taudio.reproducir(audio.ruido(0.1))\n"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(generated)
	if !strings.Contains(source, `"fmt"`) {
		source = strings.Replace(source, "import (", "import (\n\"fmt\"", 1)
	}
	source += `
func muestra(s *_hgSonido,i int) int {return int(int16(uint16(s.pcm[i*4])|uint16(s.pcm[i*4+1])<<8))}
func cruces(s *_hgSonido) int {
 n:=0
 for i:=1;i<len(s.pcm)/4;i++{if muestra(s,i-1)<0&&muestra(s,i)>=0{n++}}
 return n
}
func pico(s *_hgSonido) int {
 m:=0
 for i:=0;i<len(s.pcm)/4;i++{v:=muestra(s,i);if v<0{v=-v};if v>m{m=v}}
 return m
}
func cerca(nombre string,valor,esperado,margen int) {if valor<esperado-margen||valor>esperado+margen{panic(fmt.Sprintf("%s = %d, se esperaba %d ± %d",nombre,valor,esperado,margen))}}
func main(){
 la:=_hgaudioTono(440,0.5,0,0,1)
 cerca("muestras",len(la.pcm)/4,24000,0)
 cerca("cruces de 440 Hz",cruces(la),220,2)
 cerca("pico",pico(la),16383,1)
 if muestra(la,0)!=0||muestra(la,23999)!=0{panic("el sonido debe empezar y terminar en silencio")}
 for i:=0;i<24000;i++{if la.pcm[i*4]!=la.pcm[i*4+2]||la.pcm[i*4+1]!=la.pcm[i*4+3]{panic("los dos canales deben ser iguales")}}
 for onda:=_hgOnda(0);onda<4;onda++{
  s:=_hgaudioTono(1000,0.25,onda,0,0.5)
  cerca(fmt.Sprintf("cruces de la onda %d",onda),cruces(s),250,2)
  cerca(fmt.Sprintf("pico de la onda %d",onda),pico(s),8191,40)
 }
 cerca("cruces del barrido",cruces(_hgaudioTono(200,1,3,800,1)),500,3)
 nan:=0.0;nan=nan/nan
 for _,d:=range []float64{0,-1,nan}{if len(_hgaudioTono(440,d,0,0,1).pcm)!=0{panic("una duración inválida debe dar un sonido vacío")}}
 cerca("duración máxima",len(_hgaudioTono(440,100,0,0,1).pcm)/4,30*48000,0)
 if pico(_hgaudioTono(440,0.1,0,0,0))!=0||pico(_hgaudioTono(440,0.1,0,0,-2))!=0{panic("volumen 0 debe ser silencio")}
 cerca("volumen limitado",pico(_hgaudioTono(440,0.1,0,0,9)),16383,1)
 cerca("frecuencia 0",cruces(_hgaudioTono(0,0.1,3,0,1)),0,0)
 ruido:=_hgaudioRuido(0.25,1)
 cerca("muestras de ruido",len(ruido.pcm)/4,12000,0)
 cerca("cruces del ruido",cruces(ruido),3000,300)
 if string(ruido.pcm)!=string(_hgaudioRuido(0.25,1).pcm){panic("el ruido debe ser siempre el mismo")}
 agudo:=_hgaudioRemuestrear(la,2)
 cerca("muestras a tono 2",len(agudo.pcm)/4,12000,0)
 cerca("cruces a tono 2",cruces(agudo),220,2)
 grave:=_hgaudioRemuestrear(la,0.5)
 cerca("muestras a tono 0.5",len(grave.pcm)/4,48000,0)
 cerca("cruces a tono 0.5",cruces(grave),220,2)
 if _hgaudioRemuestrear(la,1)!=la||_hgaudioRemuestrear(la,0)!=la||_hgaudioRemuestrear(la,nan)!=la{panic("tono 1 o inválido debe devolver el mismo sonido")}
 cerca("tono máximo",len(_hgaudioRemuestrear(la,100).pcm)/4,6000,0)
 cerca("tono mínimo",len(_hgaudioRemuestrear(la,0.01).pcm)/4,96000,0)
 vacio:=_hgaudioTono(440,0,0,0,1)
 if _hgaudioRemuestrear(vacio,2)!=vacio{panic("un sonido vacío no se remuestrea")}
 fmt.Println("ok")
}
`
	runGeneratedGo(t, []byte(source), "ok\n")
}
