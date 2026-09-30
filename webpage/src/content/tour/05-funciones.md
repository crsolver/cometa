---
titulo: "Funciones"
orden: 5
resumen: "Parámetros, resultados, retorno implícito, valores por defecto y argumentos con nombre."
ejemplo: "basico/05_funciones.cometa"
---

El tipo después de los paréntesis es el resultado, y **la última expresión se devuelve automáticamente**. `retornar` sale antes.

```cometa
fn sumar(a entero, b entero) entero
	a + b

fn doble(numero entero) entero numero * 2   // en una sola línea
```

Un parámetro puede tener valor por defecto, y al llamar puedes usar nombres:

```cometa
potencia(5)                            // exponente vale 2 por defecto
potencia(exponente = 3, base = 2)      // argumentos con nombre
```

## Programa completo
