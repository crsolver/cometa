---
titulo: "Tu primer juego"
orden: 14
resumen: "Construye paso a paso un juego 2D con Pincel: ventana, movimiento, colisiones y puntos."
ejemplo: "pincel/04_teclado.cometa"
---

Pincel es el motor de juegos 2D de la biblioteca estándar. Vamos a construir un juego pequeño por pasos: un cuadrado que se mueve con el teclado y recoge monedas. Cada paso funciona por sí solo, y el final es el juego completo.

Para probarlo, guarda el código en un archivo (por ejemplo `juego.cometa`) y ejecuta `cometa ejecutar juego.cometa`. La primera compilación descarga las dependencias gráficas y tarda un poco.

## La idea: un tipo con dos métodos

Un juego es un `tipo` con dos métodos públicos que Pincel llama por ti, unas 60 veces por segundo:

- `actualizar(dt)` **cambia el estado**: mueve cosas, cuenta puntos. `dt` son los segundos desde la última vez.
- `pintar()` **dibuja** el estado actual. No debe cambiarlo.

Separar «qué pasa» de «cómo se ve» mantiene el código ordenado. Y `inicio()` solo entrega el juego a `pincel.ejecutar`, que abre la ventana y espera a que se cierre.

## Paso 1: una ventana

```cometa
usar std/pincel
usar std/pincel/graficos

tipo Juego
	pub fn actualizar(dt decimal) retornar

	pub fn pintar()
		graficos.limpiar(.AzulNoche)
		graficos.texto_depuracion("¡Hola desde Pincel!", 110, 80)

fn inicio()
	pincel.ejecutar(Juego {}, 320, 180, titulo = "Mi juego", escala = 3) atrapar |error|
		imprimir(error)
```

- Cada módulo se importa por separado: `std/pincel` (el motor) y `std/pincel/graficos` (dibujo).
- `320, 180` es el tamaño **lógico**: las coordenadas van de 0 a 319 en x y de 0 a 179 en y, con el `(0, 0)` **arriba a la izquierda** y la `y` creciendo hacia abajo. `escala = 3` solo agranda la ventana.
- `ejecutar` devuelve un resultado (lección 10), por eso se maneja con `atrapar |error|`.
- `actualizar` está vacío por ahora; `retornar` en una línea basta.

## Paso 2: estado y movimiento

El estado del juego vive en los campos del tipo. Añadimos la posición del jugador y la movemos con el teclado:

```cometa
usar std/mate
usar std/pincel/entrada

const velocidad decimal = 100

tipo Juego
	pos mate.Vec2

	pub fn actualizar(dt decimal)
		var direccion = mate.Vec2 {}
		si entrada.tecla_mantenida(.Derecha)
			direccion.x = 1
		si entrada.tecla_mantenida(.Izquierda)
			direccion.x = -1
		si entrada.tecla_mantenida(.Abajo)
			direccion.y = 1
		si entrada.tecla_mantenida(.Arriba)
			direccion.y = -1

		@pos = @pos + direccion.normalizado() * velocidad * dt

	pub fn pintar()
		graficos.limpiar(.VerdePino)
		graficos.rectangulo_v(@pos, {16, 16}, .Celeste)
```

- `mate.Vec2` es un vector `{x, y}`. Se suma y se multiplica por un número directamente.
- `tecla_mantenida` es verdadero mientras la tecla está pulsada. (`tecla_presionada` solo lo es en el cuadro en que se pulsa: útil para saltar o disparar.)
- **Multiplicar por `dt`** hace que la velocidad sea «100 píxeles por segundo» sin importar cuántos cuadros se dibujen. Sin `dt`, el juego iría más rápido en un ordenador más rápido.
- `normalizado()` deja la dirección con longitud 1, para que ir en diagonal no sea más rápido.

Empezamos en la esquina porque el `Juego {}` de `inicio` deja `pos` en `{0, 0}`. Para otro punto de partida: `Juego {pos: {x: 152, y: 82}}`.

## Paso 3: una moneda y colisiones

Ahora una moneda que aparece en un lugar al azar. Al tocarla sumamos un punto y la movemos. Para saber si dos rectángulos se tocan usamos `mate.Rect` y `interseca`:

