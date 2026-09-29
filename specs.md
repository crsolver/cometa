# Cometa: especificación del MVP

Cometa es un lenguaje estáticamente tipado, con sintaxis en español e indentación significativa, de propósito general con bibliotecas opcionales para juegos 2D. El compilador está escrito en Go y genera un único archivo Go perteneciente a `package main`.

## Diagnósticos y recuperación

El frontend acumula errores léxicos, sintácticos y semánticos recuperables en una ejecución. Tanto la CLI como el editor los muestran ordenados por archivo y posición, sin duplicados, hasta 100 errores más un aviso de truncamiento. Se mantiene el formato `archivo:línea:columna: mensaje` de la CLI.

La recuperación usa límites de línea, declaración e indentación; no interpreta espacios como tabuladores. Se conservan fragmentos válidos para comprobar errores independientes y ofrecer navegación en el editor. Los nombres de declaraciones fallidas se conservan como inválidos para evitar errores derivados en sus usos. Cuando faltan datos necesarios, se posponen las comprobaciones dependientes, incluidas algunas validaciones globales y de resultados no usados. Corregir la causa puede revelar errores adicionales en esos fragmentos.

Un AST o modelo parcial nunca permite generar Go, construir ni ejecutar. Los errores del frontend no reemplazan archivos de salida existentes.

## Biblioteca estándar y Pincel

`usar std/mate/ruido` expone `suave(x decimal, y decimal, semilla entero = 0) decimal` y `fractal(x decimal, y decimal, semilla entero = 0, octavas entero = 4, persistencia decimal = 0.5, lacunaridad decimal = 2.0) decimal`. Son funciones puras de ruido de valores 2D, reproducibles por semilla, con resultado en [0, 1], sin Ebitengine ni estado aleatorio. El ruido suave interpola esquinas mediante suavizado quíntico; el fractal normaliza la suma ponderada de octavas con semillas derivadas (la primera usa la original). Se requieren coordenadas finitas dentro de (-2^52, 2^52) en cada octava, 1..16 octavas, persistencia en [0, 1] y lacunaridad finita >= 1; entradas inválidas provocan panic. Véase [la API de ruido](docs/ruido.md).

`usar std/mate/curvas` importa el namespace `curvas`, independiente de `std/mate` y de Ebitengine. Expone `lineal` y las familias `cuadratica`, `cubica`, `cuartica`, `quintica`, `senoidal`, `circular`, `exponencial`, `elastica`, `retroceso` y `rebote`, cada una con sufijos `_entrada`, `_salida` y `_entrada_salida`: 31 funciones puras `(progreso decimal) decimal`. Los enteros se amplían a decimal como en otras llamadas. Para progreso menor o igual a 0 devuelven exactamente 0; para progreso mayor o igual a 1 devuelven exactamente 1. Los infinitos se limitan a esos extremos y NaN se propaga. Solo se limita la entrada: las curvas elásticas y de retroceso conservan resultados fuera de [0, 1]. No administran tiempo ni estado. Véase [la API de curvas](docs/curvas.md).

`usar std/mate` y `usar std/azar` importan matemáticas y azar sin Ebitengine. `usar std/pincel` importa el núcleo de Pincel (`pincel.ejecutar` y la interfaz `pincel.Juego`). El resto de Pincel son módulos independientes bajo `std/pincel/`: `graficos`, `color`, `entrada`, `audio`, `ventana`, `tiempo`, `recursos`, `retro`, `lienzo` y `ui`. El último segmento es el alias predeterminado; se admite `como`. `usar std/pincel` no importa los submódulos; no existe importación comodín ni reexportación implícita. Cada archivo importa lo que utiliza. `./std/...` sigue siendo una ruta local.

Los tipos se califican: `mate.Vec2`, `mate.Rect`, `graficos.Camara2D`, `color.Color`, `entrada.Tecla`, `entrada.BotonRaton`, `graficos.Imagen`, `graficos.Fuente`, `audio.Sonido` y `audio.Reproduccion`. Los seis primeros conservan semántica de valor; los recursos son handles opacos compartidos. Sus nombres no están reservados globalmente. Se conservan operadores vectoriales, métodos, literales nombrados/posicionales y constantes contextuales.

`fn inicio()` es el único punto de entrada. Se llama explícitamente a `pincel.ejecutar(instancia, ...)`, que recibe un objeto compatible con la interfaz estructural `pincel.Juego`: métodos `pub fn actualizar(dt decimal)` y `pub fn pintar()`, ambos sin resultado. La biblioteca invoca esos métodos; pueden pertenecer a un módulo importado. Los nombres de funciones globales no activan ningún perfil. No se invoca `iniciar` automáticamente y se elimina `pincel.configuracion`.

La llamada acepta dimensiones, título, escala, redimensionamiento, pantalla completa y TPS; mantiene los defaults 320×180, "Cometa", escala 1 y 60 TPS. Valida la configuración, inicia Ebitengine y bloquea hasta cerrar la ventana. Devuelve `!`: `.Ok` al cerrar, `.Error(cadena)` ante configuración inválida, otra ejecución en el mismo proceso o un error del backend. El usuario prepara su estado antes de la llamada. `dt` son segundos por tick.

Dibujar requiere estar dentro de la fase activa de `pintar`; la biblioteca lo comprueba en ejecución, también en helpers. Ya no hay análisis de efectos del ciclo de juego en el compilador. Se conservan recursos con rutas literales en inicializadores globales directos, relativas al archivo declarante, validados e incorporados en compilación.

El compilador incorpora solamente el cierre de dependencias nativas de los módulos importados. `compilar` genera un único archivo Go; `construir` y `ejecutar` usan un módulo Go temporal con las dependencias requeridas. Pincel usa Ebitengine v2.10.1. Los programas sin dependencias Go externas conservan validación de tipos Go durante `compilar`; el resto se verifica completamente al construir.

Las firmas, coordenadas y recursos se detallan en [Pincel](docs/juegos.md).

`std/pincel/retro` añade dibujo incorporado de texto y glifos bitmap de 8×8: `texto`, `icono` y `glifo`, con coordenadas y escala enteras, color contextual y tipos opacos de valor `Icono` y `Atlas`. `texto` mapea Unicode al atlas Dungeon-437; `icono` usa nombres contextuales del atlas Dungeon-mode; `glifo` permite índices 0–255 y selección `.Dungeon`/`.ASCII`. Los atlas se incorporan al Go generado únicamente al importar este módulo. No son recursos declarados por el usuario. Los dibujos obedecen la cámara y la restricción de fase `pintar` de Pincel.

`std/pincel/lienzo` crea `graficos.Imagen` editables en memoria de CPU: `nuevo`, `desde_texto`/`hoja_desde_texto` (una fila de texto por fila de píxeles con un mapa `[cadena: Color]` de claves de un carácter; `.` y el espacio son transparentes salvo redefinición), `ancho`, `alto`, `copiar`, `limpiar`, `pixel`, `leer_pixel` (`Color?`), `rect`, `linea`, `circulo`, `rellenar`, `pegar` y `guardar(imagen, ruta, escala = 1) !`. Las coordenadas son enteras y sin suavizado; las primitivas reemplazan píxeles y recortan fuera de los bordes; `pegar` mezcla con alfa y admite espejo horizontal y vertical. Las funciones no dependen de la fase `pintar` y pueden usarse en globales o en `inicio`; la imagen se sube a la GPU al dibujarse y otra vez tras editarse. Los datos inválidos detienen el programa con la fila o cuadro afectado. Consulta [Lienzo](docs/lienzo.md).

