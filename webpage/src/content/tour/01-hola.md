---
titulo: "Hola, mundo"
orden: 1
resumen: "Tu primer programa: la función inicio y cómo ejecutarlo."
ejemplo: "basico/01_hola.cometa"
---

Todo programa Cometa empieza en la función `inicio`. Los bloques se marcan con **tabuladores**.

```cometa
fn inicio()
	imprimir("¡Hola, mundo!")
```

Guárdalo como `hola.cometa` y ejecútalo:

```console
cometa ejecutar hola.cometa
```

## Los comandos principales

| Comando | Qué hace |
| --- | --- |
| `cometa nuevo nombre` | Crea una carpeta con un programa inicial |
| `cometa ejecutar archivo.cometa` | Compila y ejecuta |
| `cometa construir archivo.cometa -o salida` | Genera un ejecutable nativo |
| `cometa compilar archivo.cometa -o salida.go` | Muestra el código Go que produce Cometa |
| `cometa probar pruebas.cometa` | Ejecuta pruebas (lección 13) |

## Comentarios
Usa '//' para escribir un comentario de una línea
```cometa
// Esto es un comentario
```


## Programa completo
