---
titulo: "Errores"
orden: 10
resumen: "T! para lo que puede fallar, o, atrapar e intentar."
ejemplo: "basico/10_errores.cometa"
---

Un opcional (`T?`, lección 7) dice que un valor puede faltar. Cuando una operación puede **fallar** y quieres explicar por qué, devuelve un **resultado** `T!` (`.Ok` / `.Error("mensaje")`).

| Forma | Uso |
| --- | --- |
| `x o 0` | Valor por defecto si hay error (igual que con opcionales) |
| `x atrapar \|error\|` | Maneja el error: lo recibe en `error` |
| `intentar x` | Si falla, la función actual devuelve ese error |

```cometa
fn dividir(a entero, b entero) entero!
	si b == 0
		retornar .Error("no se puede dividir entre cero")
	a / b

fn mitad(a entero, b entero) entero!
	var r = intentar dividir(a, b)
	r / 2
```

Descartar un resultado sin mirarlo es un error de compilación: Cometa te obliga a decidir qué pasa si algo sale mal. `T!E` permite un tipo de error propio.

## Programa completo
