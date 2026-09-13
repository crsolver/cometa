const { existsSync, mkdirSync } = require("node:fs");
const { join } = require("node:path");
const { spawnSync } = require("node:child_process");

const workspace = join(__dirname, "..");
const outputDirectory = join(workspace, "bin");
const outputName = process.platform === "win32" ? "hacha.exe" : "hacha";
const outputPath = join(outputDirectory, outputName);
const cacheDirectory = join(workspace, ".cache");
const buildCache = join(cacheDirectory, "go-build");
const tempDirectory = join(cacheDirectory, "go-tmp");

const candidates = [];
if (process.env.HACHA_GO) {
  candidates.push(process.env.HACHA_GO);
}
if (process.platform === "win32") {
  candidates.push("C:\\Program Files\\Go\\bin\\go.exe");
}
candidates.push("go");

const go = candidates.find((candidate) => candidate === "go" || existsSync(candidate));
if (!go) {
  throw new Error("Go was not found. Install Go or set HACHA_GO to the Go executable.");
}

mkdirSync(outputDirectory, { recursive: true });
mkdirSync(buildCache, { recursive: true });
mkdirSync(tempDirectory, { recursive: true });
const environment = {
  ...process.env,
  GOCACHE: process.env.GOCACHE || buildCache,
  GOTMPDIR: process.env.GOTMPDIR || tempDirectory,
};
const result = spawnSync(go, ["build", "-o", outputPath, "./cmd/hacha"], {
  cwd: workspace,
  stdio: "inherit",
  env: environment,
});

if (result.error) {
  throw result.error;
}
if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

console.log(`Built Hacha language server: ${outputPath}`);