Los parámetros opcionales finales de `pincel.ejecutar` son `pixelado bool = falso` y `retro bool = falso`, en ese orden. `pixelado` controla la presentación por vecino más cercano; no cambia la resolución lógica ni impone escalas enteras. `retro = verdadero` activa vecino más cercano aunque `pixelado = falso` y aplica un shader CRT a toda la imagen, incluido el texto: líneas de barrido suaves, una máscara RGB tenue y viñeta ligera, sin curvatura ni parpadeo. Los patrones finos se atenúan a escalas pequeñas. Conserva la relación de aspecto, las bandas de presentación y las coordenadas del ratón y la cámara. No requiere importar `std/pincel/retro`. Los errores de inicialización del shader se devuelven mediante el resultado de `ejecutar`. El texto TTF existente conserva su API.

## Compilar

```console
cometa compilar programa.cometa
cometa compilar programa.cometa -o salida.go
cometa captura juego.cometa -o captura.png --escala 4 --cuadros 30
```

Sin `-o`, la salida usa el mismo nombre base con la extensión `.go`. El compilador formatea y verifica los tipos del código Go antes de escribirlo. Los errores incluyen archivo, línea y columna.

`captura` construye y ejecuta un juego de Pincel con la ventana oculta, guarda la pantalla lógica como PNG después de `--cuadros` actualizaciones (1 por defecto), ampliada por vecino más cercano con `--escala` (1–64), y termina. Sin `-o` escribe `<archivo>.png`. Los programas sin `pincel.ejecutar` se rechazan.

El mismo ejecutable inicia el servidor LSP mediante `cometa lsp`. El servidor se comunica por entrada/salida estándar, publica los errores del compilador al abrir, cambiar o guardar un documento, ofrece el esquema jerárquico del archivo, completa campos y métodos después de `.` o de `@` dentro de métodos, y muestra los tipos inferidos de variables y las firmas de funciones al pasar el cursor. Las líneas `//` consecutivas inmediatamente anteriores a una función se muestran como su documentación.

## Indentación y comentarios

- La indentación usa exclusivamente tabuladores. Los espacios iniciales en una línea con código son un error.
- Una línea vacía o que solo contiene un comentario `//` no afecta los bloques.
- `tipo`, `enum`, `interfaz`, `fn`, `si`, `osi`, `sino`, `casos` y `repetir` abren bloques. Las variantes de un enum y las ramas de `casos` siempre se indentan.
- Un cuerpo en la misma línea contiene una sola sentencia o expresión y termina con esa línea.
- No existen `fin`, dos puntos de declaración ni bloques de control con llaves. Las llaves delimitan literales de estructuras. `=>` se usa exclusivamente entre el patrón y el cuerpo de una rama de `casos`.

## Parámetros variádicos y argumentos nombrados

```cometa
fn sumar(base entero, valores ...entero) entero
	var total = base
	repetir (valores) |valor|
		total = total + valor
	total

fn inicio()
	var lista = [1, 2]
	imprimir(sumar(10))
	imprimir(sumar(10, 1, 2))
	imprimir(sumar(10, lista...))
	imprimir(sumar(valores = lista, base = 10))
```

- Solo el último parámetro de una función o método puede ser variádico: `nombre ...T`. Dentro del cuerpo tiene tipo `[T]`.
- Una llamada posicional proporciona cero o más elementos `T`, o una única lista `[T]` seguida de `...` como último argumento. No se pueden combinar elementos variádicos individuales con una expansión. La expansión comparte el almacenamiento de la lista; sin elementos se genera una lista Go nil. Los elementos individuales forman una lista nueva.
- Los argumentos nombrados usan `nombre = expresión` y pueden aparecer en cualquier orden después de los posicionales. No se permiten posicionales después de nombrados, nombres desconocidos, parámetros duplicados ni parámetros obligatorios ausentes.
- Un argumento variádico nombrado recibe la lista completa: `valores = lista`; también admite `valores = lista...` si es el último argumento. Se puede omitir el parámetro variádico.
- El receptor se evalúa primero y los argumentos se evalúan una vez, en el orden escrito, aunque sus nombres requieran reordenarlos en Go.
- Los tipos esperados se propagan a cada argumento, incluyendo literales vacíos, estructuras y variantes de enum contextuales. Los tipos declarados conservan su semántica de referencia.
- Se puede asignar a elementos de listas (`valores[0] = 42`), incluyendo listas recibidas mediante expansión; la escritura es visible para quien comparte esa lista.
- `imprimir(valor = expresión)` equivale a `imprimir(expresión)`. Los constructores de payload de enum no tienen nombres de parámetros y solo aceptan su argumento posicional, sin expansión.

## Valores predeterminados de parámetros

```cometa
fn saludar(nombre cadena = "mundo", saludo cadena = "hola " + nombre) cadena
	saludo

fn inicio()
	imprimir(saludar())
	imprimir(saludar("Ana"))
	imprimir(saludar(saludo = "buenos días"))
```

- Las funciones y los métodos aceptan `nombre Tipo = expresión`. Los parámetros obligatorios deben preceder a los que tienen valores predeterminados. Puede seguir un parámetro variádico final, que no admite un valor predeterminado explícito.
- Los argumentos posicionales ocupan los parámetros de izquierda a derecha; los nombrados permiten omitir parámetros con valores predeterminados. Un valor explícito, incluso `0`, `falso`, `""` o `[]`, sustituye al predeterminado.
- Cada expresión predeterminada se evalúa una sola vez por llamada, únicamente si se omite su argumento. Primero se evalúan el receptor y todos los argumentos explícitos en el orden escrito; después se evalúan los valores omitidos en el orden de declaración, antes del cuerpo.
- La expresión recibe el tipo esperado del parámetro y se valida aunque la función nunca se llame. Admite literales contextuales, llamadas a funciones y referencias a parámetros anteriores; en métodos también admite miembros con `@`. No puede referirse al propio parámetro, a parámetros posteriores ni a variables del cuerpo o del llamador.
- Los literales de listas y estructuras crean valores nuevos en cada evaluación. Las referencias obtenidas de parámetros anteriores conservan su semántica habitual de compartir objetos y almacenamiento.
- `inicio` sigue sin aceptar parámetros. Los constructores de enum e `imprimir` conservan sus reglas de argumentos.

## Valores predeterminados de campos

```cometa
tipo Mascota
	nombre cadena
	energia entero = 10

fn inicio()
	var gato = Mascota {nombre: "Michi"}
	imprimir(gato.energia)
	var perro = Mascota {nombre: "Rex", energia: 3}
	imprimir(perro.energia)
```

