# Cambios

Las notas de cada versión. La sección de la etiqueta publicada se usa como texto de la página de versiones.

## v0.1.1

### Cambios que pueden afectar a tus programas

- **Las listas se comparten por completo, como los mapas:** asignar una lista a otra variable, guardarla en un campo o pasarla a una función comparte sus elementos y su longitud. `agregar`, `insertar` y `eliminar` dentro de una función cambian la lista de quien la llamó; antes ese cambio no se veía fuera. Para tener una lista independiente se usa `copiar()`.
- **Un rango ya no cuenta hacia atrás por su cuenta:** `repetir (a..b)` solo cuenta hacia adelante y no da ninguna vuelta si `a` no es menor que `b`, así que `repetir (0..lista.longitud() - 1)` no entra al cuerpo con una lista vacía. Para bajar se escribe el paso: `repetir (5..0, paso = -1)`. Los límites literales escritos al revés sin `paso` dan un error que lo sugiere.

### Lenguaje

- **`repetir (n)`:** con un `entero` da `n` vueltas contadas desde 0, sin necesidad de escribir un rango.
- **`repetir (a..b, paso = n)`:** cuenta a saltos (`paso = 2`) o hacia atrás (`paso = -1`). Un paso 0 es un error.
- **`repetir` sobre una `cadena`:** la recorre letra por letra.
- **Comparar con `==` opcionales, resultados y constantes:** `si buscado == .Ninguno`, `si elegido == .Alguno("Ana")`, `si guardar() != .Ok` y `si tecla == .Espacio` compilan sin `casos`. Las constantes de la biblioteca también pueden escribirse con su tipo (`retro.Icono.Arriba`).
- **Números en hexadecimal y binario:** `0xFF` y `0b1010`.
- **Métodos sobre un número escrito:** `7.formato(2)`.
- **Interpolar cualquier valor:** las listas, los mapas, las estructuras, los enums, los opcionales y los resultados se pueden poner en `"${...}"` y pasar a `cadena()`, con el mismo formato que `imprimir`.
- **`si` de una línea:** una rama puede empezar por `-`, `.` o `(`, como en un valor negativo o una `.Variante`.
- Una función con resultado puede terminar en un `repetir` infinito sin `romper`.

### Biblioteca

- **Listas:** `ordenar(descendente = falso)` ordena una lista de `entero`, `decimal` o `cadena`, y `ordenar_por(claves, descendente = falso)` ordena cualquier lista según una lista paralela de claves. Las dos son estables.
- **Cadenas:** `rellenar_izquierda(ancho, relleno)`, `rellenar_derecha(ancho, relleno)` y `repetir(veces)`. Un marcador con ceros es `cadena(puntos).rellenar_izquierda(7, "0")`.
- **`std/mate`:** operaciones de bits con `bit_y`, `bit_o`, `bit_xor`, `bit_no`, `desplazar_izquierda` y `desplazar_derecha`.
- **Sonidos sin archivos:** `audio.tono` y `audio.ruido` crean un sonido en memoria, y `audio.reproducir` acepta un parámetro `tono` que cambia la altura y la velocidad.
- **Colores:** `color.mezclar`, `Color.con_alfa`, los canales de lectura `r`, `g`, `b` y `a`, y `color.a_entero`/`color.desde_entero` para el formato `0xRRGGBB` de las rejillas.
- **`tiempo.ahora()`:** los segundos transcurridos desde que empezó el programa, también en programas de consola.
- **`Rejilla.buscar(valor)`:** las celdas que contienen un valor, para leer de una capa dónde empieza el jugador o dónde están las llaves.
- **`retro`:** más de 40 iconos nuevos (`Cara`, `Oro`, `Casa`, `Castillo`, `Gato`, `Manzana`…), y `retro.icono` y `retro.glifo` aceptan `voltear_h` y `voltear_v`.

### Pintar en una imagen

- **`con graficos.pintar_en(imagen)`:** lo que se dibuja dentro del bloque va a la imagen en vez de a la pantalla, sin cámara ni recorte, y al salir todo vuelve a como estaba. Sirve para minimapas, escenas a baja resolución que luego se amplían o fondos que se pintan una sola vez; el bloque puede usarse en cualquier momento, no solo desde `pintar`. Las funciones de `lienzo` ven lo pintado.
- **`graficos.recortar` con `con`:** `con graficos.recortar(rect)` quita el recorte al salir del bloque; los bloques anidados se limitan al que los contiene.
- **`graficos.liberar(imagen)`:** libera en el momento la copia de una imagen en la tarjeta gráfica; la imagen sigue siendo válida y vuelve a subirse al dibujarla.

