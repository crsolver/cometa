package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformerExampleCompiles(t *testing.T) {
	if _, err := CompileProject(filepath.Join("..", "..", "examples", "plataformas.hacha"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestPlatformerSimulation(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "plataformas", "mundo.hacha")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := Compile(path, source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "ebiten") {
		t.Fatal("simulation should not require a graphics context")
	}
	harness := `
func require(ok bool, message string) { if !ok { panic(message) } }
func arena() *Mundo {
 m:=&Mundo{};m.Generar(42)
 for y:=int64(1);y<67;y++ {for x:=int64(1);x<239;x++ {m.Poner(x,y,0)}}
 for x:=int64(1);x<239;x++ {m.Poner(x,20,3)}
 m.Jugador=&Cuerpo{Pos:_hgVec2{160,149},Ancho:6,Alto:11,Suelo:true}
 m.Enemigos=nil;m.Invulnerable=0;m.Camara()
 return m
}
func jump(held bool) float64 {
 m:=arena();lowest:=m.Jugador.Pos.Y
 for i:=0;i<70;i++ {
  m.Caminar(&Control{Salto:i==0,Sostener:held||i==0},1.0/60)
  lowest=math.Min(lowest,m.Jugador.Pos.Y)
 }
 require(m.Jugador.Suelo,"jump did not land")
 return 149-lowest
}
func main() {
 for _,seed:=range []int64{1,2,42,2026,999999} {
  m:=&Mundo{};m.Generar(seed);n:=&Mundo{};n.Generar(seed)
  require(len(m.Celdas)==240*68,"map size")
  require(!m.Choca(m.Jugador.Pos,6,11),"spawn obstructed")
  materials:=map[int64]bool{}
  for i,v:=range m.Celdas {require(v==n.Celdas[i],"seed not deterministic");materials[v]=true}
  require(len(materials)==6,"missing material")
  for _,e:=range m.Enemigos {require(!m.Choca(e.Cuerpo.Pos,8,7),"enemy spawned in terrain")}
  require(len(m.Enemigos)>10,"world unpopulated")
  // Flood all air from the spawn: every section's main gallery must connect.
  start:=int64(12*240+8);queue:=[]int64{start};seen:=map[int64]bool{start:true}
  for h:=0;h<len(queue);h++ {i:=queue[h];for _,d:=range []int64{-1,1,-240,240} {
   j:=i+d;if j>=0&&j<int64(len(m.Celdas))&&!seen[j]&&m.Celdas[j]==0 {seen[j]=true;queue=append(queue,j)}
  }}
  for s:=int64(0);s<6;s++ {x:=s*40+8;for level:=int64(0);level<2;level++ {
   y:=33+level*21+int64(math.Floor(math.Sin(float64(x)*0.045)*2))-2
   require(seen[y*240+x],"gallery disconnected from spawn")
  }}
 }
 high,low:=jump(true),jump(false)
 require(high>28&&high<40&&low>5&&low<high-10,"variable jump height")
 m:=arena()
 for i:=0;i<20;i++ {m.Caminar(&Control{Eje:1},1.0/60)}
 require(m.Jugador.Vel.X==88,"run speed")
 for i:=0;i<6;i++ {m.Caminar(&Control{},1.0/60)}
 require(m.Jugador.Vel.X==0,"ground braking")
 // A jump shortly after leaving a ledge works; an expired one does not.
 m=arena();m.Jugador.Suelo=false;m.Coyote=0.08
 m.Caminar(&Control{Salto:true,Sostener:true},1.0/60)
 require(m.Jugador.Vel.Y< -200,"coyote jump")
 m=arena();m.Jugador.Pos.Y=90;m.Jugador.Suelo=false;m.Coyote=0
 m.Caminar(&Control{Salto:true,Sostener:true},1.0/60)
 require(m.Jugador.Vel.Y>0,"midair jump allowed")
 m=arena();m.Jugador.Pos.Y=145;m.Jugador.Suelo=false;m.Jugador.Vel.Y=100
 for i:=0;i<5;i++ {m.Caminar(&Control{Salto:i==0,Sostener:true},1.0/60)}
 require(m.Jugador.Vel.Y<0,"buffered landing jump")
 // Walls, ceilings and floors at high velocity cannot be crossed.
 m=arena();for y:=int64(1);y<20;y++ {m.Poner(25,y,3)}
 m.Jugador.Vel=_hgVec2{1000,1000};m.Mover(m.Jugador,0.2)
 require(m.Jugador.Pos.X+6<=200&&!m.Choca(m.Jugador.Pos,6,11),"wall/floor tunneling")
 m=arena();m.Poner(20,14,3);m.Jugador.Vel.Y=-1000;m.Mover(m.Jugador,0.1)
 require(m.Jugador.Pos.Y>=120,"ceiling tunneling")
 // Exact room boundary behavior, including the half tile at y=180.
 for _,c:=range []struct{x,y float64;cx,cy int64}{{316.9,174.4,0,0},{317,174.5,320,180},{636.9,354.4,320,180},{637,354.5,640,360},{10,10,0,0},{1910,525,1600,360}} {
  m.Jugador.Pos=_hgVec2{c.x,c.y};m.Camara();require(m.Cam_x==c.cx&&m.Cam_y==c.cy,"camera transition")
 }
 m=arena();m.Poner(23,18,4);m.Poner(24,18,3)
 require(m.Apuntar(_hgVec2{196,148})==18*240+23,"mining through front block")
 require(m.Apuntar(_hgVec2{260,148})==-1,"mining reach")
 ctl:=&Control{Minar:true,Raton:_hgVec2{188,148}}
 for i:=0;i<40;i++ {m.Herramientas(ctl,1.0/60)}
 require(m.Celda(23,18)==0&&m.Mineral==1,"ore mining")
 m.Reaparecer();require(m.Celda(23,18)==0&&m.Mineral==1,"respawn lost world edits")
 require(m.Apuntar(_hgVec2{64,114})==-1,"spawn platform is mineable")
 // Sweep bullets against the first terrain/enemy hit in all directions.
 for _,d:=range []_hgVec2{{1,0},{-1,0},{0,1},{0,-1},{0.707,0.707},{-0.707,-0.707}} {
  m=arena();p:=_hgVec2{500,100};target:=_hgadd(p,_hgscale(d,18))
  e:=&Enemigo{Cuerpo:&Cuerpo{Pos:_hgsub(target,_hgVec2{4,3}),Ancho:8,Alto:7},Vida:3}
  m.Enemigos=[]*Enemigo{e};m.Balas=[]*Bala{{Pos:p,Dir:d,Vida:1}}
  m.Combatir(0.1);require(e.Vida==2&&len(m.Balas)==0,"directional projectile missed")
 }
 m=arena();m.Poner(23,18,3)
 e:=&Enemigo{Cuerpo:&Cuerpo{Pos:_hgVec2{198,146},Ancho:8,Alto:7},Vida:3}
 m.Enemigos=[]*Enemigo{e};m.Balas=[]*Bala{{Pos:_hgVec2{163,150},Dir:_hgVec2{1,0},Vida:1}}
 m.Combatir(0.2);require(e.Vida==3&&len(m.Balas)==0,"shot passed through terrain")
 m=arena();e=&Enemigo{Cuerpo:&Cuerpo{Pos:m.Jugador.Pos,Ancho:8,Alto:7},Vida:3}
 m.Enemigos=[]*Enemigo{e};m.Criaturas(0);require(m.Vida==4,"contact damage")
 m.Criaturas(0);require(m.Vida==4,"invulnerability")
 m.Invulnerable=0;m.Vida=1;m.Criaturas(0);require(m.Vida==5&&m.Jugador.Pos.X==61,"death respawn")
 m=arena();m.Emitir(_hgVec2{},0,500);require(len(m.Particulas)==180,"particle cap")
 for i:=0;i<120;i++ {m.Actualizar(&Control{},1.0/60)}
 require(len(m.Particulas)<10,"particles did not expire")
}
`
	runGeneratedGo(t, append(generated, []byte(harness)...), "")
}

// Exercise the real draw code in a hidden Ebitengine context, including the
// partially visible tile row at vertical room boundaries. Optional captures
// are useful when adjusting the example's palette and sprites.
func TestPlatformerRenderingRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("desktop rendering")
	}
	generated, err := CompileProject(filepath.Join("..", "..", "examples", "plataformas.hacha"), nil)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(generated), "func main() {", "func unusedMain() {", 1)
	source = strings.Replace(source, `_ "image/png"`, `"image/png"`, 1)
	source = strings.Replace(source, "import (", "import (\n\"os\"\n\"path/filepath\"", 1)
	source += `
type platformerRender struct{}
func (*platformerRender) Layout(w,h int)(int,int){return 320,180}
func (*platformerRender) Draw(screen *ebiten.Image){}
func (*platformerRender) Update()error {
 m:=HachaGlobal_70617274696461;m.Generar(42);m.Reloj=10;m.Invulnerable=0
 _hgscreen=ebiten.NewImage(320,180);defer func(){_hgscreen.Dispose();_hgscreen=nil}()
 for row:=int64(0);row<3;row++ {for col:=int64(0);col<6;col++ {
  m.Cam_x=col*320;m.Cam_y=row*180
  m.Mira=_hgVec2{float64(m.Cam_x+180),float64(m.Cam_y+80)}
  if row==0&&col==0 {m.Jugador.Pos=_hgVec2{61,101}} else {
   x:=col*40+8;y:=int64(14);if row>0 {y=33+(row-1)*21+int64(math.Floor(math.Sin(float64(x)*0.045)*2))}
   m.Jugador.Pos=_hgVec2{float64(x*8),float64(y*8-11)}
  }
  m.Fogonazo=0.02;m.Emitir(m.Centro(),2,3)
  _hgrestablecerCamara();(&Partida{}).Pintar()
  pixels:=make([]byte,320*180*4);_hgscreen.ReadPixels(pixels)
  unique:=map[[3]byte]bool{};for i:=0;i<len(pixels);i+=4 {unique[[3]byte{pixels[i],pixels[i+1],pixels[i+2]}]=true}
  if len(unique)<20 {panic("platformer scene lacks terrain/detail")}
  if dir:=os.Getenv("HACHA_PLATFORMER_CAPTURES");dir!=""&&col==0 {
   if err:=os.MkdirAll(dir,0755);err!=nil{panic(err)}
   for _,scale:=range []int{1,4} {
    img:=image.NewNRGBA(image.Rect(0,0,320*scale,180*scale))
    for y:=0;y<180*scale;y++ {for x:=0;x<320*scale;x++ {
     i:=((y/scale)*320+x/scale)*4;img.SetNRGBA(x,y,color.NRGBA{pixels[i],pixels[i+1],pixels[i+2],pixels[i+3]})
    }}
    f,err:=os.Create(filepath.Join(dir,fmt.Sprintf("section-%d-%dx.png",row,scale)));if err!=nil{panic(err)}
    if err:=png.Encode(f,img);err!=nil{panic(err)};if err:=f.Close();err!=nil{panic(err)}
   }
  }
 }}
 return ebiten.Termination
}
func main(){ebiten.SetWindowVisible(false);ebiten.SetRunnableOnUnfocused(true);if err:=ebiten.RunGame(&platformerRender{});err!=nil{panic(err)}}
`
	runGeneratedGo(t, []byte(source), "")
}
