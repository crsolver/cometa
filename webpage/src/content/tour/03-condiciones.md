---
titulo: "Condiciones"
orden: 3
resumen: "si, osi, sino, si como valor y casos sobre valores simples."
ejemplo: "basico/03_condiciones.cometa"
---

`si` ejecuta un bloque cuando la condición es verdadera; `osi` encadena otra condición y `sino` recoge el resto. Los operadores lógicos son `&&`, `||` y `!`.

```cometa
si temperatura > 30
	imprimir("¡Hace mucho calor!")
osi temperatura > 20
	imprimir("Hace buen tiempo")
sino
	imprimir("Hace frío")
```

## `si` también produce un valor

En ese caso `sino` es obligatorio y todas las ramas deben tener el mismo tipo:

```cometa
var etapa = si edad >= 18 "adulto" sino "joven"
```

## `casos` con valores simples

Sobre enteros, cadenas y booleanos, `casos` elige entre valores concretos. Una rama puede tener varios valores y la rama final `_` recoge todo lo demás:

```cometa
casos dia
	6, 7 => imprimir("Fin de semana")
	_ => imprimir("Día laborable")
```

## Programa completo
