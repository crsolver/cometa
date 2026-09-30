---
titulo: "Pruebas"
orden: 12
resumen: "Escribe pruebas con std/pruebas y ejecútalas con cometa probar."
---

Cada función `prueba_*` de un archivo es una prueba; falla en la primera afirmación incumplida.

```cometa
usar solucion
usar std/pruebas

fn prueba_doble()
	pruebas.igual(4, solucion.doble(2))

fn prueba_doble_de_tres()
	pruebas.igual(6, solucion.doble(3), "doble de 3")
```

```console
cometa probar pruebas.cometa
```

Las afirmaciones son `pruebas.afirmar`, `pruebas.igual`, `pruebas.casi_igual` y `pruebas.fallar`. El código de salida es 0 si todo pasa. Los detalles están en la [guía de pruebas](/guias/pruebas/) y en la [referencia del módulo](/biblioteca/pruebas/).
