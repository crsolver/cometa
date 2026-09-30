---
titulo: "Interfaces y genéricos"
orden: 10
resumen: "Interfaces estructurales, genéricos con restricciones e inspección segura de tipos."
ejemplo: "basico/11_interfaces.cometa"
---

Una `interfaz` describe métodos. Cualquier tipo que los tenga (públicos) la cumple, sin declararlo:

```cometa
interfaz Forma
	fn area() decimal
	fn nombre() cadena
```

Funciones, tipos, enums e interfaces aceptan parámetros de tipo con `<T>`, opcionalmente restringidos: `<T Forma>`.

```cometa
tipo Caja<T>
	valor T
```

Para inspeccionar con seguridad qué hay dentro de una interfaz: `valor como Circulo` devuelve un `Circulo?`, y `casos` sobre una interfaz usa patrones de tipo con una rama final `_`.

## Programa completo
