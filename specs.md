# Hacha: especificación del MVP

Hacha es un lenguaje estáticamente tipado, con sintaxis en español e indentación significativa. El compilador está escrito en Go y genera un único archivo Go perteneciente a `package main`.

## Compilar

```console
hacha compilar programa.hacha
hacha compilar programa.hacha -o salida.go
```

Sin `-o`, la salida usa el mismo nombre base con la extensión `.go`. El compilador formatea y verifica los tipos del código Go antes de escribirlo. Los errores incluyen archivo, línea y columna.

El mismo ejecutable inicia el servidor LSP mediante `hacha lsp`. El servidor se comunica por entrada/salida estándar, publica los errores del compilador al abrir, cambiar o guardar un documento, ofrece el esquema jerárquico del archivo, completa campos y métodos después de `.` o de `@` dentro de métodos, y muestra los tipos inferidos de variables y las firmas de funciones al pasar el cursor. Las líneas `//` consecutivas inmediatamente anteriores a una función se muestran como su documentación.

## Indentación y comentarios

- La indentación usa exclusivamente tabuladores. Los espacios iniciales en una línea con código son un error.
- Una línea vacía o que solo contiene un comentario `//` no afecta los bloques.
- `tipo`, `enum`, `fn`, `si`, `osi`, `sino`, `casos` y `repetir` abren bloques. Las variantes de un enum y las ramas de `casos` siempre se indentan.
- Un cuerpo en la misma línea contiene una sola sentencia o expresión y termina con esa línea.
- No existen `fin`, dos puntos de declaración ni bloques de control con llaves. Las llaves delimitan literales de estructuras. `=>` se usa exclusivamente entre el patrón y el cuerpo de una rama de `casos`.

## Parámetros variádicos y argumentos nombrados

```hacha
fn sumar(base num, valores ...num) num
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

```hacha
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

## Tipos

| Hacha | Go |
| --- | --- |
| `num` | `float64` |
| `cadena` | `string` |
| `bool` | `bool` |
| `Usuario` | `*Usuario` |
| `[Usuario]` | `[]*Usuario` |

Todos los tipos declarados por el programa tienen semántica de referencia. Los tipos, campos, métodos y funciones generados se exportan en Go.

```hacha
tipo Usuario
	nombre cadena
	edad num
	activo bool
	referido Usuario?
	amigos [Usuario]
```

## Variables y literales compuestos

`var` declara una variable local. El tipo puede inferirse desde el valor o escribirse entre el nombre y `=`. Un literal de estructura puede indicar su tipo (`Usuario { ... }`) o recibirlo de la declaración (`var usuario Usuario = { ... }`). Los campos omitidos reciben un valor predeterminado válido: escalares en cero, listas vacías, opcionales ausentes y estructuras nuevas construidas recursivamente. Los campos de resultado o enum requieren inicialización explícita, incluso dentro de estructuras anidadas. El diagnóstico indica la ruta del campo que falta. Se rechazan ciclos directos e indirectos de campos de estructura obligatorios; los opcionales y las listas permiten recursión. Cada construcción crea sus propios objetos predeterminados, sin compartirlos con otras instancias.

```hacha
var usuario1 = Usuario {
	nombre: "andres",
	edad: 29
}

var usuario2 Usuario = {
	nombre: "andres",
	amigos: [usuario1]
}
```

Las listas no vacías infieren su tipo desde el primer elemento y exigen que los demás sean compatibles. Una lista vacía requiere un tipo esperado.

```hacha
var lista = []                 // error: no se puede inferir el elemento
var lista2 [Usuario] = []      // válido
var usuarios = [usuario1, usuario2]
```

Una lista se indexa con una expresión `num` entre corchetes. El acceso produce un valor del tipo de sus elementos y puede usarse dentro de otra expresión, incluso como argumento de una función.

```hacha
var primero = usuarios[0]
procesar_usuario(usuarios[1])
```

Los campos y métodos de una variable se acceden con `.`. El marcador `@` sigue reservado para el receptor del método actual.

```hacha
usuario2.activar(verdadero)
imprimir(usuario2.nombre)
```

## Funciones y métodos

Los parámetros siempre declaran su tipo. Un tipo después de `)` declara el resultado. Si no aparece, la función no devuelve ningún valor. La última expresión de una función con resultado se devuelve implícitamente.

```hacha
fn sumar(a num, b num) num a + b

fn inicio()
	imprimir("hola")
```

Una función anidada dentro de un `tipo` es un método. Dentro de ella, `@nombre` accede a un campo o método del receptor. Los nombres sin `@` solo pueden referirse a parámetros; no se realiza una búsqueda implícita de campos.

