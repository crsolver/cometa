# Juegos 2D con Hacha

El módulo raíz declara `fn actualizar(dt num)` y `fn pintar()`, sin resultados,
genéricos ni defaults. `fn iniciar()` es opcional.
El runtime prepara los defaults, inicializa las globales y los recursos, llama a
`iniciar`, valida y aplica la configuración final de la ventana y arranca el bucle. Los módulos importados
no declaran callbacks. `inicio` pertenece al perfil tradicional y no se combina
con juegos. `dt` es el tiempo fijo de un tick en segundos (1/60 por defecto);
el renderizado sigue la frecuencia de frames de Ebitengine.

```hacha
fn iniciar()
	juego.configuracion(480, 320, titulo = "Mi juego", escala = 2)
```

Por defecto: 320×180, escala 1, título "Hacha", 60 TPS, sin redimensionamiento ni
pantalla completa. El tamaño lógico permanece fijo cuando cambia la ventana.
Dimensiones/TPS deben ser enteros positivos, y escala positiva y finita.
Se rechazan valores superiores a 32768. `juego.configuracion` no devuelve un
valor y solo se permite desde `iniciar` y sus helpers. Todos los parámetros son
opcionales; cada llamada reemplaza la configuración completa usando sus defaults,
y la última llamada ejecutada gana. Sin llamada se usan los defaults anteriores.
`configurar` es un nombre de función ordinario, sin significado de callback.

## Valores, coordenadas y recursos

`Vec2` tiene `x num` e `y num`. `Rect` tiene `pos Vec2` y `tamano Vec2`.
`Camara2D` tiene `pos Vec2`, `origen Vec2`, `zoom num` (1 por defecto) y
`rotacion num`. Estos tipos se copian por valor.
Las estructuras del usuario conservan referencias.

X crece a la derecha, Y hacia abajo. Distancias en píxeles lógicos y ángulos en
radianes. `Vec2` admite suma/resta de vectores, negación, multiplicación por escalar
en ambos órdenes y división por escalar. Normalizar cero devuelve cero. La sintaxis
es `Vec2 {x: 2, y: 3}`; `{2, 3}` queda aplazada.

`Color`, `Tecla` y `BotonRaton` son valores opacos con constantes contextuales:

- Color: `.Blanco`, `.Negro`, `.Rojo`, `.Verde`, `.Azul`, `.Amarillo`, `.Magenta`,
  `.Cian`, `.Transparente`. `color.rgba` limita los canales a 0–255.
- Tecla: `.A` a `.Z`, `.Izquierda`, `.Derecha`, `.Arriba`, `.Abajo`, `.Espacio`,
  `.Escape`, `.Enter`, `.Tab`, `.Retroceso`, `.Shift`, `.Control`.
- BotonRaton: `.Izquierdo`, `.Derecho`, `.Medio`.

No son enums del usuario y no admiten `casos`. Se requiere un tipo esperado para
las constantes, por ejemplo `entrada.tecla_mantenida(.Espacio)`.

`Imagen`, `Fuente`, `Sonido` y `Reproduccion` son handles opacos compartidos.
No admiten literales ni nulos; un campo de recurso requiere inicialización.
Se puede usar `Imagen?` para ausencia.

```hacha
var sprite = recursos.imagen("assets/jugador.png")
var fuente = recursos.fuente("assets/texto.ttf")
var sonido = recursos.sonido("assets/recoger.wav")
```

Las llamadas de recursos solo se aceptan como inicializadores globales directos,
con rutas literales relativas al módulo que las declara. Se validan los datos y
se incorporan al Go generado. Formatos: PNG/JPEG, TTF/OTF, WAV/Ogg Vorbis/MP3.
WAV admite PCM de 8/16 bits, mono/estéreo. Audio se carga completo en memoria y se
decodifica a PCM estéreo de 48 kHz. No se necesitan archivos externos al ejecutar.

## API incorporada

Todos los namespaces están disponibles sin `usar`. Los parámetros después de `=`
son opcionales. Las expresiones de argumentos se evalúan una vez, en orden fuente,
también al usar argumentos nombrados.