- Los campos de un `tipo` aceptan `nombre Tipo = expresión`. Un valor explícito en el literal, incluso `0`, `falso`, `""` o `[]`, sustituye al predeterminado; omitir el campo evalúa la expresión.
- Cada expresión predeterminada se evalúa una sola vez por literal que omite ese campo, en el orden de declaración de los campos. Admite literales contextuales, llamadas a funciones y referencias a globales, pero no puede referirse a otros campos del mismo tipo (ni siquiera a sí mismo) ni usar `@`, porque no hay un receptor durante la construcción del literal.
- Los literales de listas y estructuras usados como predeterminados crean valores nuevos en cada literal que los omite; no se comparten entre instancias distintas.
- Un campo con valor predeterminado queda exento de la regla de inicialización obligatoria, incluyendo la de los ciclos de campos requeridos: un tipo autorreferencial que sería rechazado como ciclo sin valor predeterminado se acepta si el campo que cierra el ciclo lo tiene, aunque construir una instancia sin una anulación explícita que termine la recursión sigue siendo responsabilidad de quien llama.
- Los campos embebidos no admiten valores predeterminados. Los campos declarados directamente como un parámetro de tipo `T` siguen requiriendo inicialización explícita en cada instanciación.

## Tipos

| Cometa | Go |
| --- | --- |
| `entero` | `int64` |
| `decimal` | `float64` |
| `cadena` | `string` |
| `bool` | `bool` |
| `Usuario` | `*Usuario` |
| `[Usuario]` | `[]*Usuario` |

Las estructuras y enums declarados por el programa tienen semántica de referencia. Las interfaces conservan el valor dinámico y los parámetros de tipo conservan la representación de su argumento. Los tipos, campos, métodos y funciones generados se exportan en Go.

### Números

`entero` tiene signo y 64 bits; `decimal` usa IEEE-754 binario de 64 bits, no aritmética decimal exacta. `num` fue eliminado: debe elegirse uno de los dos tipos. Los literales sin punto son enteros, los literales con punto son decimales; se rechazan literales fuera del rango de su tipo, y se admite `-9223372036854775808`.

`entero` se promueve implícitamente a `decimal` en asignaciones, argumentos, retornos, campos, ramas y elementos de literales. Dos operandos enteros conservan el tipo con `+`, `-`, `*` y `/`; `5 / 2` vale `2` y `-5 / 2` vale `-2`. `%` solo admite enteros y su resto tiene el signo del dividendo. Si algún operando es decimal, los operadores aritméticos y las comparaciones promueven el entero. El tipo esperado del resultado no cambia la división: `var x decimal = 5 / 2` vale `2.0`; use `5 / 2.0` para obtener `2.5`.

Listas y ramas numéricas sin tipo explícito infieren `decimal` si contienen algún resultado decimal. Los contenedores ya construidos conservan su tipo: `[entero]` no se convierte a `[decimal]`, ni `entero?` a `decimal?`. Un valor entero individual sí puede promoverse antes de envolverse una vez en `decimal?` o `decimal!`. Las interfaces conservan la identidad dinámica de `entero` y `decimal`; las firmas de métodos y los argumentos genéricos siguen siendo exactos.

`decimal(valor)` hace explícita la promoción opcional. `entero(valor)` trunca hacia cero y produce un error de ejecución en español ante NaN, infinito o valores fuera de `int64`. `mate.piso`, `mate.techo` y `mate.redondear` devuelven `entero`, con la misma validación después de redondear; los empates de `redondear` se alejan de cero. `mate.absoluto`, `minimo`, `maximo` y `limitar` conservan enteros si todos sus argumentos son enteros, y producen decimal si alguno es decimal.

La promoción a `decimal` usa redondeo IEEE-754: por encima de `2^53` algunos enteros pierden precisión. La aritmética entera en ejecución sigue las reglas de `int64` de Go, incluido desbordamiento; la división entera por cero falla. Las expresiones constantes no representables son errores de compilación. La aritmética decimal admite NaN e infinitos en ejecución.

### Cadenas

`+` concatena dos valores `cadena`; no convierte otros tipos automáticamente. Una cadena puede incluir expresiones con `${expresión}`. La interpolación acepta `cadena`, `entero`, `decimal` y `bool`, evalúa cada expresión una sola vez de izquierda a derecha y representa booleanos como `verdadero` o `falso`. `\${` escribe los caracteres `${` literalmente. Las cadenas interpoladas son expresiones de ejecución y no se admiten en `const`.

```cometa
var nombre = "Ana"
imprimir("Hola " + nombre)
imprimir("${nombre}, tienes ${20 + 1} años")
```

Los índices y longitudes cuentan puntos de código Unicode. Los índices deben tener tipo `entero`. `subcadena` usa un límite final exclusivo; los límites inválidos devuelven `.Ninguno` y un rango vacío válido devuelve `.Alguno("")`.

| Método | Resultado |
| --- | --- |
| `longitud()` | `entero` |
| `esta_vacia()` | `bool` |
| `contiene(valor cadena)` | `bool` |
| `buscar_indice(valor cadena)` | `entero?` |
| `empieza_con(prefijo cadena)` | `bool` |
| `termina_con(sufijo cadena)` | `bool` |
| `mayusculas()` | `cadena` |
| `minusculas()` | `cadena` |
| `recortar()` | `cadena` |
| `reemplazar(buscar cadena, reemplazo cadena)` | `cadena` |
| `dividir(separador cadena)` | `[cadena]` |
| `obtener(indice entero)` | `cadena?` |
| `subcadena(inicio entero, fin entero)` | `cadena?` |

`dividir("")` divide por punto de código y devuelve una lista vacía para la cadena vacía. `reemplazar("", texto)` inserta el reemplazo en los límites entre puntos de código, incluidos ambos extremos. Los métodos no modifican el receptor.

```cometa
tipo Usuario
	nombre cadena
	edad entero
	activo bool
	referido Usuario?
	amigos [Usuario]
```

## Interfaces y genéricos

`interfaz Nombre` declara un conjunto de firmas de métodos. Su cuerpo indentado contiene firmas `fn` sin cuerpo ni valores predeterminados, o nombres de otras interfaces que incorpora. Una declaración sin cuerpo es una interfaz vacía y acepta cualquier tipo que produzca un valor.

```cometa
interfaz Describible
	fn describir() cadena

interfaz Proveedor<T>
	fn obtener() T

tipo Caja<T>
	valor T
	pub fn obtener() T @valor

fn identidad<T>(valor T) T valor
fn describir<T Describible>(valor T) cadena valor.describir()
```

### Compatibilidad de interfaces

