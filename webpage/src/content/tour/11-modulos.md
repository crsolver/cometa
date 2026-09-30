---
titulo: "Módulos y pub"
orden: 11
resumen: "usar, importaciones relativas y visibilidad."
ejemplo: "basico/12_modulos.cometa"
---

`usar` importa otro archivo, relativo al actual y sin extensión. Sus declaraciones públicas quedan bajo un nombre (o un alias con `como`):

```cometa
usar modulos/temperatura
usar modulos/temperatura como temp

fn inicio()
	imprimir(temperatura.a_fahrenheit(25.0))
```

- Todo es **privado al archivo** por defecto. `pub` expone funciones, constantes, tipos y, uno a uno, campos y métodos.
- Los `usar` van antes de las declaraciones y no se reexportan.
- Solo el archivo raíz declara `inicio`. Los ciclos de importación se rechazan.
- La biblioteca estándar se importa igual: `usar std/mate`, `usar std/pincel/graficos`… (mira la [biblioteca](/biblioteca/)).

## Programa completo
