#!/usr/bin/env node
// postinstall: download the platform-matching orcadub-mcp binary from the
// GitHub release that corresponds to this package version, verify its
// SHA-256 against the release's checksums.txt, then install it. goreleaser
// uploads bare binaries named orcadub-mcp_{version}_{os}_{arch}[.exe], so no
// archive extraction is needed — node built-ins only.
"use strict";

const fs = require("fs");
const path = require("path");
const https = require("https");
const crypto = require("crypto");

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
const base = `https://github.com/Continuum-AI-Corp/orcadub-mcp-server/releases/download/v${version}`;
const destDir = path.join(__dirname, "vendor");
const dest = path.join(destDir, `orcadub-mcp${ext}`);

function fetch(u, redirectsLeft, cb) {
  if (redirectsLeft <= 0) return cb(new Error("too many redirects"));
  https
    .get(u, { headers: { "User-Agent": "orcadub-mcp-npm-installer" } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        return fetch(res.headers.location, redirectsLeft - 1, cb);
      }
      if (res.statusCode !== 200) {
        res.resume();
        return cb(new Error(`HTTP ${res.statusCode} for ${u}`));
      }
      const chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => cb(null, Buffer.concat(chunks)));
      res.on("error", cb);
    })
    .on("error", cb);
}

console.log(`orcadub-mcp install: downloading ${asset} ...`);
fetch(`${base}/checksums.txt`, 5, (err, sums) => {
  if (err) fail(`cannot fetch checksums.txt: ${err.message}`);
  const line = sums
    .toString("utf8")
    .split("\n")
    .find((l) => l.trim().endsWith(asset));
  if (!line) fail(`no checksum entry for ${asset} in checksums.txt`);
  const expected = line.trim().split(/\s+/)[0];

  fetch(`${base}/${asset}`, 5, (err2, bin) => {
    if (err2) {
      fail(
        `${err2.message}\n` +
          `  expected release asset: ${base}/${asset}\n` +
          `  check https://github.com/Continuum-AI-Corp/orcadub-mcp-server/releases`
      );
    }
    const actual = crypto.createHash("sha256").update(bin).digest("hex");
    if (actual !== expected) {
      fail(`SHA-256 mismatch for ${asset}: expected ${expected}, got ${actual}`);
    }
    fs.mkdirSync(destDir, { recursive: true });
    fs.writeFileSync(dest, bin, { mode: 0o755 });
    console.log(`orcadub-mcp install: verified sha256 and installed ${dest}`);
  });
});
