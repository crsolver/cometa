# Cometa para VS Code

Soporte para [Cometa](https://github.com/crsolver/cometa), un lenguaje de programación con sintaxis en español pensado para aprender y para hacer juegos 2D.

- Colores de sintaxis e ícono para los archivos `.cometa`.
- Errores mientras escribes, con sugerencias («¿quisiste decir…?»).
- Autocompletado, ayuda al pasar el cursor e ir a la definición, también dentro de la biblioteca estándar.
- Sangría con tabuladores, como exige el lenguaje.

La extensión incluye el servidor de lenguaje. Para ejecutar tus programas necesitas además el programa `cometa` y Go; los pasos están en la [guía para empezar](https://github.com/crsolver/cometa/blob/main/docs/empezar.md).

## Ajustes

- `cometa.server.path`: ruta absoluta a otro ejecutable de `cometa`. Si está vacío se usa el que viene con la extensión.

Si algo falla, mira **Salida → Cometa Language Server**.
