// Run after npm run build: node scripts/smoke-lsp.js
const { spawn } = require("node:child_process");
const { join } = require("node:path");

const executable = join(__dirname, "..", "bin", process.platform === "win32" ? "cometa.exe" : "cometa");
const child = spawn(executable, ["lsp", "--stdio"], { windowsHide: true });
let buffered = Buffer.alloc(0);
let stderr = "";
let stage = "initialize";
let failed = false;

function fail(message) {
  if (failed) return;
  failed = true;
  clearTimeout(timer);
  console.error(message);
  child.kill();
  process.exitCode = 1;
}

const timer = setTimeout(() => fail(`LSP smoke test timed out during ${stage}. ${stderr}`), 15000);

function send(message) {
  const body = Buffer.from(JSON.stringify({ jsonrpc: "2.0", ...message }));
  child.stdin.write(`Content-Length: ${body.length}\r\n\r\n`);
  child.stdin.write(body);
}

child.on("error", (error) => fail(error.message));
child.stdin.on("error", (error) => fail(error.message));
child.stderr.on("data", (data) => { stderr += data.toString(); });
child.stdout.on("data", (data) => {
  buffered = Buffer.concat([buffered, data]);
  while (!failed) {
    const end = buffered.indexOf("\r\n\r\n");
    if (end < 0) return;
    const match = /Content-Length:\s*(\d+)/i.exec(buffered.subarray(0, end).toString());
    if (!match) return fail("Missing Content-Length in LSP response.");
    const length = Number(match[1]);
    if (buffered.length < end + 4 + length) return;
    let response;
    try {
      response = JSON.parse(buffered.subarray(end + 4, end + 4 + length).toString());
    } catch (error) {
      return fail(`Invalid LSP response: ${error.message}`);
    }
    buffered = buffered.subarray(end + 4 + length);
    if (response.error) return fail(JSON.stringify(response.error));
    if (stage === "initialize" && response.id === 1) {
      if (!response.result?.capabilities) return fail("Missing initialize capabilities.");
      stage = "shutdown";
      send({ method: "initialized", params: {} });
      send({ id: 2, method: "shutdown" });
    } else if (stage === "shutdown" && response.id === 2) {
      stage = "exit";
      send({ method: "exit" });
      child.stdin.end();
    }
  }
});
child.on("close", (code) => {
  clearTimeout(timer);
  if (failed) return;
  if (stage !== "exit" || code !== 0) return fail(`Server exited during ${stage} with code ${code}. ${stderr}`);
  console.log("LSP smoke test passed: initialize, shutdown response, exit (code 0).");
});

send({ id: 1, method: "initialize", params: { processId: process.pid, rootUri: null, capabilities: {} } });
