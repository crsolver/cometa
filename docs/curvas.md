# Curvas de animación

`usar std/mate/curvas` ofrece 31 funciones puras para transformar el progreso de una animación. El alias predeterminado es `curvas`; también se permite `usar std/mate/curvas como c`. No requiere Pincel ni Ebitengine y no importa automáticamente `std/mate`.

Todas las funciones tienen la firma `(progreso decimal) decimal`. `lineal` mantiene el progreso. Las demás combinan uno de los siguientes nombres con `_entrada`, `_salida` o `_entrada_salida`:

| Familia | Forma |
| --- | --- |
| `cuadratica` | Potencia de grado 2 |
| `cubica` | Potencia de grado 3 |
| `cuartica` | Potencia de grado 4 |
| `quintica` | Potencia de grado 5 |
| `senoidal` | Transición sinusoidal |
| `circular` | Arco circular |
| `exponencial` | Transición exponencial |
| `elastica` | Oscilación elástica |
| `retroceso` | Retroceso y sobrepaso |
| `rebote` | Rebotes sucesivos |

Por ejemplo: `cubica_entrada`, `cubica_salida`, `cubica_entrada_salida`. Entrada concentra el efecto al inicio; salida lo refleja al final; entrada-salida aplica ambas mitades. Las fórmulas siguen las curvas de [Odin core:math/ease](https://pkg.odin-lang.org/core/math/ease/), originadas en AHEasing; Hacha añade límites de entrada y extremos exactos.

Para `progreso <= 0` el resultado es exactamente `0`; para `progreso >= 1` es exactamente `1`. También se limitan los infinitos; NaN se propaga. **El resultado no se limita**: `elastica` y `retroceso` pueden producir valores menores que 0 o mayores que 1 dentro del intervalo. Esto conserva el movimiento característico de esas curvas.

## Uso con tiempo e interpolación

```hacha
usar std/mate
usar std/mate/curvas

fn posicion(transcurrido decimal, duracion decimal) decimal
	mate.interpolar(20, 140, curvas.cubica_entrada_salida(transcurrido / duracion))
```

Usa una duración positiva. El programa acumula tiempo, por ejemplo sumando `dt` desde `actualizar(dt decimal)`, y divide por la duración para obtener progreso. Cuando se supera la duración, la curva devuelve 1 y la interpolación conserva el destino. Para detectar la finalización, compara el tiempo con la duración; una curva con sobrepaso puede cruzar el destino antes de terminar.

La biblioteca no mantiene relojes, objetos tween, callbacks ni colas. No incluye funciones inversas ni un selector de curvas. Las funciones se pueden consultar desde el editor mediante completar, hover y definición en `hacha-std:///std/mate/curvas.hacha`.

El [ejemplo completo](../examples/curvas.hacha) imprime una animación de 20 a 140 sin abrir una ventana:

```console
hacha ejecutar examples/curvas.hacha
```

Para ver las curvas en movimiento, ejecuta la [comparación con Pincel](../examples/curvas_pincel.hacha). Seis filas comparten el reloj y los extremos; los puntos grises indican el movimiento lineal. La animación va y vuelve automáticamente. Espacio pausa y R reinicia. No necesita archivos de recursos externos.

```console
hacha ejecutar examples/curvas_pincel.hacha
```
