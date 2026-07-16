#!/usr/bin/env node
// Launcher shim: exec the vendored platform binary with inherited stdio so
// MCP stdio framing passes straight through.
"use strict";

const path = require("path");
const fs = require("fs");
const { spawn } = require("child_process");

const ext = process.platform === "win32" ? ".exe" : "";
const bin = path.join(__dirname, "..", "vendor", `orcadub-mcp${ext}`);

if (!fs.existsSync(bin)) {
  console.error(
    "orcadub-mcp: binary not found — postinstall may have failed.\n" +
      "Reinstall with: npm rebuild orcadub-mcp (or npx -y orcadub-mcp@latest)"
  );
  process.exit(1);
}

const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  process.exit(code === null ? 1 : code);
});
child.on("error", (err) => {
  console.error(`orcadub-mcp: ${err.message}`);
  process.exit(1);
});