```hacha
juego.configuracion(ancho num = 320, alto num = 180, titulo cadena = "Hacha", escala num = 1, redimensionable bool = falso, pantalla_completa bool = falso, tps num = 60)
Vec2.longitud() num
Vec2.normalizado() Vec2
Vec2.distancia_a(otro Vec2) num
Vec2.producto_punto(otro Vec2) num
Vec2.rotado(angulo num) Vec2
Vec2.colision_circulo(radio num, otro Vec2, radio_otro num) bool
Rect.interseca(otro Rect) bool
Rect.contiene(punto Vec2) bool
mate.pi // constante num
mate.absoluto(valor num) num
mate.minimo(a num, b num) num
mate.maximo(a num, b num) num
mate.limitar(valor num, minimo num, maximo num) num
mate.interpolar(a num, b num, t num) num
mate.piso(valor num) num
mate.techo(valor num) num
mate.redondear(valor num) num
mate.raiz(valor num) num
mate.seno(angulo num) num
mate.coseno(angulo num) num
mate.atan2(y num, x num) num
azar.real(minimo num, maximo num) num
azar.entero(minimo num, maximo num) num
color.rgba(r num, g num, b num, a num = 255) Color

graficos.limpiar(color Color)
graficos.rectangulo(x num, y num, ancho num, alto num, color Color, origen Vec2 = {}, rotacion num = 0)
graficos.rectangulo_v(pos Vec2, tamano Vec2, color Color, origen Vec2 = {}, rotacion num = 0)
graficos.rectangulo_rect(rect Rect, color Color, origen Vec2 = {}, rotacion num = 0)
graficos.circulo(x num, y num, radio num, color Color)
graficos.circulo_v(centro Vec2, radio num, color Color)
graficos.linea(x1 num, y1 num, x2 num, y2 num, color Color, grosor num = 1)
graficos.linea_v(a Vec2, b Vec2, color Color, grosor num = 1)
graficos.imagen(imagen Imagen, x num, y num, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)
graficos.imagen_v(imagen Imagen, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)
graficos.imagen_rect(imagen Imagen, destino Rect, origen Vec2 = {}, rotacion num = 0, tinte Color = .Blanco)
graficos.region(imagen Imagen, fuente Rect, x num, y num, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)
graficos.region_v(imagen Imagen, fuente Rect, pos Vec2, origen Vec2 = {}, escala Vec2 = {x: 1, y: 1}, rotacion num = 0, tinte Color = .Blanco)
graficos.region_rect(imagen Imagen, fuente Rect, destino Rect, origen Vec2 = {}, rotacion num = 0, tinte Color = .Blanco)
graficos.texto(texto cadena, fuente Fuente, x num, y num, tamano num = 20, color Color = .Blanco, origen Vec2 = {}, rotacion num = 0)
graficos.texto_v(texto cadena, fuente Fuente, pos Vec2, tamano num = 20, color Color = .Blanco, origen Vec2 = {}, rotacion num = 0)
graficos.texto_depuracion(texto cadena, x num = 0, y num = 0)
graficos.texto_depuracion_v(texto cadena, pos Vec2 = {})
graficos.tamano() Vec2
graficos.usar_camara(camara Camara2D)
graficos.restablecer_camara()

entrada.tecla_mantenida(tecla Tecla) bool
entrada.tecla_presionada(tecla Tecla) bool
entrada.tecla_soltada(tecla Tecla) bool
entrada.raton_mantenido(boton BotonRaton) bool
entrada.raton_presionado(boton BotonRaton) bool
entrada.raton_soltado(boton BotonRaton) bool
entrada.posicion_raton() Vec2
entrada.rueda() Vec2

audio.reproducir(sonido Sonido, volumen num = 1, bucle bool = falso) Reproduccion
audio.pausar(reproduccion Reproduccion)
audio.reanudar(reproduccion Reproduccion)
audio.detener(reproduccion Reproduccion)
audio.volumen(reproduccion Reproduccion, valor num)
ventana.titulo(titulo cadena)
ventana.pantalla_completa(activa bool)
ventana.tamano() Vec2
tiempo.fps() num
tiempo.tps() num
recursos.imagen(ruta cadena) Imagen
recursos.fuente(ruta cadena) Fuente
recursos.sonido(ruta cadena) Sonido
```

`azar` incluye el mínimo y excluye el máximo cuando son distintos; límites iguales
devuelven ese valor. Los límites deben ser finitos y ordenados; para `entero`
deben ser enteros. Rectángulo–rectángulo requiere área solapada, círculo–círculo
incluye contacto y punto–rectángulo incluye arriba/izquierda, excluye abajo/derecha.

Las formas son rellenas. La región de sprite usa píxeles de la imagen original.
La cámara traslada por `-pos`, rota por `-rotacion`, escala por `zoom` y traslada
por `origen`; su zoom debe ser positivo. Cada frame restablece la cámara.
`limpiar` y `texto_depuracion` usan pantalla; el ratón usa coordenadas lógicas de
pantalla. Las imágenes con posición/escala aplican su origen antes de escala/rotación/posición.
Las variantes `_rect` de imagen y región ajustan la imagen o recorte completo al
rectángulo destino; su origen se mide en píxeles del destino, después de escalar.
Rectángulos y texto aplican origen, rotación y posición antes de la cámara.
Todas las rotaciones son radianes y valen cero por defecto. Círculos, líneas y
texto de depuración no tienen rotación. Los métodos de Vec2 devuelven valores
nuevos y no mutan el receptor; normalizar cero devuelve cero.

El compilador rechaza dibujo alcanzable desde globales,
inicialización o actualización, incluyendo helpers, métodos y defaults.
Para interfaces comprueba conservadoramente los métodos con ese nombre.
Los helpers llamados desde `pintar` pueden dibujar; el runtime también verifica
el contexto. Lea los estados presionado/soltado en `actualizar`: corresponden al tick.

Audio retiene las reproducciones activas aunque se descarte el handle. Al terminar
libera el reproductor; `reanudar` reinicia un sonido terminado/detenido. Volumen 0–1.

## Comandos y alcance

`hacha compilar juego.hacha -o juego.go` produce Go formateado y sintácticamente
válido. `hacha ejecutar juego.hacha` construye y ejecuta. `hacha construir
juego.hacha -o juego.exe` construye sin abrir ventana; sin `-o` usa el nombre del
archivo sin extensión (`.exe` en Windows). Ambos usan un módulo temporal, eliminado
al finalizar. La primera compilación requiere red para descargar dependencias.

Se fija Ebitengine v2.10.1. Go 1.25+ y las dependencias nativas de Ebitengine son
necesarios. `HACHA_GO` permite elegir Go. La entrega inicial soporta escritorio;
web, móvil, gamepads, touch, física completa, tilemaps, partículas, UI y shaders
quedan fuera. Los tipos y namespaces están reservados; la interfaz genérica de
los ejemplos antes llamada `Fuente` ahora se llama `Proveedor`.