- La implementación es implícita: coinciden los nombres de métodos, los tipos y orden de parámetros, la condición variádica y el tipo de resultado. Los nombres de parámetros y los valores predeterminados de la implementación no forman parte de la compatibilidad. No hay covarianza de resultados.
- Una interfaz compuesta combina las firmas incorporadas. Firmas idénticas se combinan; firmas incompatibles y ciclos de incorporación se rechazan. Si las firmas incorporadas usan nombres de parámetros diferentes, la interfaz debe redeclarar explícitamente el método con los nombres elegidos.
- Las llamadas por interfaz usan los nombres de parámetros de esa interfaz y suministran todos los argumentos no variádicos. Los valores predeterminados de un método concreto solo se aplican en llamadas a través de su tipo concreto.
- Un valor concreto compatible se asigna implícitamente a una interfaz. Una interfaz se asigna a otra si garantiza sus métodos. Solo se exponen los métodos declarados por el tipo estático.
- Los campos de interfaz requieren inicialización explícita, también dentro de estructuras anidadas. No existe una interfaz nula; `I?` representa ausencia. Guardar una estructura en una interfaz conserva la referencia original.
- Las listas, estructuras/enums genéricos y wrappers ya construidos conservan sus argumentos exactos: `[Usuario]` no se convierte en `[Describible]`. Un literal con tipo esperado `[Describible]` sí acepta elementos individuales compatibles. Una conversión incorporada a `I?` o `I!E` puede envolver una vez un valor compatible con `I`.
- No se permite igualdad entre interfaces. Guardar o pasar explícitamente un resultado sigue contando como uso; no se agregan reglas de consumo dinámico para resultados guardados dentro de interfaces.

### Inspección segura de tipos

`valor como Tipo` requiere un operando de interfaz y devuelve `Tipo?`. Un destino concreto comprueba el tipo dinámico exacto; un destino de interfaz comprueba sus métodos. Un fallo produce `.Ninguno`, nunca un pánico. Se admiten escalares, listas, wrappers, tipos genéricos instanciados y parámetros de tipo en alcance. Se rechazan comprobaciones concretas que nunca pueden implementar la interfaz de origen.

`como` tiene menor precedencia que la aritmética y mayor que las comparaciones. Para extraer el resultado antes de acceder a sus miembros, agrúpelo: `(valor como Usuario) o alternativa`.

```cometa
interfaz Cualquiera

fn mostrar(valor Cualquiera)
	casos valor |dato|
		Usuario => imprimir(dato.nombre)
		Describible => imprimir(dato.describir())
		[entero] => imprimir(dato[0])
		_ => imprimir("otro")
```

En `casos` sobre una interfaz, las etiquetas son tipos completos. La ligadura tiene el tipo de la etiqueta y no existe en `_`. Se evalúa el operando una sola vez y se ejecuta la primera rama compatible. Siempre se requiere `_` al final, porque la interfaz admite futuras implementaciones. Se rechazan etiquetas de tipo duplicadas; otras superposiciones son válidas. Las reglas de resultados, ramas y control de ciclos coinciden con las de `casos` existente. Las variantes de enum siguen requiriendo `.Variante` o `Enum.Variante`.

### Parámetros de tipo

- Las funciones, estructuras, enums e interfaces aceptan parámetros `<T, U>`. Cada parámetro puede indicar una interfaz: `<T Describible>`. La restricción puede referirse a otros parámetros de la misma declaración; incorporar interfaces permite combinar requisitos. Los nombres de parámetros deben ser únicos y no ocultar tipos declarados.
- Los cuerpos genéricos se verifican aunque no se utilicen. Un parámetro sin restricción puede guardarse, pasarse y devolverse; sus métodos requieren una restricción que los declare. No se permite aritmética, orden ni igualdad sobre parámetros de tipo. No existen restricciones de campos, uniones de tipos, tipos subyacentes, especializaciones ni valores predeterminados de parámetros de tipo.
- Los métodos de estructuras genéricas heredan sus parámetros y no pueden declarar otros. Los enums continúan sin métodos.
- Las llamadas de función infieren parámetros emparejando estructuralmente los tipos de los argumentos explícitos con los tipos formales. Se incluyen argumentos nombrados, elementos variádicos y expansiones. Las apariciones repetidas de un parámetro deben coincidir exactamente. No se infiere desde el resultado esperado, argumentos omitidos ni métodos de las restricciones. Si falta información, se deben escribir todos los argumentos: `f<entero, Usuario>(...)`.
- Los usos de tipos requieren argumentos explícitos: `Caja<Usuario>`, `Evento<entero>.Dato(1)`. Los literales contextuales y constructores `.Variante` pueden recibir una instanciación completa como tipo esperado. No hay inferencia de parámetros desde los campos de `Caja { ... }`.
- Un campo declarado directamente como `T` requiere un valor incluso en `Caja<entero>`. `[T]` y `T?` conservan valores predeterminados válidos. Las estructuras anidadas conservan sus requisitos de inicialización. Se rechazan ciclos de campos obligatorios e instanciaciones recursivas que expandan indefinidamente sus argumentos; las listas y opcionales permiten recursión regular.
- Las estructuras y enums instanciados conservan referencias: `Caja<Usuario>` genera `*Caja[*Usuario]` en Go. Los parámetros de tipo y las interfaces se generan sin añadir un puntero adicional.
- En expresiones, una secuencia completa `<tipos>` después de un nombre seguida por `(`, `{` o `.` se interpreta como argumentos de tipo antes que como comparaciones. Se admiten cierres anidados `>>`. El resto de comparaciones conserva su sintaxis.

Consulte `examples/basico/11_interfaces.cometa`.

## Globales y constantes

`var` y `const` pueden declararse sin indentación al nivel superior. Ambas formas requieren un inicializador y aceptan un tipo explícito opcional. Las globales forman parte del mismo espacio de nombres que los tipos y funciones. Son privadas por defecto; `pub var` y `pub const` permiten acceder desde otro módulo mediante su alias.

```cometa
const limite entero = 10
var contador = 0

fn incrementar()
	contador = contador + 1
```

Una `var` global puede leerse y reasignarse desde funciones. Una `const` nunca puede ser objetivo de una asignación y solo admite `entero`, `decimal`, `cadena` o `bool`, formados por literales, otras constantes y operadores unarios o binarios compatibles. No admite llamadas, listas, estructuras, enums ni wrappers.

Los inicializadores de `var` admiten literales compuestos, constructores, referencias a otras globales y llamadas ordinarias. No admiten `retornar`, `intentar`, `capturar`, bloques ni expresiones `si` o `casos`. Las referencias adelantadas son válidas; un ciclo entre inicializadores globales es un error. Los módulos importados se inicializan antes que el módulo que los importa.

## Variables y literales compuestos

Dentro de una función, `var` declara una variable local. El tipo puede inferirse desde el valor o escribirse entre el nombre y `=`. Un literal de estructura puede indicar su tipo (`Usuario { ... }`) o recibirlo de la declaración (`var usuario Usuario = { ... }`). Los campos omitidos reciben un valor predeterminado válido: escalares en cero, listas vacías, opcionales ausentes y estructuras nuevas construidas recursivamente. Los campos de resultado o enum requieren inicialización explícita, incluso dentro de estructuras anidadas. El diagnóstico indica la ruta del campo que falta. Se rechazan ciclos directos e indirectos de campos de estructura obligatorios; los opcionales y las listas permiten recursión. Cada construcción crea sus propios objetos predeterminados, sin compartirlos con otras instancias.