### Entrada simulada

- **Pruebas:** `std/pruebas` puede simular la entrada: `pruebas.mantener`, `soltar` y `pulsar` (teclas), sus variantes `_raton` y `_mando`, `pruebas.raton(x, y)`, `pruebas.rueda(x, y)`, `pruebas.texto("...")` y `pruebas.eje_mando(eje, valor)`. Un ejercicio como «al pulsar D el jugador se mueve» ya se comprueba a través de `entrada`.
- **`cometa captura --entrada`:** el guion admite `texto "hola"`, `rueda x y`, botones del mando 0 (`+MandoA`) y palancas (`eje IzquierdoX 1`), y `std/pincel/ui` recibe el texto escrito.

### Editor de mapas

- **`cometa mapa`:** abre un editor de mapas de tiles en el navegador (`tools/mapa-editor`, un solo archivo HTML), con el atlas dungeon como paleta, un color por celda, capas, pintar, borrar, rellenar y rectángulo. Guarda y abre archivos `.mapa.txt` y exporta un módulo `.cometa` con los iconos y los colores de cada capa.
- **`rejilla.dibujar(..., colores = otra_rejilla)`:** una rejilla de colores (`0xRRGGBB` por celda) pinta cada celda de su color; sin ella se usa `tinte`, como antes.
- **`retro.cargar_mapa(ruta)` y `retro.mapa(mapa)`:** cargan un `.mapa.txt` y lo dibujan como se ve en el editor. `mapa.capa("suelo")` da la rejilla de una capa para consultar celdas, colisionar o cambiar el mapa mientras se juega.
- **Marcas con nombre:** la herramienta Marca del editor le pone un nombre a una celda (`jugador`, `enemigo`, `puerta`) sin dibujar nada en ella, y `mapa.marcas("enemigo")` devuelve la columna y la fila de cada celda con ese nombre. Ya no hace falta reservar un icono por cada cosa que el juego coloca. En el `.mapa.txt` son líneas `@ nombre columna,fila`.
- **`retro.hoja(atlas = .Dungeon)`:** el atlas retro como `graficos.Hoja`, para dibujar una rejilla con los iconos dungeon: `rejilla.dibujar(mapa, retro.hoja(), tinte = .Gris)`.
- **`rejilla.desde_csv(texto, capa = "")`:** crea una rejilla a partir del texto de un `.mapa.txt` (una fila por línea, valores separados por comas en decimal o `0x…`, `# nombre` antes de cada capa).
- **`recursos.texto(ruta)`:** incorpora un archivo de texto al programa, como `recursos.imagen` hace con las imágenes.

### Correcciones

- **Nombres que solo cambian en la mayúscula inicial:** una función `punto` y un tipo `Punto` ya pueden convivir (antes daban «error interno del compilador»). Dos campos, métodos o tipos que solo se distinguen así dan un error claro. Las variables llamadas como algo de Go (`len`, `string`, `copy`, `fmt`…) ya no rompen el programa.
- **`"${ valor }"`:** una interpolación puede llevar espacios alrededor de la expresión; antes daba un error de indentación.
- **Rutas en Windows:** los errores, los fallos en ejecución y «ir a la definición» muestran los archivos con sus mayúsculas, no la ruta entera en minúsculas.
- **Editor más ágil en juegos con recursos:** el editor ya no vuelve a leer y decodificar cada imagen y sonido con cada tecla; solo cuando el archivo cambia.
- **Go instalado dos veces:** si el primer `go` del `PATH` no funciona, Cometa prueba el siguiente en vez de fallar, y usa el del `PATH` antes que el de la carpeta del instalador.
- **`mientras` con una condición que no es `bool`:** el error habla de la condición y no de un operador `!` que no está escrito.
- Un dígito de otro alfabeto (`٣`) en un número da un error que lo explica.
- **La pista de `@` llega a todos los programas:** escribir `vida` en vez de `@vida` dentro de un método explica que falta la `@` también en los programas que usan `usar` (antes solo en los que no importaban nada), y con campos de tipos embebidos.
- **Autocompletado de `.Variante` en cualquier archivo:** las etiquetas de `casos`, los opcionales y los resultados (`.Alguno`, `.Ok`…) se completan también en los archivos con `usar`, que antes se quedaban sin esas sugerencias.
- **Errores de nombres repetidos más claros:** dicen si el nombre ya era una función, un tipo o una global.
- **Errores de ejecución con columna:** un índice fuera de la lista o una división entre cero muestran la línea y señalan la expresión con `^`.
- **Por qué un tipo no cumple una interfaz:** el error dice el primer motivo: el método que falta, el resultado o los parámetros que no coinciden, o el método o el tipo embebido sin `pub`.
- **`usar` que no encuentra su archivo:** el mensaje está en español y nombra el archivo como se escribió y el módulo que lo importa.
- **Función que no produce su resultado:** el error señala el tipo de resultado declarado y no la última sentencia.
- **`|variable|` en una rama de `casos`:** el error explica dónde va cuando se escribe en el lugar equivocado.
- **Estructuras omitidas:** un campo de estructura que no se escribe ya no exige inicializar los campos enum o de resultado de dentro cuando su tipo les da un valor predeterminado.
- **Miembros sobre un literal de estructura:** leer un campo o llamar a un método directamente sobre un literal ya compila.
- **`!` en la cabecera de una función:** en `fn f() bool !@g()` el `!` separado se lee como negación, no como tipo de resultado.
- **Mayúsculas acentuadas en `retro.texto`:** `Á`, `Í`, `Ó` y `Ú`, que la fuente no tiene, se dibujan con la letra sin tilde.
- `imprimir` ya no falla con una estructura sin campos.

