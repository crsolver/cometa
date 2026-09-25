# Ruido 2D para niveles procedurales

`usar std/mate/ruido` importa funciones puras sin Ebitengine ni estado aleatorio compartido. Se admite `como r`; cada archivo importa el módulo explícitamente.

```cometa
fn suave(x decimal, y decimal, semilla entero = 0) decimal
fn fractal(x decimal, y decimal, semilla entero = 0, octavas entero = 4, persistencia decimal = 0.5, lacunaridad decimal = 2.0) decimal
```

Ambas devuelven valores en `[0, 1]`. Las mismas coordenadas, semilla y opciones producen el mismo resultado, sin importar el orden de las llamadas. Se aceptan semillas y coordenadas negativas; los argumentos enteros se amplían a decimal cuando corresponde.

`suave` interpola valores determinados por un hash entero en las cuatro esquinas de cada celda, con suavizado quíntico. Es ruido de valores, no ruido de gradientes. Evita muestrear únicamente en coordenadas enteras si deseas observar transiciones suaves.

`fractal` suma octavas y divide por la suma de sus amplitudes. La primera usa la semilla original; las siguientes usan semillas derivadas mediante mezcla entera fija. Una sola octava equivale a `suave`. Persistencia cero conserva solo su valor, aunque se validan las coordenadas de todas las octavas solicitadas.

## Uso

```cometa
usar std/mate/ruido

fn inicio()
	var altura = ruido.fractal(12 * 0.08, 5 * 0.08, semilla = 42)
	imprimir(si altura < 0.4 "agua" sino "tierra")
```

- **Semilla:** cambia la distribución. Guárdala junto con los parámetros para regenerar el mundo.
- **Escala:** multiplica las coordenadas; valores pequeños producen regiones grandes. Usa coordenadas del mundo para que bloques vecinos coincidan.
- **Octavas:** de 1 a 16; añaden detalle y aumentan el coste de muestreo.
- **Persistencia:** de 0 a 1; multiplica la amplitud de cada octava. Predeterminada: 0.5.
- **Lacunaridad:** finita y mayor o igual a 1; multiplica las coordenadas de cada octava. Predeterminada: 2.
- **Umbral:** convierte el resultado en terreno, pared o espacio libre. La distribución no es uniforme: un umbral de 0.4 no garantiza un 40% de agua.

Las coordenadas deben ser finitas y estar estrictamente dentro de `(-2^52, 2^52)` en todas las octavas. NaN, infinitos, expansión fuera del rango y opciones inválidas provocan un panic descriptivo en español. Cerca del límite numérico se pierde resolución fraccionaria; usa escalas razonables para mapas jugables.

Ejecuta `vscode-extension/bin/cometa ejecutar examples/ruido.cometa` (en Windows, `vscode-extension\bin\cometa.exe`) para ver terreno y cuevas en consola. Los umbrales pueden producir zonas desconectadas: comprueba conectividad y añade pasillos si el juego lo requiere. El módulo no genera mapas completos ni ofrece ruido 3D, celular o mosaicos periódicos.
