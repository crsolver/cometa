---
titulo: "Tipos propios y métodos"
orden: 6
resumen: "tipo, campos con valores por defecto, métodos con @ y semántica de referencia."
ejemplo: "basico/08_tipos.cometa"
---

`tipo` agrupa datos y comportamiento. Los campos pueden tener valor por defecto. Dentro de un método, **`@` accede al propio objeto** (`@energia`); sin `@` un nombre no se busca entre los campos.

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

Los objetos tienen **semántica de referencia**: `var mismo = perro` no copia, apunta al mismo objeto. Todo `tipo` trae `copiar()` para hacer una copia independiente.

Un tipo puede **embeber** otro escribiendo su nombre en una línea (por ejemplo `Persona`); sus campos y métodos se promueven al tipo que lo contiene.

Todo es privado a su archivo salvo lo marcado con `pub` (lección 11).

## Programa completo