### Herramientas

- **`cometa compilar` es más rápido** en programas de consola (de alrededor de 1 s a 0,2 s) y ya no necesita tener Go instalado: el Go generado deja de comprobarse en cada compilación; esa comprobación corre ahora en las pruebas del compilador.
- **El editor analiza menos:** los errores se recalculan cuando dejas de escribir un instante, no con cada tecla, y pasar el ratón o ir a una definición reutiliza el último análisis.
- El Go de los programas más simples sale ahora por el mismo camino que el de los demás, con una variable temporal por operando (más largo, mismo resultado).
- **Constantes de un tipo en el editor:** al pasar el ratón por `color.Color`, `entrada.Tecla` y los demás tipos con constantes se ven sus valores (`.Rojo`, `.Espacio`…). En `retro` aparecían bajo el tipo siguiente.
- **«Ir a la definición» en la biblioteca:** `mate.angulo`, `mate.interpolar` y los métodos `entrar`/`salir` de `graficos.Recorte` llevan a su propia línea y no a otra declaración con el mismo nombre.
- La referencia de `graficos.Camara2D` muestra que `zoom` vale 1 si no se indica.
- **`cometa captura` y `cometa probar` no tocan tus partidas guardadas:** `std/pincel/datos` usa una carpeta temporal vacía en las dos órdenes; `--datos carpeta` parte de un guardado concreto y conserva lo que se escriba.

## v0.1.0

Primera versión pública de Cometa: un lenguaje de programación con sintaxis en español, pensado para aprender y para hacer juegos 2D por diversión. Los programas se traducen a Go.

### Qué incluye

- **El lenguaje:** tipos estáticos, bloques con tabuladores, listas, mapas, opcionales (`T?`), resultados (`T!`), enums con `casos`, interfaces, genéricos y módulos con `usar`. La referencia completa está en [specs.md](https://github.com/crsolver/cometa/blob/main/specs.md).
- **Pincel**, la biblioteca de juegos 2D: gráficos, sprites y hojas, texto, cámara, teclado, ratón y mandos, audio, rejillas con colisiones, interfaz (`ui`), fuente retro, curvas de suavizado, ruido y guardado de datos.
- **Herramientas:** `cometa nuevo`, `ejecutar`, `construir`, `probar` (con pruebas visuales) y `captura`.
- **Extensión de VS Code:** colores, errores mientras escribes, autocompletado e ir a la definición.
- Errores en español, con sugerencias («¿quisiste decir…?») y la línea de tu programa cuando algo falla al ejecutar.

### Instalar

1. Descarga el archivo `cometa_…` de tu sistema (abajo), descomprímelo y pon `cometa` en tu `PATH`.
2. Instala [Go 1.25 o superior](https://go.dev/dl/).
3. Opcional: descarga el `.vsix` de tu sistema e instálalo con `code --install-extension cometa-win32-x64.vsix` (cambia el nombre por el de tu archivo).

Los ejecutables no están firmados; la [guía para empezar](https://github.com/crsolver/cometa/blob/main/docs/empezar.md) explica cómo abrirlos en Windows y macOS, y sigue con tu primer programa y tu primer juego.

### Limitaciones conocidas

Es una versión temprana: el lenguaje y la biblioteca pueden cambiar. Las asperezas conocidas están en [MEJORAS.md](https://github.com/crsolver/cometa/blob/main/MEJORAS.md).
