import type { ThemeRegistration } from 'shiki';

/** Tema de resaltado "cometa-noche", derivado de la paleta de la marca. */
export const temaCometa: ThemeRegistration = {
	name: 'cometa-noche',
	type: 'dark',
	colors: {
		'editor.background': '#2D2F50',
		'editor.foreground': '#DDE7F2',
	},
	tokenColors: [
		{ scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: '#8C95C4', fontStyle: 'italic' } },
		{
			scope: ['keyword.control', 'storage.type.function', 'storage.type.struct', 'storage.type.variable'],
			settings: { foreground: '#5EB1C7', fontStyle: 'bold' },
		},
		{ scope: ['storage.modifier'], settings: { foreground: '#B4B8F0', fontStyle: 'bold' } },
		{ scope: ['storage.type.cometa', 'support.type', 'entity.name.type'], settings: { foreground: '#9AAFE3' } },
		{ scope: ['entity.name.function', 'support.function.builtin'], settings: { foreground: '#A6DDEA' } },
		{ scope: ['string'], settings: { foreground: '#7FD3C4' } },
		{
			scope: ['constant.character.escape', 'punctuation.section.interpolation'],
			settings: { foreground: '#B4B8F0' },
		},
		{ scope: ['meta.interpolation'], settings: { foreground: '#DDE7F2' } },
		{ scope: ['constant.numeric', 'constant.language'], settings: { foreground: '#F2D08A' } },
		{ scope: ['constant.other.enum'], settings: { foreground: '#C9B6F2' } },
		{ scope: ['entity.name.namespace'], settings: { foreground: '#C9B6F2' } },
		{ scope: ['variable.other.member'], settings: { foreground: '#C4D3EC' } },
		{ scope: ['variable.other.readwrite'], settings: { foreground: '#DDE7F2' } },
		{ scope: ['keyword.operator'], settings: { foreground: '#7FA3D6' } },
		{ scope: ['punctuation'], settings: { foreground: '#A0A9CE' } },
	],
};
