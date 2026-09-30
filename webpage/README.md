# Sitio web de Cometa

Página principal del lenguaje, hecha con Astro y TypeScript.

```sh
npm install
npm run dev      # servidor local
npm run build    # genera dist/
npm run check    # revisión de tipos
```

- `src/content/tour/*.md`: lecciones del tour (`/tour/`). El campo `ejemplo` incrusta un programa completo de `examples/`.
- `docs/*.md` y `specs.md` del repositorio se publican como `/guias/` y `/referencia/` (sin copias); los enlaces relativos se reescriben en `src/lib/enlaces.mjs`.
- `/biblioteca/`: referencia de la biblioteca estándar. `npm run sync` (y `prebuild`) ejecuta `go run ./cmd/gendocs`, que genera `src/data/biblioteca.json` (ignorado por Git) desde `internal/stdlib`, y copia la gramática de VS Code. Requiere Go.
- `.github/workflows/pages.yml` publica el sitio en GitHub Pages.
- `src/ejemplos/*.cometa`: ejemplos mostrados en la página. Son programas válidos; verifícalos con `cometa compilar`.
- `src/lib/cometa.tmLanguage.json`: copia de `vscode-extension/syntaxes/cometa.tmLanguage.json`, actualizada por `npm run sync`.
- `src/lib/tema.ts`: tema de resaltado `cometa-noche`.
- `src/styles/global.css`: paleta y estilos base.

## Identidad visual

| Token | Color | Uso |
| --- | --- | --- |
| `--noche` | `#111225` | Fondo |
| `--indigo` | `#404576` | Profundidad, bordes |
| `--acero` | `#435E85` | Superficies secundarias |
| `--azul` | `#4F6B9A` | Degradados, órbita del logo |
| `--celeste` | `#5EB1C7` | Color principal, palabras clave |
| `--hielo` / `--lavanda` | `#A6DDEA` / `#B4B8F0` | Derivados claros para texto y degradados |
| `--estrella` | `#F2D08A` | Acento cálido mínimo (estrella del logo, números) |

Logo: la trayectoria del cometa forma una «C» con el núcleo brillante en la punta y una estrella dentro.

Tipografías: Baloo 2 (títulos), Inter (texto), JetBrains Mono (código).
