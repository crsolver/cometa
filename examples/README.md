# Ejemplos de Cometa

¿Primera vez? Empieza por la [guía «Empezar»](../docs/empezar.md). Aquí hay programas cortos para aprender Cometa paso a paso. Cada archivo se ejecuta solo y está comentado en español. Síguelos en orden: cada uno usa lo aprendido en los anteriores.

Desde la raíz del repositorio (en Windows usa `vscode-extension\bin\cometa.exe`):

```console
vscode-extension/bin/cometa ejecutar examples/basico/01_hola.cometa
```

## Básico

Programas de consola: solo imprimen texto.

| Archivo | Aprenderás |
| --- | --- |
| [01_hola](basico/01_hola.cometa) | La función `inicio` e `imprimir` |
| [02_variables](basico/02_variables.cometa) | `var`, `const`, tipos básicos, interpolación, operaciones y conversiones |
| [03_condiciones](basico/03_condiciones.cometa) | `si`, `osi`, `sino`, comparaciones y `casos` con valores |
| [04_bucles](basico/04_bucles.cometa) | `repetir` con rangos y listas, `mientras`, `+=`, `continuar` y `romper` |
| [05_funciones](basico/05_funciones.cometa) | Parámetros, resultados, valores predeterminados y argumentos con nombre |
| [06_listas](basico/06_listas.cometa) | Crear, modificar y recorrer listas |
| [07_mapas](basico/07_mapas.cometa) | Mapas clave-valor y contar palabras |
| [08_tipos](basico/08_tipos.cometa) | Tipos propios con campos y métodos (`@`) y `copiar()` |
| [09_enums](basico/09_enums.cometa) | `enum`, `casos` y `==` con variantes |
| [10_errores](basico/10_errores.cometa) | `T!`, `.Ok`/`.Error`, `o`, `atrapar` e `intentar` |
| [11_interfaces](basico/11_interfaces.cometa) | Interfaces: tipos distintos con el mismo comportamiento |
| [12_modulos](basico/12_modulos.cometa) | Repartir el código en archivos con `usar` y `pub` |
| [13_dados](basico/13_dados.cometa) | Programa completo con números aleatorios |

## Pincel: juegos 2D

Cada ejemplo abre una ventana. La primera compilación descarga las dependencias gráficas y puede tardar un poco.

| Archivo | Aprenderás |
| --- | --- |
| [01_ventana](pincel/01_ventana.cometa) | La estructura de un juego: `actualizar`, `pintar` y `pincel.ejecutar` |
| [02_formas](pincel/02_formas.cometa) | Rectángulos, círculos, líneas y colores |
| [03_rebote](pincel/03_rebote.cometa) | Movimiento con velocidad y `dt` |
| [04_teclado](pincel/04_teclado.cometa) | Mover un personaje con el teclado |
| [05_raton](pincel/05_raton.cometa) | Posición y clics del ratón |
| [06_retro](pincel/06_retro.cometa) | Texto e iconos retro sin archivos de imagen |
| [07_sprites](pincel/07_sprites.cometa) | Dibujar sprites animados con texto usando `lienzo` |
| [08_curvas](pincel/08_curvas.cometa) | Animaciones suaves con curvas |
| [09_atrapa](pincel/09_atrapa.cometa) | Un juego completo con imagen, sonido, puntos y tiempo |
| [10_ui](pincel/10_ui.cometa) | Interfaces en modo inmediato: campo de texto, barra de progreso y grupos de opciones |
| [11_mapa](pincel/11_mapa.cometa) | Mapas de tiles con `rejilla`, hojas de sprites, colisiones, mandos y `datos` para guardar la mejor marca |
| [12_escenas](pincel/12_escenas.cometa) | Menú, juego y pantalla final con un `enum` de escenas y `casos` |
| [13_plataformas](pincel/13_plataformas.cometa) | Plataformas con gravedad, salto, colisiones contra tiles, monedas y pinchos |
| [14_ladrillos](pincel/14_ladrillos.cometa) | Rompe-ladrillos: rebotes bola-rectángulo, lista de ladrillos y estados con `enum` |
| [15_mazmorra](pincel/15_mazmorra.cometa) | Roguelike por turnos en dos archivos: generador de mazmorras con semilla (`mazmorra/nivel.cometa`), enemigos y escenas |

Para guardar una captura de un juego sin abrir la ventana:

```console
vscode-extension/bin/cometa captura examples/pincel/02_formas.cometa -o formas.png --escala 2
```

## Más información

- [Guía de Pincel](../docs/juegos.md), [lienzo](../docs/lienzo.md) e [interfaces](../docs/ui.md)
- [Especificación del lenguaje](../specs.md)
