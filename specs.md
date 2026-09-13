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
	referido Usuario
	amigos [Usuario]
```

## Variables y literales compuestos

`var` declara una variable local. El tipo puede inferirse desde el valor o escribirse entre el nombre y `=`. Un literal de estructura puede indicar su tipo (`Usuario { ... }`) o recibirlo de la declaración (`var usuario Usuario = { ... }`). Los campos no indicados conservan su valor cero.

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

Por ahora `casos` solo acepta enums. No hay patrones de literales primitivos, guardas, desestructuración anidada, métodos de enums ni operadores de igualdad entre enums. El comodín descarta los payloads restantes.

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

`continuar` salta a la siguiente iteración y `romper` termina el ciclo más cercano. Solo pueden usarse dentro de un `repetir`.

Sin una lista ni variables, `repetir` crea un ciclo infinito. Su cuerpo también puede escribirse en la misma línea.

```hacha
repetir imprimir("hola")
```

## Alcance del MVP

El MVP incluye declaraciones de tipos, enums con payloads explícitos, campos, funciones, métodos y variables locales; parámetros; llamadas; asignaciones; acceso mediante `@` y `.`, e indexación de listas; literales escalares, de estructuras y listas; inferencia contextual de literales compuestos; operadores numéricos, booleanos y de comparación; condicionales y `casos` exhaustivos como sentencias o valores; ciclos sobre listas e infinitos; listas como tipos; y retornos implícitos.

Quedan fuera por ahora las importaciones, los valores nulos, las referencias explícitas, los programas de varios archivos y la creación directa de ejecutables.
