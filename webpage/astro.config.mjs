// @ts-check
import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';
import { unified } from '@astrojs/markdown-remark';
import gramatica from './src/lib/cometa.tmLanguage.json' with { type: 'json' };
import rehypeEnlaces from "./src/lib/enlaces.mjs";
import { temaCometa } from './src/lib/tema.ts';

export default defineConfig({
	site: 'https://cometa.dev',
	integrations: [sitemap()],
	vite: { server: { fs: { allow: [".."] } } },
	markdown: {
		processor: unified({ rehypePlugins: [rehypeEnlaces] }),
		shikiConfig: {
			theme: temaCometa,
			langs: [{ ...gramatica, name: 'cometa' }],
		},
	},
});
