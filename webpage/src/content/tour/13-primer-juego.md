---
titulo: "Tu primer juego"
orden: 13
resumen: "Una ventana y un personaje con Pincel."
ejemplo: "pincel/04_teclado.cometa"
---

Pincel es el motor de juegos 2D de la biblioteca estándar. Un juego es un tipo con dos métodos públicos: `actualizar(dt)` cambia el estado y `pintar()` dibuja, unas 60 veces por segundo.

```cometa
usar std/pincel
usar std/pincel/graficos

tipo Ventana
	pub fn actualizar(dt decimal) retornar

	pub fn pintar()
		graficos.limpiar(.AzulNoche)
		graficos.texto_depuracion("¡Hola desde Pincel!", 110, 80)

fn inicio()
	pincel.ejecutar(Ventana {}, 320, 180, titulo = "Mi ventana", escala = 3) atrapar |error|
		imprimir(error)
```

Cada módulo se importa por separado (`std/pincel/graficos`, `entrada`, `audio`…). Sigue con las guías: [juegos](/guias/juegos/), [pixel art con lienzo](/guias/lienzo/), [interfaces](/guias/ui/) y [curvas de animación](/guias/curvas/); y explora todos los módulos en la [biblioteca](/biblioteca/).

## Ejemplo: mover un personaje