```cometa
usar std/azar

tipo Juego
	pos mate.Vec2
	moneda mate.Vec2
	puntos entero

	fn mover_moneda()
		@moneda = {x: azar.entero(0, 312), y: azar.entero(0, 172)}

	pub fn actualizar(dt decimal)
		// ... el movimiento del paso 2 ...

		var caja_jugador = mate.Rect {pos: @pos, tamano: {16, 16}}
		var caja_moneda = mate.Rect {pos: @moneda, tamano: {8, 8}}
		si caja_jugador.interseca(caja_moneda)
			@puntos += 1
			@mover_moneda()
```

- Los métodos sin `pub` (`mover_moneda`) son internos del tipo; Pincel solo necesita `actualizar` y `pintar`.
- `azar.entero(0, 312)` da un entero de 0 a 311 (el final se excluye, como en los rangos).
- Recuerda `@` (lección 6): `@moneda` y `@mover_moneda()` son el campo y el método **de este juego**.

## Paso 4: el juego completo

Unimos todo, mantenemos al jugador dentro de la pantalla y mostramos los puntos. Esto es un programa completo que puedes copiar y ejecutar:

```cometa
usar std/mate
usar std/azar
usar std/pincel
usar std/pincel/graficos
usar std/pincel/entrada

const velocidad decimal = 100

tipo Juego
	pos mate.Vec2
	moneda mate.Vec2
	puntos entero

	fn mover_moneda()
		@moneda = {x: azar.entero(0, 312), y: azar.entero(12, 172)}

	pub fn actualizar(dt decimal)
		var direccion = mate.Vec2 {}
		si entrada.tecla_mantenida(.Derecha) || entrada.tecla_mantenida(.D)
			direccion.x = 1
		si entrada.tecla_mantenida(.Izquierda) || entrada.tecla_mantenida(.A)
			direccion.x = -1
		si entrada.tecla_mantenida(.Abajo) || entrada.tecla_mantenida(.S)
			direccion.y = 1
		si entrada.tecla_mantenida(.Arriba) || entrada.tecla_mantenida(.W)
			direccion.y = -1

		@pos = @pos + direccion.normalizado() * velocidad * dt
		@pos.x = mate.limitar(@pos.x, 0, 304)
		@pos.y = mate.limitar(@pos.y, 12, 164)

		var caja_jugador = mate.Rect {pos: @pos, tamano: {16, 16}}
		var caja_moneda = mate.Rect {pos: @moneda, tamano: {8, 8}}
		si caja_jugador.interseca(caja_moneda)
			@puntos += 1
			@mover_moneda()

	pub fn pintar()
		graficos.limpiar(.VerdePino)
		graficos.rectangulo_v(@moneda, {8, 8}, .Amarillo)
		graficos.rectangulo_v(@pos, {16, 16}, .Celeste)
		graficos.texto_depuracion("Puntos: ${@puntos}", 4, 2)

fn inicio()
	var juego = Juego {pos: {x: 152, y: 82}}
	juego.mover_moneda()
	pincel.ejecutar(juego, 320, 180, titulo = "Monedas", escala = 3) atrapar |error|
		imprimir(error)
```

`mate.limitar(valor, minimo, maximo)` mantiene un número dentro de un rango. Fíjate en que dibujamos la moneda antes que al jugador: **lo que se dibuja después queda encima**.

## Ideas para seguir

- Añade un tiempo límite: un campo `restante decimal` al que le restas `dt` en `actualizar`, y muestra «Fin» cuando llegue a 0 (`examples/pincel/09_atrapa.cometa` lo hace, con imagen y sonido).
- Cambia los cuadrados por sprites dibujados con texto usando `lienzo` (ejemplo `07_sprites`).
- Usa `retro.texto` y `retro.icono` para texto e iconos de aspecto retro sin archivos de imagen.

Sigue con las guías: [juegos](/guias/juegos/), [pixel art con lienzo](/guias/lienzo/), [interfaces](/guias/ui/) y [curvas de animación](/guias/curvas/); y explora todos los módulos en la [biblioteca](/biblioteca/).

## Ejemplo: mover un personaje
