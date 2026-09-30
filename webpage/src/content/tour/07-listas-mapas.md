---
titulo: "Listas y mapas"
orden: 7
resumen: "Listas [T] y mapas [K: V] con sus métodos."
ejemplo: "basico/07_mapas.cometa"
---

Una lista guarda valores del mismo tipo. Los índices empiezan en 0 y `obtener` no falla fuera de rango: devuelve un opcional.

```cometa
var notas = [7, 9, 6]
notas.agregar(10)
imprimir(notas.obtener(10) o 0)

var nombres [cadena] = []      // una lista vacía necesita su tipo
```

Un **mapa** relaciona claves (`cadena`, `entero` o `bool`) con valores. Se escribe `[clave: valor]`, y `[:]` es el mapa vacío. Leer devuelve un opcional:

```cometa
var edades = ["Ana": 12, "Luis": 14]
edades["Marta"] = 11
imprimir(edades["Pedro"] o 0)

repetir (edades) |edad, nombre|     // el orden no está garantizado
	imprimir("${nombre}: ${edad}")
```

Métodos de mapa: `longitud`, `esta_vacia`, `contiene`, `obtener`, `eliminar`, `claves`, `valores`, `copiar` y `vaciar`. El ejemplo completo de listas está en `examples/basico/06_listas.cometa`.

## Ejemplo de mapas
