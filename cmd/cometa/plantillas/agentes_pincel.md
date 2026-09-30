
## Juegos con Pincel

Este proyecto es un juego 2D con **Pincel** (biblioteca estándar). Guía a la persona para que entienda el ciclo de un juego: estado, actualizar, pintar.

- Cada archivo importa de forma explícita lo que usa: `usar std/pincel` (núcleo: `pincel.ejecutar`), y módulos sueltos como `std/pincel/graficos`, `color`, `entrada`, `audio`, `tiempo`, `recursos`, `rejilla`, `lienzo`, `retro`, `ui`. Además `std/mate` y `std/azar`. `std/pincel` no importa los submódulos.
- Los tipos se califican: `mate.Vec2`, `color.Color`, `graficos.Imagen`.
- Un juego es un objeto con `pub fn actualizar(dt decimal)` (lógica, ~60 veces por segundo; `dt` son segundos) y `pub fn pintar()` (dibujar). Solo se dibuja dentro de `pintar`.
- Se arranca en `inicio()` con `pincel.ejecutar(juego, ancho, alto, titulo = "...", escala = 3) atrapar |error| ...`.
- Para ver el resultado sin abrir la ventana: `cometa captura principal.cometa -o captura.png --escala 4`.
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
