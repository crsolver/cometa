import type { LanguageRegistration } from 'shiki';
import gramatica from './cometa.tmLanguage.json';

/** Gramática TextMate de Cometa, copiada de vscode-extension/syntaxes. */
export const cometa = {
	...gramatica,
	name: 'cometa',
} as unknown as LanguageRegistration;