```hacha
tipo Contador
	valor num
	fn incrementar(cantidad num)
		@valor = @valor + cantidad
```

La función superior `inicio` debe escribirse exactamente como `fn inicio()` y se genera como `func main()`. `imprimir(valor)` acepta un valor y se genera como `fmt.Println(valor)`.

## Condicionales

Las condiciones deben ser `bool`. Los paréntesis son opcionales. `osi` pertenece al `si` alineado anterior y `sino` es opcional cuando el condicional se usa como sentencia.

```hacha
si (@edad < 18) @activo = falso
osi (@edad == 18) @activo = verdadero
sino
	@activo = verdadero
```

Un condicional también puede producir un valor. En ese caso requiere `sino` y todas sus ramas deben producir el mismo tipo.

```hacha
@activo = si (@edad < 18) falso sino verdadero
```

Una función con resultado puede terminar en un condicional multilínea. Cada camino debe terminar en una expresión compatible con el resultado declarado.

```hacha
fn limitar(numero num) num
	si (numero < 0) 0
	osi (numero > 10) 10
	sino numero
```

## Enums y casos exhaustivos

`enum` declara un tipo con un conjunto cerrado y no vacío de variantes. Cada variante tiene un nombre único y puede declarar un único tipo de payload explícito: `num`, `cadena`, `bool`, una estructura, una lista u otro enum. El nombre `_` está reservado para el patrón comodín. Los enums comparten el espacio de nombres de tipos con `tipo`.

```hacha
tipo Buton_Presionado
	caracter cadena

enum Evento
	Cargar_Pagina
	Buton_Presionado Buton_Presionado
	Texto cadena
	Cantidad num
	Activado bool
	Eventos [Evento]
```

Una variante sin payload se construye como `Evento.Cargar_Pagina`, sin paréntesis. Una variante con payload requiere exactamente un argumento compatible, por ejemplo `Evento.Texto("hola")` o `Evento.Buton_Presionado(Buton_Presionado {caracter: "a"})`. El nombre de una variante nunca infiere automáticamente un tipo de payload. No hay conversión implícita desde un payload a un enum.

Los enums pueden usarse en parámetros, resultados, campos, variables y listas. Como los demás tipos declarados, se representan mediante referencias en Go. Un payload de estructura conserva la referencia original: modificar un campo mediante el payload modifica el objeto compartido; reasignar la variable ligada solo cambia esa variable local. Los payloads escalares se almacenan por valor; las listas conservan las reglas de las listas existentes. Los detalles internos del enum no son campos accesibles en Hacha.

`casos` evalúa su operando una sola vez y ejecuta una sola rama. Las etiquetas se indentan debajo de `casos`; cada etiqueta usa `Evento.Variante`, `.Variante` o `_`, seguida de `=>`. No se aceptan nombres de variantes sin punto. El enum calificado debe coincidir con el tipo del operando; ambas formas identifican la misma variante para detectar duplicados y cobertura. Después de `=>`, el cuerpo contiene una sentencia o expresión en la misma línea, o un bloque indentado adicional cuando empieza en la línea siguiente. Una dedentación termina el bloque correspondiente.

Los constructores aceptan `.Variante` y `.Variante(payload)` cuando existe un tipo enum esperado: argumentos de funciones y métodos, variables tipadas, asignaciones, retornos implícitos, campos de estructuras, elementos de listas y resultados de ramas. Se conserva el orden de inferencia contextual existente; no hay búsqueda global ni inferencia hacia atrás. `var ip = .V4` e `imprimir(.V4)` son errores por falta de tipo enum esperado. Las variantes unitarias no admiten paréntesis y las demás requieren exactamente un payload del tipo declarado.

```hacha
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

```hacha
fn texto(evento Evento) cadena
	var resultado = casos evento |e|
		.Texto => e
		_ => "otro evento"
	resultado
