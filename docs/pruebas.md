# Pruebas: `std/pruebas` y `cometa probar`

Escribe pruebas en Cometa y ejecútalas con un solo comando. Pensado para ejercicios: el autor publica un archivo de pruebas y la persona que resuelve el ejercicio entrega el módulo que las pasa.

```cometa
usar solucion
usar std/pruebas

fn prueba_doble()
	pruebas.igual(4, solucion.doble(2))

fn prueba_doble_de_tres()
	pruebas.igual(6, solucion.doble(3), "doble de 3")
```

```
cometa probar pruebas.cometa [--json] [--tiempo SEGUNDOS] [--filtro texto]
```

- Se ejecuta cada función de nivel superior `prueba_*` (sin parámetros ni resultado), aislada de las demás: un fallo o un pánico no detiene el resto.
- El archivo de pruebas no declara `inicio`; importa la solución con `usar`, que debe exponer lo necesario con `pub`.
- `--tiempo` (por defecto 30) limita toda la ejecución; la prueba en curso se marca `tiempo_agotado` y las siguientes `omitida`.
- `--filtro` ejecuta solo las pruebas cuyo nombre contiene el texto.

## Afirmaciones

| Función | Falla cuando |
| --- | --- |
| `pruebas.afirmar(condicion, mensaje = "")` | la condición es falsa |
| `pruebas.igual(esperado, obtenido, mensaje = "")` | difieren (dos `entero`, `decimal`, `cadena` o `bool`; entero y decimal se comparan como decimal) |
| `pruebas.casi_igual(esperado, obtenido, tolerancia = 0.000001, mensaje = "")` | difieren en más que la tolerancia |
| `pruebas.fallar(mensaje)` | siempre |

La primera afirmación incumplida termina esa prueba.

## Códigos de salida

| Código | Significado |
| --- | --- |
| 0 | todas las pruebas pasaron |
| 1 | alguna prueba falló o terminó con error |
| 2 | el programa no compila |
| 3 | tiempo agotado |
| 4 | fallo interno (por ejemplo, `go build`) |

## Salida `--json`

La salida estándar contiene solo el informe; lo que las pruebas impriman va a la salida de error.

```json
{
  "estado": "fallo",
  "total": 2, "pasaron": 1, "fallaron": 1,
  "pruebas": [
    {"nombre": "prueba_doble", "estado": "ok", "ms": 0},
    {"nombre": "prueba_doble_de_tres", "estado": "fallo",
     "mensaje": "doble de 3: se esperaba 7 pero se obtuvo 6",
     "archivo": "pruebas.cometa", "linea": 8, "ms": 0}
  ]
}
```

`estado` global: `ok`, `fallo`, `error_compilacion` (con `diagnosticos`: `archivo`, `linea`, `columna`, `etapa`, `mensaje`), `tiempo_agotado` o `error`. Por prueba: `ok`, `fallo`, `error` (pánico o cierre inesperado), `tiempo_agotado`, `omitida`.

Como `cometa ejecutar`, requiere Go instalado.
