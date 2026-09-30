---
titulo: "Opcionales y errores"
orden: 9
resumen: "T? para lo que puede faltar y T! para lo que puede fallar."
ejemplo: "basico/10_opcionales_errores.cometa"
---

Cometa no tiene valores nulos. Lo que puede faltar es un **opcional** `T?` (`.Alguno` / `.Ninguno`); lo que puede fallar es un **resultado** `T!` (`.Ok` / `.Error("mensaje")`).

| Forma | Uso |
| --- | --- |
| `x o 0` | Valor por defecto si falta |
| `si x \|v\|` | Entra al bloque solo si hay valor |
| `x atrapar 0` | Valor por defecto si hay error |
| `x atrapar \|error\|` | Un bloque que maneja el error |
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