```

Dentro de un `casos` usado como sentencia, `romper` y `continuar` siguen controlando el `repetir` más cercano. Un `casos` usado como expresión no puede transferir control hacia un ciclo exterior; sí puede contener y controlar sus propios ciclos.

`casos` acepta enums, opcionales y resultados. No hay patrones de literales primitivos, guardas, desestructuración anidada, métodos de enums ni operadores de igualdad entre enums u otros wrappers. El comodín descarta los payloads restantes.

## Opcionales, errores y retornos explícitos

Los opcionales y resultados son constructores de tipos incorporados; no requieren genéricos definidos por el usuario. Un valor de tipo `T` siempre contiene un valor válido. Las representaciones internas de Go no exponen `nil`, etiquetas ni campos de payload al programa Hacha.

| Tipo | Significado |
| --- | --- |
| `T?` | Valor de `T` o ausencia |
| `T!` | Valor de `T` o error `cadena`; equivale a `T!cadena` |
| `T!E` | Valor de `T` o error de tipo `E` |
| `!` / `!E` | Éxito sin valor o error |
| `T?!` | Resultado cuyo éxito contiene un opcional |
| `(T!)?` | Opcional que contiene un resultado |

`?` envuelve el tipo anterior. `!E` introduce un resultado. El tipo de error se escribe inmediatamente después de `!`, sin espacio; así `fn f(n num) num! n` tiene resultado con error cadena y cuerpo `n`. Los errores compuestos y resultados repetidos se agrupan con paréntesis, por ejemplo `T!(E?)` y `(T!E)!F`. `T!E?` no sustituye a una agrupación explícita. La forma sin valor `!E` también puede usarse en campos, listas y parámetros.

Los constructores contextuales son `.Alguno(valor)` y `.Ninguno` para opcionales, y `.Ok(valor)` y `.Error(error)` para resultados. El éxito sin valor se escribe `.Ok`, sin paréntesis. Requieren un tipo esperado. Una expresión de tipo `T` se convierte implícitamente a presencia o éxito cuando se espera `T?` o `T!E`. Se añade una sola capa por conversión; un destino anidado no convierte recursivamente un valor simple. Por ejemplo, `num?!` acepta `.Ok(.Ninguno)` o `.Ok(1)`, pero no `1` directamente. Un opcional puede envolverse en resultado si es exactamente su tipo de éxito, conservando su ausencia interna; nunca se transforma ausencia en error automáticamente.

```hacha
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
- Se rechazan expresiones opcionales/resultados descartadas y variables locales wrapper que nunca se leen. Pasar, guardar, devolver o manejar explícitamente el valor cuenta como uso. Esta comprobación no es un sistema de propiedad ni exige consumo en todos los caminos; una rama explícita puede ignorar un error.

Cada operando se evalúa una sola vez, en el orden escrito; las alternativas y los operadores booleanos mantienen evaluación condicional. Los retornos y la propagación salen de la función Hacha original, incluso en argumentos nombrados, listas, campos y `casos` usados como expresiones. `inicio` conserva su firma sin resultado y maneja los errores localmente. Consulte `examples/errores.hacha`.

## Ciclos

`repetir` recorre una lista y liga cada elemento a la variable escrita entre `|`. Una segunda variable opcional recibe el índice como `num`, comenzando en `0`. Ambas variables solo existen dentro del cuerpo del ciclo.

```hacha
repetir (usuarios) |usuario|
	imprimir(usuario.nombre)

repetir (usuarios) |usuario, indice|
	si (indice == 2) continuar
	si (indice == 3) romper
	imprimir(usuario)
```

Los rangos `inicio..final` solo se permiten dentro de los paréntesis de `repetir`; no son valores que puedan guardarse, pasarse a funciones o incluirse en listas. Ambos límites deben ser expresiones `num` y se evalúan una sola vez, de izquierda a derecha, antes del ciclo. El inicio se incluye y el final se excluye. El paso es `1` si el inicio es menor y `-1` si es mayor; límites iguales producen cero iteraciones. Se permiten límites negativos y decimales, sin redondearlos. La variable del ciclo es `num`; una segunda variable opcional recibe el índice desde `0`. Modificar estas variables dentro del cuerpo no altera la progresión del rango.

```hacha
repetir (0..5) |i| imprimir(i) // 0, 1, 2, 3, 4
repetir (5..0) |i|
	imprimir(i) // 5, 4, 3, 2, 1
repetir (0.5..2) |valor, indice|
	imprimir(valor) // 0.5, 1.5
```

`continuar` salta a la siguiente iteración y `romper` termina el ciclo más cercano. Solo pueden usarse dentro de un `repetir`.

Sin una lista ni variables, `repetir` crea un ciclo infinito. Su cuerpo también puede escribirse en la misma línea.

```hacha
repetir imprimir("hola")
```

## Alcance del MVP

El MVP incluye declaraciones de tipos, enums con payloads explícitos, campos, funciones, métodos y variables locales; parámetros; llamadas; asignaciones; acceso mediante `@` y `.`, e indexación de listas; literales escalares, de estructuras y listas; inferencia contextual de literales compuestos; operadores numéricos, booleanos y de comparación; condicionales y `casos` exhaustivos como sentencias o valores; ciclos sobre listas e infinitos; listas como tipos; y retornos implícitos.

Quedan fuera por ahora las importaciones, un literal nulo independiente, las referencias explícitas, los genéricos definidos por el usuario, los programas de varios archivos y la creación directa de ejecutables.
