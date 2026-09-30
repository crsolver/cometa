---
titulo: "Tipos propios y métodos"
orden: 6
resumen: "tipo, campos con valores por defecto, métodos con @ y semántica de referencia."
ejemplo: "basico/08_tipos.cometa"
---

`tipo` agrupa datos y comportamiento. Los campos pueden tener valor por defecto.

```cometa
tipo Mascota
	nombre cadena
	energia entero = 10

	fn jugar()
		@energia = @energia - 2

fn inicio()
	var perro = Mascota {nombre: "Toby"}
	perro.jugar()
```

## Qué es `@`

Dentro de un método, `@` significa «este objeto», el que recibió la llamada. En `perro.jugar()`, `@` es `perro`.

- `@energia` es el campo `energia` del objeto; `@dormir()` llama a otro método suyo.
- **Sin `@` no se busca entre los campos.** Un nombre suelto solo puede ser un parámetro o una variable local, así que se ve de un vistazo cuál es cuál.
- `@` solo, sin nombre, es el propio objeto como valor. Sirve para pasarlo o guardarlo. No se le puede asignar (`@ = ...`).

```cometa
tipo Cuenta
	saldo entero

	fn depositar(saldo entero)
		// el parámetro "saldo" y el campo "@saldo" no se confunden
		@saldo = @saldo + saldo

	fn mostrar()
		imprimir("saldo: ${@saldo}")

	fn registrar_en(banco Banco)
		banco.cuentas.agregar(@)   // guarda una referencia a esta cuenta

tipo Banco
	cuentas [Cuenta]
```

## Semántica de referencia

Las variables de un `tipo` no contienen el objeto: **apuntan a él**. Asignar, pasar a una función o guardar en una lista comparte el mismo objeto; nada se copia.

```cometa
tipo Punto
	x entero
	y entero

fn mover(p Punto)
	p.x = p.x + 10      // modifica el objeto original

fn inicio()
	var a = Punto {x: 1, y: 2}
	var b = a           // b y a son el mismo objeto
	b.x = 99
	imprimir(a.x)       // 99

	mover(a)
	imprimir(b.x)       // 109

	var lista = [a, b]  // la lista guarda referencias, no copias
	lista[0].y = 7
	imprimir(a.y)       // 7

	var c = a.copiar()  // objeto nuevo e independiente
	c.x = 0
	imprimir(a.x)       // 109: a no cambió
```

Todo `tipo` trae `copiar()`. La copia es **superficial**: si el objeto tiene campos que son otros objetos, la copia comparte esos.

## Embeber tipos

Para reutilizar campos y métodos, un tipo puede **embeber** otro escribiendo su nombre solo en una línea. Es composición, no herencia: el tipo de fuera *contiene* al de dentro y sus miembros se promueven.

```cometa
tipo Persona
	nombre cadena

	fn saludar()
		imprimir("Hola, soy ${@nombre}")

tipo Empleado
	Persona              // embebe Persona
	puesto cadena

	fn presentar()
		@saludar()       // método promovido, se llama con @
		imprimir("Trabajo de ${@puesto}")

fn inicio()
	var ana = Empleado {Persona: {nombre: "Ana"}, puesto: "ingeniera"}

	ana.saludar()                // promovido: sin escribir .Persona
	ana.nombre = "Ana María"     // también se puede asignar
	imprimir(ana.Persona.nombre) // ruta explícita: el campo se llama Persona

	// Como los objetos son referencias, se puede compartir lo embebido:
	var base = Persona {nombre: "Luis"}
	var jefe = Empleado {Persona: base, puesto: "gerente"}
	base.nombre = "Luis Pérez"
	imprimir(jefe.nombre)        // Luis Pérez: jefe.Persona es el mismo objeto que base
```

- En el literal se usa el nombre del tipo embebido (`Persona: {...}`); los campos promovidos no se pueden inicializar directamente.
- Si el tipo de fuera declara un miembro con el mismo nombre, ese gana sobre el promovido.
- Los métodos promovidos cuentan para cumplir interfaces (lección 11).

Todo es privado a su archivo salvo lo marcado con `pub` (lección 12).

## Programa completo
