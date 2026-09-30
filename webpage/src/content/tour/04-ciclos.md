---
titulo: "Ciclos"
orden: 4
resumen: "repetir sobre rangos y listas, mientras, continuar y romper."
ejemplo: "basico/04_bucles.cometa"
---

`repetir` recorre rangos y listas. El rango `1..6` cuenta de 1 a 5 (el final no se incluye), y `|nombre|` da nombre al valor de cada vuelta:

```cometa
repetir (1..6) |numero|
	imprimir(numero)

repetir (frutas) |fruta, indice|
	imprimir("${indice}: ${fruta}")
```

`mientras` repite mientras la condición sea verdadera. Para sumar o restar basta con `+=`, `-=`, `*=`, `/=` y `%=`:

```cometa
var vidas = 3
mientras vidas > 0
	vidas -= 1
```

`repetir` sin nada es un ciclo infinito. `continuar` salta a la siguiente vuelta y `romper` termina el ciclo más cercano.

## Programa completo