Un literal puede usar valores posicionales, que corresponden a los campos directos en el orden de su declaración. Los campos embebidos ocupan una posición; los campos promovidos no. Se permite proporcionar solo un prefijo y los campos restantes usan las mismas reglas de valores predeterminados o de inicialización obligatoria. Un literal no puede mezclar valores posicionales y campos nombrados. Cambiar el orden de los campos cambia el significado de los literales posicionales existentes.

```cometa
var usuario1 = Usuario {
	nombre: "andres",
	edad: 29
}

var usuario2 Usuario = {
	nombre: "andres",
	amigos: [usuario1]
}

var usuario3 = Usuario {"Ana", 29}
```

Las listas no vacías infieren su tipo desde el primer elemento y exigen que los demás sean compatibles. Una lista vacía requiere un tipo esperado.

Los literales de lista admiten varias líneas después de `[`: los elementos se indentan con tabuladores y `]` vuelve al nivel de la línea inicial. Los elementos se separan con comas; la coma final es opcional, también en una sola línea. Se permiten líneas vacías, comentarios y listas anidadas.

```cometa
var lista = []                 // error: no se puede inferir el elemento
var lista2 [Usuario] = []      // válido
var usuarios = [usuario1, usuario2]
```

Una lista se indexa con una expresión `entero` entre corchetes. El acceso produce un valor del tipo de sus elementos y puede usarse dentro de otra expresión, incluso como argumento de una función.

```cometa
var primero = usuarios[0]
procesar_usuario(usuarios[1])
```

Las listas ofrecen métodos incorporados. Los métodos `agregar`, `extender`, `insertar`, `eliminar` e `invertir` actualizan directamente la variable, parámetro, campo o elemento de lista que recibe la llamada. Los demás métodos aceptan también expresiones temporales.

| Método sobre `[T]` | Resultado | Comportamiento |
|---|---|---|
| `longitud()` | `entero` | Cantidad de elementos |
| `esta_vacia()` | `bool` | Indica si la lista está vacía |
| `contiene(valor T)` | `bool` | Busca un valor igual |
| `buscar_indice(valor T)` | `entero?` | Primer índice, o `.Ninguno` |
| `obtener(indice entero)` | `T?` | Elemento, o `.Ninguno` si el índice no es válido |
| `primero()` / `ultimo()` | `T?` | Elemento extremo, o `.Ninguno` |
| `agregar(valor T)` | sin valor | Agrega al final |
| `extender(otra [T])` | sin valor | Agrega todos los elementos de otra lista |
| `insertar(indice entero, valor T)` | `bool` | Inserta antes del índice; permite el final |
| `eliminar(indice entero)` | `bool` | Elimina el elemento conservando el orden |
| `copiar()` | `[T]` | Copia superficial con almacenamiento independiente |
| `invertir()` | sin valor | Invierte los elementos en el mismo almacenamiento |

Los índices de estos métodos deben tener tipo `entero` y estar dentro del rango. Los decimales se rechazan durante el chequeo de tipos. `insertar` y `eliminar` devuelven `falso` sin modificar la lista ante un índice inválido. `obtener` devuelve `.Ninguno`. `contiene` y `buscar_indice` admiten números, cadenas, booleanos y estructuras, que se comparan por identidad; todavía no admiten enums, interfaces, wrappers, listas anidadas ni parámetros de tipo.

```cometa
var valores = [10, 20]
valores.agregar(30)
imprimir(valores.buscar_indice(20) o -1)
imprimir(valores.obtener(99) o 0)
```

La asignación de una lista copia su descriptor de slice: cada alias conserva su propia longitud, aunque puede compartir el almacenamiento de los elementos. Por eso agregar mediante un alias no cambia la longitud de los demás; escribir, insertar, eliminar o invertir puede hacer visibles cambios de elementos en aliases que todavía compartan almacenamiento. Cambiar la longitud de un parámetro de lista tampoco cambia la variable del llamador. `copiar()` crea almacenamiento independiente, pero conserva las referencias contenidas.

Los campos y métodos de una variable se acceden con `.`. El marcador `@` sigue reservado para el receptor del método actual.

```cometa
usuario2.activar(verdadero)
imprimir(usuario2.nombre)
```

## Mapas

Un mapa mutable tiene tipo `[K: V]`, con claves `cadena`, `entero` o `bool` y cualquier tipo almacenable como valor. Los parámetros genéricos solo se admiten como valores (`[cadena: T]`); no existen restricciones genéricas de claves comparables. Los mapas no admiten igualdad ni ordenación.

```cometa
var puntos [cadena: entero] = [:]
var iniciales = ["Ana": 10, "Luis": 20]
puntos["Ana"] = 10
imprimir(puntos["Ana"] o 0)
repetir (puntos) |valor, clave|
	imprimir("${clave}: ${valor}")
```

`[:]` requiere un tipo esperado; `[]` sigue siendo una lista. Sin tipo esperado, la primera entrada determina ambos tipos y las siguientes deben ser compatibles. Con tipo esperado, se propaga a claves y valores, incluidos literales contextuales, genéricos y conversiones habituales. Los mapas ya construidos son invariantes en clave y valor. Se admiten literales multilínea con las mismas reglas de tabuladores, comentarios y comas finales que las listas.

Las entradas se evalúan en orden escrito, primero la clave y luego el valor. Las claves repetidas conservan el último valor, sin omitir los efectos de entradas anteriores. `mapa[clave]` y `obtener(clave)` devuelven `V?`; una entrada existente cuyo valor sea `.Ninguno` se distingue de una clave ausente mediante el opcional exterior. La asignación `mapa[clave] = valor` recibe `V`, evalúa receptor, clave y valor una vez en ese orden, y solo entonces inserta o reemplaza. No se puede acceder directamente al contenido de una lectura sin extraer el opcional.

| Método sobre `[K: V]` | Resultado | Comportamiento |
|---|---|---|
| `longitud()` | `entero` | Cantidad de entradas |
| `esta_vacia()` | `bool` | Indica si no hay entradas |
| `contiene(clave K)` | `bool` | Comprueba la presencia de una clave |
| `obtener(clave K)` | `V?` | Valor presente o `.Ninguno` |
| `eliminar(clave K)` | `bool` | Elimina la entrada; devuelve si existía |
| `claves()` | `[K]` | Lista independiente de claves sin orden garantizado |
| `valores()` | `[V]` | Lista independiente de valores sin orden garantizado |
| `copiar()` | `[K: V]` | Copia superficial con almacenamiento independiente |
| `vaciar()` | sin valor | Elimina todas las entradas |

La asignación y el paso de parámetros comparten las entradas: insertar, reemplazar, eliminar y vaciar es visible para todos los aliases. Reasignar una variable de mapa no cambia las demás. `copiar()` conserva las referencias contenidas. Cada literal y cada campo de mapa omitido crea un mapa nuevo y escribible; los campos de mapa permiten recursión sin construir entradas predeterminadas. Los métodos mutadores también aceptan receptores temporales. Estas operaciones incorporadas no satisfacen requisitos de métodos de interfaces; los mapas sí se pueden almacenar en interfaces vacías e inspeccionar con `como`.

