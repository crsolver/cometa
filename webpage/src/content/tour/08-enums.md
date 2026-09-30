---
titulo: "Enums y casos"
orden: 8
resumen: "Variantes con datos y casos exhaustivos."
ejemplo: "basico/09_enums.cometa"
---

Un `enum` es un valor que puede ser una de varias variantes; cada una puede llevar **un dato** con tipo explícito.

```cometa
enum Figura
	Circulo decimal
	Cuadrado decimal
	Punto
```

`casos` debe cubrir todas las variantes (o terminar en `_`). Con `|dato|` accedes al dato de la variante:

```cometa
fn area(figura Figura) decimal
	casos figura |dato|
		.Circulo => 3.14159 * dato * dato
		.Cuadrado => dato * dato
		.Punto => 0
```

Se construyen con `Figura.Circulo(1)`, o solo `.Circulo(1)` cuando el tipo esperado es evidente. Para comprobar una variante sin dato basta `==`: `si luz == .Rojo`.

## Programa completo
