// Prepara los datos generados: gramática de VS Code y referencia de la biblioteca estándar.
import { copyFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';

copyFileSync('../vscode-extension/syntaxes/cometa.tmLanguage.json', 'src/lib/cometa.tmLanguage.json');

// En esta máquina hay dos instalaciones de Go y una falla; se prueban en orden.
const candidatos = [
	process.env.GO,
	'C:/Program Files/Go/bin/go.exe',
	'go',
].filter(Boolean);

for (const go of candidatos) {
	const r = spawnSync(go, ['run', './cmd/gendocs'], { cwd: '..', stdio: 'inherit' });
	if (r.status === 0) process.exit(0);
}
console.error('No se pudo ejecutar cmd/gendocs con ninguna instalación de Go.');
process.exit(1);
