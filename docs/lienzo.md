# Lienzo: pixel art con código

`usar std/pincel/lienzo` crea y edita imágenes (`graficos.Imagen`) píxel a píxel, sin archivos PNG. Las imágenes resultantes se dibujan con `graficos.imagen`, `graficos.region`, etc., como cualquier imagen cargada.

- Las coordenadas y tamaños son `entero`; no hay suavizado.
- Los píxeles viven en memoria de CPU y se suben a la GPU al dibujarlos, así que los lienzos pueden crearse en variables globales o en `inicio()`, antes de `pincel.ejecutar`. Editar un lienzo que ya se dibujó vuelve a subirlo en el siguiente dibujo.
- Las primitivas **reemplazan** píxeles, incluida la transparencia (sirven también como borrador). Solo `pegar` mezcla la fuente sobre el destino.
- Escribir fuera de los bordes se recorta sin error.

## Sprites desde texto

```cometa
usar std/pincel/lienzo
usar std/pincel/color

var colores = [
	"o": color.rgba(24, 20, 37),
	"s": color.rgba(234, 170, 130),
	"b": .Azul,
]

var cara = lienzo.desde_texto([
	".oooo.",
	"osssso",
	"osoos.",
	".obbo.",
], colores)
```

| Función | Resultado |
| --- | --- |
| `desde_texto(filas [cadena], colores [cadena: Color]) Imagen` | Una fila de texto por fila de píxeles |
| `hoja_desde_texto(cuadros [[cadena]], colores [cadena: Color]) [Imagen]` | Cuadros de animación del mismo tamaño |

Cada clave del mapa es un solo carácter. `.` y el espacio son transparentes salvo que el mapa los redefina. Un carácter sin color, filas de distinto ancho, una lista vacía o cuadros de distinto tamaño detienen el programa con un mensaje que indica la fila y el cuadro.

## Lienzo y primitivas

| Función | Efecto |
| --- | --- |
| `nuevo(ancho, alto, color = .Transparente) Imagen` | Lienzo de 1 a 16384 píxeles por lado |
| `ancho(imagen) entero`, `alto(imagen) entero` | Tamaño en píxeles |
| `copiar(imagen) Imagen` | Copia independiente |
| `limpiar(imagen, color = .Transparente)` | Rellena todo el lienzo |
| `pixel(imagen, x, y, color)` | Un píxel |
| `leer_pixel(imagen, x, y) Color?` | `.Ninguno` fuera de los bordes |
| `rect(imagen, x, y, ancho, alto, color, relleno = verdadero)` | Rectángulo o su borde de 1 píxel |
| `linea(imagen, x1, y1, x2, y2, color)` | Línea de Bresenham, extremos incluidos |
| `circulo(imagen, x, y, radio, color, relleno = verdadero)` | Disco de píxeles con `dx² + dy² <= r² + r`; sin relleno, su borde 4-conexo |
| `rellenar(imagen, x, y, color)` | Relleno por inundación 4-conexo del color exacto |
| `pegar(destino, fuente, x, y, espejo_h = falso, espejo_v = falso)` | Copia con mezcla alfa; los píxeles opacos se copian exactos y los transparentes se ignoran |
| `guardar(imagen, ruta, escala = 1) !` | Escribe un PNG, ampliado por vecino más cercano (escala 1–64) |

`guardar` devuelve un resultado que debe manejarse, por ejemplo con `atrapar |error| imprimir(error)`. Las rutas relativas parten del directorio de trabajo del proceso.

## Ver el resultado

```console
cometa ejecutar sprites.cometa                     # un programa que llama a lienzo.guardar
cometa captura juego.cometa -o juego.png --escala 4 --cuadros 30
```

`cometa captura` ejecuta el juego con la ventana oculta, guarda la pantalla lógica tras `--cuadros` actualizaciones (1 por defecto) y cierra el juego. `--escala` amplía el PNG sin suavizado. Sin `-o`, escribe `<archivo>.png` junto a la fuente. Requiere que el programa llame a `pincel.ejecutar`; la captura ocurre antes del efecto `retro`.

Con varios cuadros separados por comas (`--cuadros 100,300,900`) se toman todas las capturas en una sola ejecución y se guardan como `juego_100.png`, `juego_300.png`, etc.

### Simular la entrada

Para llegar a otra sala, a un jefe o a la pantalla final sin tocar el código del juego, `--entrada` lee un guion de teclas por cuadro:

```console
cometa captura juego.cometa --entrada guion.txt --cuadros 100,300
```

```text
# cuadro  acciones
60  +D              mantiene D desde el cuadro 60
90  -D              la suelta
100 Enter           la pulsa durante un cuadro
120 raton 160 90    mueve el ratón (píxeles de la pantalla lógica)
121 RatonIzquierdo  clic
150 +D +Espacio     varias acciones en la misma línea
```

Las teclas usan los nombres de `entrada.Tecla` (`D`, `Enter`, `Izquierda`, `Espacio`…) y los botones del ratón llevan delante `Raton` (`RatonIzquierdo`, `RatonDerecho`, `RatonMedio`). Mientras corre el guion, el teclado y el ratón reales se ignoran, así que la ejecución es siempre la misma; `std/pincel/ui` también responde al guion, salvo la escritura de texto. Sin `--cuadros`, la captura se toma un cuadro después de la última acción.

Consulta [el ejemplo de sprites](../examples/pincel/07_sprites.cometa): un personaje de dos cuadros dibujado con texto y animado al caminar.
