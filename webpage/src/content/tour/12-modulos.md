---
titulo: "Módulos y pub"
orden: 12
resumen: "Repartir un programa en archivos con usar, y decidir qué se comparte con pub."
ejemplo: "basico/12_modulos.cometa"
---

Cuando un programa crece, se reparte en archivos. Cada archivo `.cometa` es un **módulo**, y `usar` trae uno a otro.

## Importar con `usar`

La ruta es relativa al archivo que importa, con `/` como separador y **sin extensión**. Las declaraciones del módulo quedan bajo su nombre (el último segmento de la ruta):

```cometa
usar modulos/temperatura

fn inicio()
	imprimir(temperatura.a_fahrenheit(25.0))
```

Con `como` eliges otro nombre, útil si es largo o si choca con algo tuyo:

```cometa
usar modulos/temperatura como temp

fn inicio()
	imprimir(temp.a_fahrenheit(25.0))
```

`./` y `../` suben o se quedan en la carpeta: `usar ../compartido/fechas`. Los `usar` van siempre al principio del archivo, antes de cualquier declaración.

## Qué se comparte: `pub`

Todo es **privado al archivo** por defecto. Para que otro archivo lo use, se marca con `pub`: funciones, constantes, variables, tipos, enums e interfaces. Dentro de un `tipo`, cada campo y cada método se marca por separado, así puedes esconder los detalles internos:

```cometa
// tienda/cuenta.cometa
pub tipo Cuenta
	pub titular cadena
	saldo entero            // privado: solo este archivo lo toca

	pub fn depositar(monto entero)
		si monto > 0
			@saldo += monto

	pub fn saldo_actual() entero @saldo

pub fn nueva(titular cadena) Cuenta
	Cuenta {titular: titular}

// sin pub: un detalle interno de este archivo
fn validar(monto entero) bool monto > 0
```

Se usa con el nombre del módulo delante, tanto para funciones como para tipos:

```cometa
// principal.cometa
usar tienda/cuenta como c

fn inicio()
	var ana = c.nueva("Ana")
	ana.depositar(50)
	imprimir("${ana.titular}: ${ana.saldo_actual()}")   // Ana: 50
```

Lo que no es `pub` no se puede tocar desde fuera. El compilador lo dice claramente:

```cometa
ana.saldo = 1000000     // error: el miembro "saldo" es privado
c.validar(5)            // error: la declaración "validar" es privada en el módulo …
```

Así `Cuenta` controla su saldo: solo cambia a través de `depositar`. Esa es la idea de `pub`: mostrar lo que otros necesitan y esconder el resto para poder cambiarlo después sin romper a nadie.

Desde otro archivo, un literal solo puede dar valor a los campos `pub`; los privados se quedan con su valor por defecto:

```cometa
var x = c.Cuenta {titular: "Ana"}            // válido: saldo empieza en 0
var y = c.Cuenta {titular: "Ana", saldo: 99} // error: el campo "saldo" es privado
```

Si un campo privado necesita un valor inicial concreto, el módulo debe ofrecer una función constructora, como `nueva`.

## Reglas a recordar

- Solo el archivo principal declara `inicio`.
- Los `usar` no se reexportan: si `a` usa `b`, quien use `a` no obtiene `b`; debe importarlo él.
- Los ciclos (`a` usa `b` y `b` usa `a`) y los imports repetidos se rechazan.
- `pub` no va en variables locales, parámetros ni en `usar`.
- La biblioteca estándar se importa igual: `usar std/mate`, `usar std/pincel/graficos`… (mira la [biblioteca](/biblioteca/)).

## Programa completo
