# Mejoras pendientes

Asperezas del lenguaje y de la biblioteca encontradas al escribir ejemplos o programas reales. Cada entrada describe lo observado, por qué molesta y una posible solución. Al resolver una, elimínala de aquí y actualiza `specs.md`, `README.md`, los ejemplos y las pruebas.

## Lenguaje

### `imprimir` de enums y resultados usa el formato de Go

- **Observado:** `imprimir(E.B(2))` muestra `{2 2}` y un resultado `T!E` muestra `{tag payload}`; booleanos, listas, mapas, estructuras y opcionales ya usan el formato de Cometa.
- **Problema:** el Go generado no conserva los nombres de variante, así que el formateador en tiempo de ejecución no puede escribir `E.B(2)`.
- **Posible solución:** que el codegen emita un método `String()` por enum (y para resultados `Ok(...)`/`Error(...)`) que el formateador use.

## Biblioteca estándar

### `ui.campo_texto` no admite selección de texto ni portapapeles

- **Observado:** el campo de texto de `std/pincel/ui` inserta y borra caracteres y mueve el cursor con las flechas/Inicio/Fin, pero no se puede seleccionar texto (arrastrar o Mayús+flecha), hacer clic para posicionar el cursor en medio del texto, ni pegar desde el portapapeles.
- **Problema:** cubre el caso de un campo corto (nombre, valor numérico), pero cualquier edición más larga es incómoda: no hay forma de reemplazar todo el contenido de una vez ni de corregir un error a mitad de la palabra sin usar solo las flechas.
- **Posible solución:** añadir un segundo índice de selección al nodo, resaltar el rango seleccionado al dibujar, calcular la posición de clic contra las líneas medidas, y usar las funciones de portapapeles de Ebitengine para copiar/pegar.
