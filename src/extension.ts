import { execFile } from "node:child_process";
import { promisify } from "node:util";
import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";
import {
  Executable,
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const serverPath = resolveServerPath(context);
  if (!fs.existsSync(serverPath)) {
    const message = `No se encontró el servidor de Hacha en ${serverPath}. Ejecuta “npm run build” o configura hacha.server.path.`;
    void vscode.window.showErrorMessage(message);
    return;
  }

  context.subscriptions.push(vscode.workspace.registerTextDocumentContentProvider("hacha-std", {
    async provideTextDocumentContent(uri: vscode.Uri): Promise<string> {
      const modulePath = uri.path.replace(/^\//, "").replace(/\.hacha$/, "");
      const { stdout } = await promisify(execFile)(serverPath, ["biblioteca", modulePath], { windowsHide: true });
      return stdout;
    },
  }));

  const workspaceDirectory = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? context.extensionPath;
  const executable: Executable = {
    command: serverPath,
    args: ["lsp"],
    transport: TransportKind.stdio,
    options: { cwd: workspaceDirectory },
  };
  const serverOptions: ServerOptions = {
    run: executable,
    debug: executable,
  };

  const watcher = vscode.workspace.createFileSystemWatcher("**/*.hacha");
  context.subscriptions.push(watcher);

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "hacha" }],
    synchronize: { fileEvents: watcher },
    diagnosticCollectionName: "hacha",
    outputChannelName: "Hacha Language Server",
  };

  const nextClient = new LanguageClient(
    "hachaLanguageServer",
    "Hacha Language Server",
    serverOptions,
    clientOptions,
  );
  client = nextClient;
  try {
    await nextClient.start();
  } catch (error) {
    client = undefined;
    const detail = error instanceof Error ? error.message : String(error);
    void vscode.window.showErrorMessage(`No se pudo iniciar el servidor de Hacha: ${detail}`);
    return;
  }
  context.subscriptions.push(nextClient);
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
    client = undefined;
  }
}

function resolveServerPath(context: vscode.ExtensionContext): string {
  const configured = vscode.workspace.getConfiguration("hacha").get<string>("server.path", "").trim();
  const workspaceDirectory = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  if (configured) {
    const expanded = workspaceDirectory
      ? configured.replaceAll("${workspaceFolder}", workspaceDirectory)
      : configured;
    return path.isAbsolute(expanded)
      ? expanded
      : path.resolve(workspaceDirectory ?? context.extensionPath, expanded);
  }

  const executable = process.platform === "win32" ? "hacha.exe" : "hacha";
  return context.asAbsolutePath(path.join("bin", executable));
}
