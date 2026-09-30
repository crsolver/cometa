---
titulo: "Interfaces y genéricos"
orden: 11
resumen: "Tipos distintos con el mismo comportamiento, y código que funciona con cualquier tipo."
ejemplo: "basico/11_interfaces.cometa"
---

## Interfaces

Una `interfaz` describe **qué sabe hacer** algo, sin decir cómo. Es una lista de firmas de métodos:

```cometa
interfaz Forma
	fn area() decimal
	fn nombre() cadena
```

Un tipo cumple una interfaz **sin declararlo**: basta con que tenga esos métodos y que sean públicos (`pub`). No hay `implementa` ni herencia; se comprueba por la forma.

```cometa
tipo Circulo
	radio decimal
	pub fn area() decimal 3.14159 * @radio * @radio
	pub fn nombre() cadena "círculo"

tipo Cuadrado
	lado decimal
	pub fn area() decimal @lado * @lado
	pub fn nombre() cadena "cuadrado"
```

Ahora una función o una lista pueden trabajar con cualquier `Forma`, sin saber cuál es:

```cometa
fn describir(forma Forma)
	imprimir("Un ${forma.nombre()} de área ${forma.area()}")

fn inicio()
	var formas [Forma] = [Circulo {radio: 2}, Cuadrado {lado: 3}]
	repetir (formas) |forma|
		describir(forma)
```

Detalles útiles:

- Con una `Forma` solo puedes llamar a los métodos de `Forma`, aunque el objeto de dentro tenga más.
- Un objeto guardado en una interfaz sigue siendo el mismo objeto (semántica de referencia), no una copia.
- Una interfaz puede incorporar otras escribiendo su nombre en una línea. Una interfaz sin métodos (`interfaz Cualquiera`) acepta cualquier valor.
- Una interfaz nunca es «nula»: si algo puede faltar se escribe `Forma?` (lección 7).

## Averiguar qué hay dentro

A veces necesitas el tipo concreto. `valor como Circulo` devuelve un `Circulo?`: `.Ninguno` si no era un círculo, nunca un error. Se combina con `si` o con `o`:

```cometa
si forma como Circulo |c|
	imprimir("radio ${c.radio}")
```

`casos` sobre una interfaz prueba tipos en orden y exige un `_` final, porque en el futuro pueden aparecer tipos nuevos:

```cometa
casos forma |f|
	Circulo => imprimir("círculo de radio ${f.radio}")
	Cuadrado => imprimir("cuadrado de lado ${f.lado}")
	_ => imprimir("otra forma")
```

## Genéricos

Un **parámetro de tipo** `<T>` deja el tipo para más tarde. Así escribes una vez lo que sirve para muchos tipos.

En funciones, `T` se deduce de los argumentos:

```cometa
fn primero_o<T>(lista [T], respaldo T) T
	lista.primero() o respaldo

fn inicio()
	imprimir(primero_o([4, 5], 0))          // T = entero
	imprimir(primero_o(["a"], "nada"))      // T = cadena
```

En tipos, el argumento se escribe al usarlos:

```cometa
tipo Caja<T>
	valor T
	pub fn obtener() T @valor

fn inicio()
	var caja = Caja<cadena> {valor: "hola"}
	imprimir(caja.obtener())
```

También hay `enum` e `interfaz` genéricos (`enum Evento<T>`, `interfaz Proveedor<T>`).

### Restricciones

Un `T` sin más es una caja cerrada: puedes guardarlo, pasarlo y devolverlo, pero no llamarle métodos. Para eso lo restringes a una interfaz con `<T Interfaz>`:

```cometa
fn mayor<T Forma>(a T, b T) T
	si a.area() >= b.area() a
	sino b
```

Ahora `mayor` acepta dos `Circulo` o dos `Cuadrado`, pero no mezclarlos: ambos son el mismo `T`. Si necesitas mezclar, usa la interfaz directamente (`fn mayor(a Forma, b Forma)`).

Cuándo usar cada cosa: la **interfaz** sirve para tratar tipos distintos de la misma manera; el **genérico** para que una misma estructura o función funcione con cualquier tipo sin perder cuál es.

Por ahora los genéricos no permiten operadores (`+`, `<`, `==`) sobre `T`; para eso restringe a una interfaz con un método.

## Programa completo
