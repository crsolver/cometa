# Interfaces con Pincel

`usar std/pincel/ui` ofrece controles de interfaz en modo inmediato: paneles, botones, casillas, deslizadores, texto, imágenes, barras de progreso, separadores y un campo de texto. Requiere Pincel. No hay estado retenido oculto: el `Contexto` vive en un campo del propio tipo del juego, y cada control recibe su valor actual y devuelve el nuevo.

## Contexto y ciclo de un cuadro

```cometa
tipo Juego
	contexto ui.Contexto

	pub fn actualizar(dt decimal)
		con ui.cuadro(@contexto)
			...

	pub fn pintar()
		ui.pintar(@contexto)
```

`ui.crear()` construye un `Contexto` (guárdalo en un campo, por ejemplo al inicializar el tipo). `con ui.cuadro(@contexto)` abre un cuadro de interfaz; dentro se declaran contenedores y controles. Solo se permite un `ui.cuadro` por `actualizar` y no se pueden anidar dos cuadros — ambas cosas producen un error en tiempo de ejecución si se intentan. `ui.pintar(contexto)` dibuja el árbol calculado en el cuadro anterior; se llama desde `pintar()`.

`ui.captura_raton(contexto)` y `ui.captura_teclado(contexto)` indican si la interfaz está usando el ratón o el teclado este cuadro, útil para no mover al jugador mientras se escribe en un campo o se hace clic en un botón.

## Medidas

| Función | Significado |
| --- | --- |
| `fijo(valor)` | Tamaño exacto en píxeles |
| `contenido(minimo = 0, maximo = ...)` | Se ajusta a su contenido, dentro de esos límites |
| `expandir(minimo = 0, maximo = ...)` | Reparte el espacio libre del contenedor |
| `porcentaje(fraccion)` | Fracción del tamaño del padre (0 a 1) |

`porcentaje` no puede usarse en el mismo eje que un padre en modo `contenido`, porque no hay tamaño de contenido del que tomar la fracción. `expandir` tampoco aporta tamaño natural a un padre en modo `contenido` en ese eje (su tamaño natural para ese cálculo es su mínimo, 0 salvo que se indique): un contenedor con un solo hijo `expandir()` y `ancho = contenido()` colapsa a ancho 0 sin ningún error. Dale al contenedor un `ancho`/`alto` explícito (`fijo` o `expandir`) siempre que alguno de sus hijos use `expandir` en ese eje.

## Contenedores

`fila`, `columna` y `panel` comparten firma: `id, ancho, alto, espacio = -1 (hereda del tema), relleno = {}, horizontal = .Inicio, vertical = .Inicio, desplazar = falso`. Se abren con `con`:

```cometa
con ui.panel("opciones", ancho = ui.fijo(280), alto = ui.fijo(160), relleno = {8, 8, 8, 8}, espacio = 8, desplazar = verdadero)
	...
```

Con `desplazar = verdadero`, la rueda del ratón desplaza el contenido cuando el puntero está encima; el desplazamiento se recuerda por `id` entre cuadros.

## Widgets

| Función | Resultado |
| --- | --- |
| `texto(valor, ancho, alto, color = .Transparente)` | Texto no interactivo, se ajusta a varias líneas. `color = .Transparente` (el valor por defecto) hereda `tema.texto`; cualquier otro color lo sustituye solo para ese texto |
| `imagen(imagen, ancho, alto, tinte = .Blanco)` | Dibuja una `graficos.Imagen` |
| `boton(id, etiqueta, ancho, alto, habilitado = verdadero) bool` | `verdadero` el cuadro en que se suelta el clic (o se activa por teclado) |
| `casilla(id, valor, etiqueta = "", ancho, alto, habilitado = verdadero) bool` | Devuelve el nuevo valor |
| `deslizador(id, valor, minimo = 0, maximo = 1, ancho, alto, habilitado = verdadero) decimal` | Devuelve el nuevo valor, ya limitado |
| `barra(valor, minimo = 0, maximo = 1, ancho, alto)` | Barra de progreso no interactiva; `valor` se limita a `[minimo, maximo]` |
| `separador(ancho = expandir(), alto = fijo(1))` | Línea divisoria delgada, sin interacción |
| `espacio(ancho = expandir(), alto = expandir())` | Hueco invisible que reparte espacio libre dentro de una fila o columna |
| `campo_texto(id, valor, ancho = contenido(0, 200), alto = contenido(), habilitado = verdadero) cadena` | Campo de una línea; devuelve el texto editado |

Todos los controles (`boton`, `casilla`, `deslizador`, `campo_texto`) son componentes controlados: reciben el valor actual y devuelven el que corresponde tras la interacción de este cuadro. El identificador (`id`) debe ser único entre hermanos; repetirlo detiene el programa.

### Grupos de opciones (radio)

No hay una primitiva de grupo de opciones: se compone con `boton`, comparando el valor propio del programa.

```cometa
con ui.columna("clases", ancho = ui.expandir(), espacio = 4)
	repetir (0..3) |i|
		etiqueta = si @clase == i "(x) " + nombres[i] sino "( ) " + nombres[i]
		si ui.boton("clase_${i}", etiqueta, ancho = ui.expandir())
			@clase = i
```

### Limitaciones de `campo_texto`

No admite selección de texto, clic para posicionar el cursor, ni pegar desde el portapapeles; ver `MEJORAS.md`.

## Foco y teclado

`Tab`/`Mayús+Tab` mueven el foco entre controles habilitados; `Enter` o `Espacio` activan el control con foco (equivalente a hacer clic). El resto de teclas dependen de qué control tiene el foco:

| Control | Teclas |
| --- | --- |
| `deslizador` | Flecha izquierda/derecha: ajusta el valor en pasos de 1% |
| `campo_texto` | Escribir inserta texto; flecha izquierda/derecha mueve el cursor; `Inicio`/`Fin`; `Retroceso`/`Suprimir` borran |

Como solo un control tiene el foco a la vez, `campo_texto` y `deslizador` pueden reutilizar las mismas teclas de flecha sin interferir entre sí.

## Temas

`ui.retro(escala = 1) Tema` da un tema con texto en la fuente retro integrada, a la escala pedida. `ui.fuente(fuente, tamano = 20) Tema` usa una `Fuente` TrueType cargada con `recursos`. `con ui.tema(tema)` aplica un tema a un bloque; se puede anidar.

| Campo de `Tema` | Uso |
| --- | --- |
| `fondo` | Relleno de paneles y del área del cuadro |
| `normal`, `sobre`, `presionado`, `deshabilitado` | Relleno de controles según su estado |
| `foco` | Borde de 1px cuando el control tiene el foco de teclado |
| `texto` | Color del texto y del relleno de la barra de progreso |
| `espacio` | Espaciado por defecto entre hijos (`espacio = -1` en un contenedor lo hereda) |

## Limitaciones

- Un `ui.cuadro` por `actualizar`; no se pueden anidar cuadros.
- No hay menú desplegable ni ningún control con lista emergente: la interfaz no tiene un mecanismo de superposición fuera del árbol normal.
- `campo_texto` no admite selección de texto ni portapapeles (ver arriba).

## Ejemplo

Consulta [el ejemplo de creación de personaje](../examples/pincel/10_ui.cometa): nombre con `campo_texto`, clase con un grupo de opciones compuesto, y una barra de progreso de vida.

```console
vscode-extension/bin/cometa ejecutar examples/pincel/10_ui.cometa
```