`repetir (mapa) |valor, clave|` liga primero el valor y después la clave; con una sola variable liga el valor. El receptor se evalúa una vez, las variables son locales al ciclo y reasignarlas no modifica la entrada. El orden no está garantizado, tampoco en `claves()` y `valores()`, y dos llamadas separadas no prometen correspondencia por índice. Durante un recorrido, las entradas eliminadas antes de visitarlas se omiten; las entradas nuevas pueden visitarse o no. `romper` y `continuar` conservan sus reglas habituales.

## Funciones y métodos

Los parámetros siempre declaran su tipo. Un tipo después de `)` declara el resultado. Si no aparece, la función no devuelve ningún valor. La última expresión de una función con resultado se devuelve implícitamente.

```cometa
fn sumar(a entero, b entero) entero a + b

fn inicio()
	imprimir("hola")
```

Una función anidada dentro de un `tipo` es un método. Dentro de ella, `@nombre` accede a un campo o método del receptor. Los nombres sin `@` solo pueden referirse a parámetros; no se realiza una búsqueda implícita de campos.

`@` solo (sin nombre) es el propio receptor como valor: tiene el tipo del receptor y puede asignarse a una variable, pasarse como argumento o devolverse, para guardar o compartir una referencia a la instancia actual. No puede asignarse (`@ = ...`) ni llamarse directamente (`@()`).

Una línea que contiene únicamente un tipo de estructura declara un campo embebido, como en Go:

```cometa
tipo Persona
	nombre cadena
	fn saludar() imprimir(@nombre)

tipo Empleado
	Persona
	puesto cadena
	fn presentar() @saludar()

fn inicio()
	var empleado = Empleado {Persona: {nombre: "Ana"}, puesto: "Ingeniera"}
	empleado.nombre = "Luis"
	empleado.saludar()
	imprimir(empleado.Persona.nombre)
```

- El nombre implícito del campo es el nombre del tipo, con su capitalización original en Cometa. Para `Caja<entero>` es `Caja`; no se permiten dos instanciaciones del mismo tipo base en una estructura. El nombre no puede duplicar un campo o método directo.
- Solo se pueden embeber estructuras declaradas, incluidas instanciaciones genéricas. No se permiten tipos primitivos, enums, interfaces, listas, opcionales, resultados ni parámetros de tipo sin instanciar. La incorporación de interfaces dentro de `interfaz` conserva sus reglas propias.
- Los campos y métodos se promueven recursivamente para lecturas, asignaciones, llamadas y acceso mediante `@`. Los miembros directos ocultan a los promovidos; gana el único miembro a menor profundidad. Campos y métodos comparten este espacio de nombres. Si hay varias rutas a esa profundidad, el miembro es ambiguo, incluso cuando llegan a la misma declaración. La ambigüedad se diagnostica al usar el selector; las rutas explícitas siguen disponibles.
- Los métodos promovidos no ambiguos participan en la implementación estructural de interfaces y restricciones genéricas. Conservan sus parámetros nombrados, valores predeterminados y receptor original; los métodos del tipo embebido no despachan a los métodos del contenedor.
- Los literales solo aceptan nombres de campos directos: `{Persona: persona}` es válido; `{nombre: "Ana"}` no inicializa el campo promovido. El tipo esperado se propaga al literal anidado.
- Omitir un campo embebido crea una estructura nueva con los valores predeterminados habituales. Los campos requeridos se diagnostican con su ruta, por ejemplo `Persona.estado`; los ciclos de campos obligatorios siguen prohibidos. Suministrar una instancia conserva la referencia compartida.
- Go recibe un campo anónimo de tipo puntero, por ejemplo `*Persona` o `*Caja[int64]`, con la convención de nombres exportados del compilador. El LSP incluye miembros promovidos no ambiguos en completado y hover, y el campo embebido en el esquema del documento.

```cometa
tipo Contador
	valor entero
	fn incrementar(cantidad entero)
		@valor = @valor + cantidad
```

La función superior `inicio` debe escribirse exactamente como `fn inicio()` y se genera como `func main()`. `imprimir(valor)` acepta un valor y lo escribe con un formateador propio de Cometa: `verdadero`/`falso`, cadenas entre comillas dentro de colecciones, listas `[1, 2]`, mapas `["a": 1]`, estructuras `Mascota {nombre: "Toby"}` y opcionales `Alguno(1)`/`Ninguno`. Las variantes de enum aún usan el formato de Go.

## Condicionales

Las condiciones deben ser `bool`. Los paréntesis son opcionales. `osi` pertenece al `si` alineado anterior y `sino` es opcional cuando el condicional se usa como sentencia.

```cometa
si (@edad < 18) @activo = falso
osi (@edad == 18) @activo = verdadero
sino
	@activo = verdadero
```

Un condicional también puede producir un valor. En ese caso requiere `sino` y todas sus ramas deben producir el mismo tipo.

```cometa
@activo = si (@edad < 18) falso sino verdadero
```

Una función con resultado puede terminar en un condicional multilínea. Cada camino debe terminar en una expresión compatible con el resultado declarado.

```cometa
fn limitar(numero entero) entero
	si (numero < 0) 0
	osi (numero > 10) 10
	sino numero
```

## Enums y casos exhaustivos

`enum` declara un tipo con un conjunto cerrado y no vacío de variantes. Cada variante tiene un nombre único y puede declarar un único tipo de payload explícito: `entero`, `decimal`, `cadena`, `bool`, una estructura, una lista u otro enum. El nombre `_` está reservado para el patrón comodín. Los enums comparten el espacio de nombres de tipos con `tipo`.

```cometa
tipo Buton_Presionado
	caracter cadena

enum Evento
	Cargar_Pagina
	Buton_Presionado Buton_Presionado
	Texto cadena
	Cantidad entero
	Activado bool
	Eventos [Evento]
```

Una variante sin payload se construye como `Evento.Cargar_Pagina`, sin paréntesis. Una variante con payload requiere exactamente un argumento compatible, por ejemplo `Evento.Texto("hola")` o `Evento.Buton_Presionado(Buton_Presionado {caracter: "a"})`. El nombre de una variante nunca infiere automáticamente un tipo de payload. No hay conversión implícita desde un payload a un enum.

Los enums pueden usarse en parámetros, resultados, campos, variables y listas. Como los demás tipos declarados, se representan mediante referencias en Go. Un payload de estructura conserva la referencia original: modificar un campo mediante el payload modifica el objeto compartido; reasignar la variable ligada solo cambia esa variable local. Los payloads escalares se almacenan por valor; las listas conservan las reglas de las listas existentes. Los detalles internos del enum no son campos accesibles en Cometa.

`casos` evalúa su operando una sola vez y ejecuta una sola rama. Las etiquetas se indentan debajo de `casos`; cada etiqueta usa `Evento.Variante`, `.Variante` o `_`, seguida de `=>`. No se aceptan nombres de variantes sin punto. El enum calificado debe coincidir con el tipo del operando; ambas formas identifican la misma variante para detectar duplicados y cobertura. Después de `=>`, el cuerpo contiene una sentencia o expresión en la misma línea, o un bloque indentado adicional cuando empieza en la línea siguiente. Una dedentación termina el bloque correspondiente.

