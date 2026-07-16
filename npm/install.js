#!/usr/bin/env node
// postinstall: download the platform-matching orcadub-mcp binary from the
// GitHub release that corresponds to this package version. goreleaser
// uploads bare binaries named orcadub-mcp_{version}_{os}_{arch}[.exe], so
// no archive extraction is needed here — node built-ins only.
"use strict";

const fs = require("fs");
const path = require("path");
const https = require("https");

const pkg = require("./package.json");

const OS_MAP = { darwin: "darwin", linux: "linux", win32: "windows" };
const ARCH_MAP = { x64: "amd64", arm64: "arm64" };

function fail(msg) {
  console.error(`orcadub-mcp install: ${msg}`);
  process.exit(1);
}

const goos = OS_MAP[process.platform];
const goarch = ARCH_MAP[process.arch];
if (!goos || !goarch) {
  fail(`unsupported platform ${process.platform}/${process.arch}`);
}

const version = pkg.version;
if (version === "0.0.0") {
  // Local dev install of the unstamped package — nothing to download.
  console.log("orcadub-mcp install: dev version 0.0.0, skipping binary download");
  process.exit(0);
}

const ext = goos === "windows" ? ".exe" : "";
const asset = `orcadub-mcp_${version}_${goos}_${goarch}${ext}`;
const url = `https://github.com/Continuum-AI-Corp/orcadub-mcp/releases/download/v${version}/${asset}`;
const destDir = path.join(__dirname, "vendor");
const dest = path.join(destDir, `orcadub-mcp${ext}`);

function download(u, redirectsLeft, cb) {
  if (redirectsLeft <= 0) return cb(new Error("too many redirects"));
  https
    .get(u, { headers: { "User-Agent": "orcadub-mcp-npm-installer" } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        return download(res.headers.location, redirectsLeft - 1, cb);
      }
      if (res.statusCode !== 200) {
        res.resume();
        return cb(new Error(`HTTP ${res.statusCode} for ${u}`));
      }
      fs.mkdirSync(destDir, { recursive: true });
      const out = fs.createWriteStream(dest, { mode: 0o755 });
      res.pipe(out);
      out.on("finish", () => out.close(cb));
      out.on("error", cb);
    })
    .on("error", cb);
}

console.log(`orcadub-mcp install: downloading ${asset} ...`);
download(url, 5, (err) => {
  if (err) {
    fail(
      `${err.message}\n` +
        `  expected release asset: ${url}\n` +
        `  check https://github.com/Continuum-AI-Corp/orcadub-mcp/releases`
    );
  }
  console.log(`orcadub-mcp install: installed ${dest}`);
});
