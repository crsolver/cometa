import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { z } from 'astro/zod';

const tour = defineCollection({
	loader: glob({ pattern: '*.md', base: './src/content/tour' }),
	schema: z.object({
		titulo: z.string(),
		orden: z.number(),
		resumen: z.string(),
		/** Ruta bajo examples/ de un programa completo que se muestra al final. */
		ejemplo: z.string().optional(),
	}),
});

// Las guías son los Markdown de docs/ del repositorio: una sola fuente de verdad.
const guias = defineCollection({
	loader: glob({ pattern: '*.md', base: '../docs' }),
});

const referencia = defineCollection({
	loader: glob({ pattern: 'specs.md', base: '..' }),
});

export const collections = { tour, guias, referencia };
