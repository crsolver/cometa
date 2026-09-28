# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### `imprimir` y la interpolación muestran los booleanos de forma distinta

- **Observado:** `imprimir(lista.contiene(9))` imprime `true`, pero `imprimir("${lista.contiene(9)}")` imprime `verdadero`.
- **Problema:** es incoherente y confunde a quien aprende, porque el lenguaje usa `verdadero`/`falso`.
- **Posible solución:** que `imprimir` formatee `bool` como `verdadero`/`falso`, también dentro de listas, mapas y opcionales.
- **Solución temporal en los ejemplos:** usar interpolación para mostrar booleanos.

### Los campos de un `tipo` no aceptan valores predeterminados

- **Observado:** `energia entero = 5` dentro de un `tipo` produce `se esperaba el final de la declaración del campo`.
- **Problema:** es una expectativa natural (las funciones sí admiten `param T = valor`) y obliga a escribir el valor en cada literal o a crear una función fábrica.
- **Posible solución:** admitir `campo Tipo = expresión`, evaluada al omitir el campo en un literal, con las mismas reglas que los valores predeterminados de parámetros. Como mínimo, un mensaje de error que diga que no se admiten.

### `repetir` con rango exige una variable aunque no se use

- **Observado:** `repetir (0..3)` sin `|i|` produce `se esperaba '|' antes de las variables del ciclo`.
- **Problema:** para "repetir N veces" hay que declarar una variable que nunca se lee.
- **Posible solución:** permitir `repetir (0..n)` sin variables, o aceptar `|_|`.

### `imprimir` de colecciones usa el formato de Go

- **Observado:** `imprimir(["Ana", "Luis"])` muestra `[Ana Luis]`, sin comillas ni comas; los mapas y estructuras también usan el formato de Go.
- **Problema:** menor, pero se nota en los ejemplos para principiantes y no se parece a la sintaxis de los literales de Cometa.
- **Posible solución:** un formateador propio: `["Ana", "Luis"]`, `["Ana": 12]`, `Mascota {nombre: "Toby"}`.

## Biblioteca estándar

### Un contenedor `contenido()` con un hijo `expandir()` colapsa a tamaño 0 sin aviso

- **Observado:** `con ui.fila("acciones", espacio = 2)` (sin `ancho`, así que hereda `contenido()`) con un único `ui.boton(..., ancho = ui.expandir())` dentro produce una fila y un botón invisibles, de ancho 0 — sin ningún error ni advertencia.
- **Problema:** el tamaño natural de un hijo `expandir()` es su mínimo (0 salvo que se indique), así que un padre `contenido()` calcula "ajústate a tu contenido" como 0. Es coherente con cómo funciona `contenido()`, pero es fácil de escribir por accidente (por ejemplo al componer un grupo de opciones con botones `expandir()`) y el resultado es una interfaz silenciosamente vacía en vez de un error.
- **Posible solución:** que `_hgUIRequire` detecte esta combinación igual que ya hace con `porcentaje` bajo `contenido` en el mismo eje, o que `contenido()` sin límites explícitos tome el mínimo de `expandir()` como advertencia en vez de 0 silencioso.

### `ui.campo_texto` no admite selección de texto ni portapapeles

- **Observado:** el campo de texto de `std/pincel/ui` inserta y borra caracteres y mueve el cursor con las flechas/Inicio/Fin, pero no se puede seleccionar texto (arrastrar o Mayús+flecha), hacer clic para posicionar el cursor en medio del texto, ni pegar desde el portapapeles.
- **Problema:** cubre el caso de un campo corto (nombre, valor numérico), pero cualquier edición más larga es incómoda: no hay forma de reemplazar todo el contenido de una vez ni de corregir un error a mitad de la palabra sin usar solo las flechas.
- **Posible solución:** añadir un segundo índice de selección al nodo, resaltar el rango seleccionado al dibujar, calcular la posición de clic contra las líneas medidas, y usar las funciones de portapapeles de Ebitengine para copiar/pegar.

### `azar.entero(min, max)` excluye `max`

- **Observado:** `azar.entero(1, 6)` nunca devuelve 6; para un dado hay que escribir `azar.entero(1, 7)`.
- **Problema:** es coherente con los rangos `a..b`, pero sorprende (muchas bibliotecas incluyen ambos extremos) y la firma no lo indica.
- **Posible solución:** documentarlo en la firma y en `docs/juegos.md`, o renombrar los parámetros (`desde`, `hasta_excluido`), o añadir una variante inclusiva.
