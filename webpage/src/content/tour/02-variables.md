---
titulo: "Variables, tipos y números"
orden: 2
resumen: "var, const, los tipos básicos, interpolación y conversiones."
ejemplo: "basico/02_variables.cometa"
---

Hay cuatro tipos básicos:

| Tipo | Ejemplo | Detalle |
| --- | --- | --- |
| `entero` | `42` | Enteros de 64 bits |
| `decimal` | `1.52` | Números con decimales |
| `cadena` | `"hola"` | Texto |
| `bool` | `verdadero`, `falso` | Verdadero o falso |

```cometa
const dias_por_semana entero = 7

fn inicio()
	var nombre = "Ana"        // el tipo se deduce: cadena
	var puntos entero = 0     // o se escribe explícitamente
	puntos = puntos + 10
	imprimir("${nombre} tiene ${puntos} puntos")
```

- `${...}` dentro de una cadena inserta un valor.
- Un entero se convierte solo a decimal cuando hace falta (`7.0 / 2` da `3.5`), pero **`7 / 2` es `3`**: la división entre enteros descarta los decimales. `%` solo existe para enteros.
- `"42".a_entero()` devuelve un opcional (`.Ninguno` si no es un número); se combina con `o` para dar un valor por defecto: `"abc".a_entero() o -1`.
- `precio.formato(2)` escribe un número con dos decimales.

## Programa completo