Los constructores aceptan `.Variante` y `.Variante(payload)` cuando existe un tipo enum esperado: argumentos de funciones y métodos, variables tipadas, asignaciones, retornos implícitos, campos de estructuras, elementos de listas y resultados de ramas. Se conserva el orden de inferencia contextual existente; no hay búsqueda global ni inferencia hacia atrás. `var ip = .V4` e `imprimir(.V4)` son errores por falta de tipo enum esperado. Las variantes unitarias no admiten paréntesis y las demás requieren exactamente un payload del tipo declarado.

```cometa
fn describir(evento Evento) cadena
	casos evento |e|
		.Cargar_Pagina => "cargando pagina"
		.Buton_Presionado =>
			imprimir("buton presionado:")
			e.caracter
		.Texto => e
		_ => "otro evento"
```

La ligadura opcional `|e|` existe únicamente dentro de las ramas explícitas con payload, donde tiene el tipo de ese payload. No existe en variantes sin payload ni en `_`, y no puede repetir un nombre local visible. Las variables declaradas dentro de una rama no escapan de ella.

Todo `casos`, incluso como sentencia, debe cubrir todas las variantes. `_` es opcional, debe ser la última rama y cubre las variantes restantes. Se rechazan variantes desconocidas, ramas duplicadas y ramas después de `_`. Al añadir una variante a un enum, sus matches sin comodín deben actualizarse.

`casos` también produce valores en asignaciones, argumentos, listas y retornos implícitos. Todas las ramas deben terminar en un valor del mismo tipo; pueden contener sentencias previas y terminar en otro `casos` o un `si` completo. Un tipo esperado se propaga a los resultados para inferir literales de estructuras y listas vacías.

```cometa
fn texto(evento Evento) cadena
	var resultado = casos evento |e|
		.Texto => e
		_ => "otro evento"
	resultado
```

Dentro de un `casos` usado como sentencia, `romper` y `continuar` siguen controlando el `repetir` más cercano. Un `casos` usado como expresión no puede transferir control hacia un ciclo exterior; sí puede contener y controlar sus propios ciclos.

`casos` acepta enums, opcionales y resultados. No hay patrones de literales primitivos, guardas, desestructuración anidada, métodos de enums ni operadores de igualdad entre enums u otros wrappers. El comodín descarta los payloads restantes.

## Opcionales, errores y retornos explícitos

Los opcionales y resultados son constructores de tipos incorporados; no requieren genéricos definidos por el usuario. Un valor de tipo `T` siempre contiene un valor válido. Las representaciones internas de Go no exponen `nil`, etiquetas ni campos de payload al programa Cometa.

| Tipo | Significado |
| --- | --- |
| `T?` | Valor de `T` o ausencia |
| `T!` | Valor de `T` o error `cadena`; equivale a `T!cadena` |
| `T!E` | Valor de `T` o error de tipo `E` |
| `!` / `!E` | Éxito sin valor o error |
| `T?!` | Resultado cuyo éxito contiene un opcional |
| `(T!)?` | Opcional que contiene un resultado |

`?` envuelve el tipo anterior. `!E` introduce un resultado. El tipo de error se escribe inmediatamente después de `!`, sin espacio; así `fn f(n entero) entero! n` tiene resultado con error cadena y cuerpo `n`. Los errores compuestos y resultados repetidos se agrupan con paréntesis, por ejemplo `T!(E?)` y `(T!E)!F`. `T!E?` no sustituye a una agrupación explícita. La forma sin valor `!E` también puede usarse en campos, listas y parámetros.

Los constructores contextuales son `.Alguno(valor)` y `.Ninguno` para opcionales, y `.Ok(valor)` y `.Error(error)` para resultados. El éxito sin valor se escribe `.Ok`, sin paréntesis. Requieren un tipo esperado. Una expresión de tipo `T` se convierte implícitamente a presencia o éxito cuando se espera `T?` o `T!E`. Se añade una sola capa por conversión; un destino anidado no convierte recursivamente un valor simple. Por ejemplo, `entero?!` acepta `.Ok(.Ninguno)` o `.Ok(1)`, pero no `1` directamente. Un opcional puede envolverse en resultado si es exactamente su tipo de éxito, conservando su ausencia interna; nunca se transforma ausencia en error automáticamente.

```cometa
fn buscar(existe bool) Usuario?
	si existe
		Usuario {nombre: "Ana"}
	sino
		.Ninguno

fn cargar(existe bool) Usuario!
	si existe retornar Usuario {nombre: "Luis"}
	.Error("no disponible")

fn nombre(existe bool) cadena!
	var usuario = intentar cargar(existe)
	usuario.nombre
```

Un wrapper no permite acceder directamente a campos, métodos o elementos de su payload, ni usarlo en operaciones que requieren ese payload. No tiene conversión implícita a `bool`. No existen operaciones de extracción que provoquen un pánico como alternativa al manejo explícito.

- `casos` exige las dos variantes: `.Alguno`/`.Ninguno` o `.Ok`/`.Error`, o un comodín final explícito. La ligadura solo existe en ramas con payload. Los payloads de estructura y enum conservan sus referencias; copiar o reasignar un wrapper no cambia otro wrapper.
- `si opcional |valor|` liga el valor únicamente dentro de la rama de presencia. `osi` también admite ligadura opcional. Una sentencia puede omitir `sino`; una expresión debe incluirlo. Las condiciones booleanas siguen usando `si condición`, sin ligadura.
- `opcional o alternativa` devuelve el payload presente o evalúa la alternativa cuando falta. `resultado capturar |error| alternativa` devuelve el éxito o evalúa la alternativa al fallar. La ligadura de error puede omitirse. La alternativa puede ser una expresión en línea o un bloque indentado; debe producir el tipo del payload o salir mediante `retornar`. Para resultados sin valor, la recuperación es un bloque de sentencias.
- `o` y `capturar` tienen la misma precedencia, inferior a los operadores booleanos, y se asocian hacia la derecha. Use paréntesis para aclarar cadenas mixtas. `intentar` tiene precedencia de prefijo e incluye llamadas y accesos posteriores en su operando: para acceder al payload use `(intentar cargar()).nombre`.
- `intentar valor` extrae una capa. La ausencia sale de la función devolviendo ausencia y exige que esta devuelva un opcional. Un error sale devolviendo ese error y exige el mismo tipo de error en el resultado de la función. El tipo de éxito de la función puede ser distinto. Una conversión entre errores o entre ausencia y error requiere manejo explícito.
- `retornar expresión` sale de la función actual desde cualquier bloque, incluso dentro de una expresión. Una función sin resultado usa `retornar` sin valor. Una función con resultado sin payload usa `retornar .Ok`. Se conservan los retornos finales implícitos. Ni `retornar` ni `intentar` se permiten en expresiones de valores predeterminados de parámetros.
- Se rechazan expresiones opcionales/resultados descartadas y variables locales de resultado (`T!E`) que nunca se leen. Las variables opcionales (`T?`) pueden declararse, copiarse y permanecer sin uso; acceder a su payload sigue requiriendo extracción explícita. Pasar, guardar, devolver o manejar explícitamente un resultado cuenta como uso. Esta comprobación no es un sistema de propiedad ni exige consumo en todos los caminos; una rama explícita puede ignorar un error.

