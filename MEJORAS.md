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

### `azar.entero(min, max)` excluye `max`

- **Observado:** `azar.entero(1, 6)` nunca devuelve 6; para un dado hay que escribir `azar.entero(1, 7)`.
- **Problema:** es coherente con los rangos `a..b`, pero sorprende (muchas bibliotecas incluyen ambos extremos) y la firma no lo indica.
- **Posible solución:** documentarlo en la firma y en `docs/juegos.md`, o renombrar los parámetros (`desde`, `hasta_excluido`), o añadir una variante inclusiva.
