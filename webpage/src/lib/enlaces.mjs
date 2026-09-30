import { GITHUB } from './sitio.ts';

/** Reescribe los enlaces relativos de docs/*.md y specs.md para que funcionen en el sitio. */
export function reescribirEnlaces(href) {
	if (/^(https?:|mailto:|#|\/)/.test(href)) return href;
	const [ruta, ancla = ''] = href.split('#');
	const sufijo = ancla ? `#${ancla}` : '';
	const guia = /^(?:\.\/|docs\/)?([a-z_]+)\.md$/.exec(ruta);
	if (guia) return `/guias/${guia[1]}/${sufijo}`;
	if (/^(\.\.\/)?specs\.md$/.test(ruta)) return `/referencia/${sufijo}`;
	const externo = /^(?:\.\.\/)?((?:examples|docs|internal)\/.*)$/.exec(ruta);
	if (externo) return `${GITHUB}/blob/main/${externo[1]}${sufijo}`;
	return href;
}

export default function rehypeEnlaces() {
	return (arbol) => {
		const visitar = (nodo) => {
			if (nodo.type === 'element' && nodo.tagName === 'a' && typeof nodo.properties?.href === 'string') {
				nodo.properties.href = reescribirEnlaces(nodo.properties.href);
			}
			nodo.children?.forEach(visitar);
		};
		visitar(arbol);
	};
}
