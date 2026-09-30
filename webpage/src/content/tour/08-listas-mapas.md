---
titulo: "Listas y mapas"
orden: 8
resumen: "Listas [T] y mapas [K: V] con sus métodos."
ejemplo: "basico/07_mapas.cometa"
---

## Listas

Una lista guarda valores del mismo tipo, en orden. Se escribe `[T]`. Los índices empiezan en 0: `notas[0]` es el primero y se puede reasignar (`notas[0] = 8`).

```cometa
var notas = [7, 9, 6]
notas.agregar(10)
notas[0] = 8

var nombres [cadena] = []      // una lista vacía necesita su tipo
nombres.agregar("Ana")
```

Leer con `notas[i]` fuera de rango es un error en tiempo de ejecución. Cuando el índice puede no existir, usa **`obtener`**, que devuelve un opcional (lección 7), y da un valor por defecto con `o`:

```cometa
imprimir(notas.obtener(10) o 0)     // 0, no falla
```

Para recorrer una lista se usa `repetir`; una segunda variable recibe el índice:

```cometa
repetir (notas) |nota|
	imprimir(nota)

repetir (nombres) |nombre, i|
	imprimir("${i}: ${nombre}")
```

| Método | Resultado | Qué hace |
| --- | --- | --- |
| `longitud()` / `esta_vacia()` | `entero` / `bool` | Tamaño de la lista |
| `contiene(v)` | `bool` | ¿Está el valor? |
| `buscar_indice(v)` | `entero?` | Primer índice, o `.Ninguno` |
| `obtener(i)` | `T?` | Elemento, o `.Ninguno` si `i` no es válido |
| `primero()` / `ultimo()` | `T?` | Elementos de los extremos |
| `agregar(v)` | — | Añade al final |
| `extender(otra)` | — | Añade todos los elementos de otra lista |
| `insertar(i, v)` | `bool` | Inserta antes de `i`; `falso` si el índice no es válido |
| `eliminar(i)` | `bool` | Quita el elemento en `i` |
| `copiar()` | `[T]` | Copia independiente |
| `invertir()` | — | Da la vuelta a la lista |

## Mapas

Un **mapa** relaciona claves (`cadena`, `entero` o `bool`) con valores. Se escribe `[clave: valor]`, y `[:]` es el mapa vacío. Leer devuelve un opcional, porque la clave puede no existir:

```cometa
var edades = ["Ana": 12, "Luis": 14]
edades["Marta"] = 11
imprimir(edades["Pedro"] o 0)

var conteo [cadena: entero] = [:]     // vacío: necesita su tipo
conteo["el"] = (conteo["el"] o 0) + 1  // contar: leer con o, escribir sumando

repetir (edades) |edad, nombre|     // valor primero, clave después; sin orden fijo
	imprimir("${nombre}: ${edad}")
```

| Método | Resultado | Qué hace |
| --- | --- | --- |
| `longitud()` / `esta_vacia()` | `entero` / `bool` | Cantidad de entradas |
| `contiene(k)` | `bool` | ¿Existe la clave? |
| `obtener(k)` | `V?` | Igual que `mapa[k]` |
| `eliminar(k)` | `bool` | Quita la entrada; dice si existía |
| `claves()` / `valores()` | lista | Todas las claves / valores |
| `copiar()` | mapa | Copia superficial e independiente |
| `vaciar()` | — | Borra todas las entradas |

Los mapas se comparten como los objetos: si pasas un mapa a una función y esta lo cambia, tú ves el cambio. Usa `copiar()` para evitarlo. El ejemplo de listas completo está en `examples/basico/06_listas.cometa`.

## Ejemplo de mapas
