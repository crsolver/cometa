---
titulo: "Opcionales"
orden: 7
resumen: "T? para lo que puede faltar, y cómo sacar el valor con o y si."
---

Cometa **no tiene valores nulos**. Una variable de tipo `entero` siempre contiene un entero. Cuando un valor puede no existir (buscar un nombre que quizá no está, leer una clave que quizá no existe), el tipo lo dice: **`T?`**, un *opcional*.

Un opcional tiene dos formas: `.Alguno(valor)` si hay valor y `.Ninguno` si no lo hay.

```cometa
fn buscar_edad(nombre cadena) entero?
	si nombre == "Ana" 12
	sino .Ninguno
```

Como el valor podría faltar, no se puede usar directamente: `buscar_edad("Ana") + 1` no compila. Hay que decidir qué hacer si falta.

## Valor por defecto con `o`

```cometa
var edad_pedro = buscar_edad("Pedro") o 0    // 0: no hay valor, se usa el de la derecha
```

## Entrar solo si hay valor con `si`

`si x |v|` ejecuta el bloque solo cuando hay valor, y lo deja disponible como `v`:

```cometa
si buscar_edad("Ana") |edad|
	imprimir("Ana tiene ${edad} años")
sino
	imprimir("No conozco a Ana")
```

## Dónde los verás

Muchas operaciones devuelven opcionales: `lista.obtener(i)`, `lista.primero()`, `mapa["clave"]`, `"42".a_entero()`. En la próxima lección los usarás con listas y mapas.

Lo que puede *fallar* con un mensaje (en lugar de simplemente faltar) se expresa con `T!`; lo verás en la lección 10.
