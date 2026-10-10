
## Juegos con Pincel

Este proyecto es un juego 2D con **Pincel** (biblioteca estándar). Guía a la persona para que entienda el ciclo de un juego: estado, actualizar, pintar.

- Cada archivo importa de forma explícita lo que usa: `usar std/pincel` (núcleo: `pincel.ejecutar`), y módulos sueltos como `std/pincel/graficos`, `color`, `entrada`, `audio`, `tiempo`, `recursos`, `rejilla`, `lienzo`, `retro`, `ui`. Además `std/mate` y `std/azar`. `std/pincel` no importa los submódulos.
- Los tipos se califican: `mate.Vec2`, `color.Color`, `graficos.Imagen`.
- Un juego es un objeto con `pub fn actualizar(dt decimal)` (lógica, ~60 veces por segundo; `dt` son segundos) y `pub fn pintar()` (dibujar). Solo se dibuja dentro de `pintar`.
- Para dibujar en una imagen en vez de en la pantalla (minimapas, escenas a baja resolución, fondos pintados una vez): `con graficos.pintar_en(imagen)` con una imagen de `lienzo.nuevo(ancho, alto)`; dentro del bloque se usan las mismas funciones de `graficos` y después la imagen se dibuja con `graficos.imagen`.
- Se arranca en `inicio()` con `pincel.ejecutar(juego, ancho, alto, titulo = "...", escala = 3) atrapar |error| ...`.
- Para ver el resultado sin abrir la ventana: `cometa captura principal.cometa -o captura.png --escala 4`. Para ver escenas posteriores sin tocar el juego, usa un guion de entrada y varios cuadros: `cometa captura principal.cometa --entrada guion.txt --cuadros 100,300` (líneas como `60 +D`, `90 -D`, `100 Enter`, `120 raton 160 90`, `121 RatonIzquierdo`, `130 texto "hola"`, `140 rueda 0 -1`, `150 +MandoA`).
- Un juego no necesita archivos: `lienzo` y `retro` crean los gráficos y `audio.tono(frecuencia, duracion)` / `audio.ruido(duracion)` los sonidos (se guardan en una variable global y se reproducen con `audio.reproducir(sonido, tono = 1.2)`).
- Consulta la API real con `cometa biblioteca std/pincel/graficos` (y los demás módulos) antes de usarla.

Ejemplo mínimo:

```cometa
usar std/pincel
usar std/pincel/graficos

tipo Juego
	x decimal

	pub fn actualizar(dt decimal)
		@x = @x + 50 * dt

	pub fn pintar()
		graficos.limpiar(.Negro)
		graficos.rectangulo(@x, 40, 16, 16, .Amarillo)

fn inicio()
	pincel.ejecutar(Juego {}, 320, 180) atrapar |error|
		imprimir(error)
```