Cada operando se evalúa una sola vez, en el orden escrito; las alternativas y los operadores booleanos mantienen evaluación condicional. Los retornos y la propagación salen de la función Cometa original, incluso en argumentos nombrados, listas, campos y `casos` usados como expresiones. `inicio` conserva su firma sin resultado y maneja los errores localmente. Consulte `examples/basico/10_opcionales_errores.cometa`.

## Ciclos

`repetir` recorre una lista y liga cada elemento a la variable escrita entre `|`. Una segunda variable opcional recibe el índice como `entero`, comenzando en `0`. Ambas variables solo existen dentro del cuerpo del ciclo.

```cometa
repetir (usuarios) |usuario|
	imprimir(usuario.nombre)

repetir (usuarios) |usuario, indice|
	si (indice == 2) continuar
	si (indice == 3) romper
	imprimir(usuario)
```

Los rangos `inicio..final` solo se permiten dentro de los paréntesis de `repetir`; no son valores que puedan guardarse, pasarse a funciones o incluirse en listas. Ambos límites deben ser expresiones `entero` y se evalúan una sola vez, de izquierda a derecha, antes del ciclo. El inicio se incluye y el final se excluye. El paso es `1` si el inicio es menor y `-1` si es mayor; límites iguales producen cero iteraciones. Se permiten límites negativos; los límites decimales se rechazan. La variable del ciclo es `entero`; una segunda variable opcional recibe el índice desde `0`. Modificar estas variables dentro del cuerpo no altera la progresión del rango.

```cometa
repetir (0..5) |i| imprimir(i) // 0, 1, 2, 3, 4
repetir (5..0) |i|
	imprimir(i) // 5, 4, 3, 2, 1
repetir (0..3) imprimir("hola") // sin variables: repite 3 veces
repetir (0..2) |valor, indice|
	imprimir(valor) // 0.5, 1.5
```

`continuar` salta a la siguiente iteración y `romper` termina el ciclo más cercano. Solo pueden usarse dentro de un `repetir`.

Sin una lista ni variables, `repetir` crea un ciclo infinito. Su cuerpo también puede escribirse en la misma línea.

```cometa
repetir imprimir("hola")
```

## Alcance del MVP

El MVP incluye declaraciones de tipos, interfaces implícitas, funciones y tipos genéricos con restricciones de interfaz, inspección segura de interfaces, enums con payloads explícitos, campos, funciones, métodos y variables locales; parámetros; llamadas; asignaciones; acceso mediante `@` y `.`, e indexación de listas; literales escalares, de estructuras y listas; inferencia contextual de literales compuestos; operadores numéricos, booleanos y de comparación; condicionales y `casos` exhaustivos como sentencias o valores; ciclos sobre listas e infinitos; listas como tipos; y retornos implícitos.

Quedan fuera por ahora un literal nulo independiente, las referencias explícitas, los operadores genéricos y la resolución de paquetes remotos de Cometa. Los ejecutables de escritorio se construyen mediante el toolchain Go.

## Módulos e importaciones

Cada archivo `.cometa` define un módulo. `usar` es una palabra reservada y solo aparece al nivel superior, antes de cualquier declaración:

```cometa
usar herramientas
usar modelos como m
usar interno/base_de_datos como bd
usar ../compartido/fechas
```

La ruta no lleva comillas ni extensión; el compilador agrega `.cometa`. Se resuelve desde la carpeta del archivo que importa, nunca desde el directorio de trabajo. Usa `/` en todas las plataformas y admite los prefijos `./` y `../` repetido. Los demás segmentos tienen forma de identificador. No admite rutas absolutas, segmentos vacíos ni espacios dentro de la ruta.

El alias predeterminado es el último segmento; `como` lo reemplaza por un identificador distinto de `_`. Los alias deben ser únicos y no pueden coincidir con declaraciones superiores. Variables y parámetros locales pueden ocultarlos en expresiones. Un namespace no es un valor.

Las declaraciones son privadas al archivo por defecto. `pub` es una palabra reservada que precede a `fn`, `tipo`, `enum`, `interfaz`, `var` o `const` para exponer la declaración. Dentro de una estructura se admiten `pub nombre Tipo`, `pub fn metodo(...)` y `pub TipoEmbebido`. Las variantes de enum y los requisitos de interfaz heredan la visibilidad del tipo y no admiten `pub` propio. Tampoco se admite `pub` en imports, variables locales, parámetros, ni `inicio`; duplicarlo es un error.

Todo el código del archivo puede acceder a sus declaraciones y miembros privados. Desde otro archivo, cada paso de una ruta de incrustación y el miembro final deben ser accesibles. Se mantienen las reglas de sombreado y ambigüedad: un miembro privado no permite acceder a otra alternativa oculta. Solo los métodos concretos `pub`, a través de rutas de incrustación `pub`, satisfacen interfaces y restricciones, incluso dentro del archivo. La inspección dinámica con `como` o `casos` tampoco expone métodos privados.

Los literales externos solo pueden inicializar campos accesibles. Los argumentos posicionales conservan el orden de los campos, sin saltar los privados. Los campos omitidos conservan sus reglas de valores predeterminados; si un campo privado requiere inicialización, la construcción debe realizarse en su módulo. Una API pública puede devolver o exponer tipos privados: los consumidores pueden usar valores inferidos y sus miembros públicos, pero no nombrar esos tipos.

El acceso a declaraciones públicas importadas requiere el alias: `m.crear()`, `m.Usuario`, `m.Caja<entero>`, `m.Usuario {nombre: "Ana"}`, `m.Evento.Texto("hola")`. Los tipos calificados funcionan en restricciones, firmas, listas, wrappers, incrustaciones, inspecciones con `como` y patrones de `casos`. Una estructura incrustada conserva como nombre de campo el nombre base del tipo: `m.Usuario` crea el campo `Usuario`. La capitalización del Go generado es un detalle de implementación y no determina la visibilidad en Cometa.

Los imports no se reexportan, no tienen efectos de inicialización y pueden quedar sin uso. Cada consumidor importa directamente los módulos cuyos nombres necesita. Solo el archivo raíz puede declarar `inicio`; una dependencia con `fn inicio()` causa error. El raíz puede omitir `inicio` al generar código sin punto de entrada.

El compilador carga cada archivo físico una sola vez, reconoce rutas equivalentes y enlaces simbólicos, y rechaza imports duplicados, autoimports y ciclos con su cadena de importación. Las declaraciones de archivos diferentes tienen identidades distintas aunque compartan nombre; las interfaces siguen siendo estructurales. Las llamadas preservan evaluación, argumentos nombrados, valores predeterminados y semántica de referencia existentes.

Todo el grafo se genera en un único archivo Go `package main`. Los nombres de las dependencias reciben prefijos internos deterministas sin rutas absolutas. Las rutas con `../` pueden salir de la carpeta raíz; los errores de lectura se señalan en `usar`. No hay manifiesto, versiones, registro de paquetes ni descarga de dependencias.
