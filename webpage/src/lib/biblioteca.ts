import datos from '../data/biblioteca.json';

/** Tipos de src/data/biblioteca.json, generado por `go run ./cmd/gendocs`. */
export interface Funcion {
	nombre: string;
	firma: string;
	doc?: string;
	dibujo?: boolean;
	recurso?: boolean;
}
export interface Tipo {
	nombre: string;
	doc?: string;
	campos?: string[];
	constantes?: string[];
	metodos?: Funcion[];
	interfaz?: boolean;
}
export interface Modulo {
	nombre: string;
	ruta: string;
	doc?: string;
	tipos: Tipo[];
	constantes?: string[];
	funciones: Funcion[];
}

export const biblioteca = datos as unknown as Modulo[];
