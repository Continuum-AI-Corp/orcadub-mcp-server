<div align="center">

<img src="assets/orcadub.png" alt="OrcaDub" width="340">

### Translate any video. Keep the voice.

**The official [MCP](https://modelcontextprotocol.io) server for [OrcaDub](https://orcadub.orcarouter.ai) — AI video dubbing, driven from your agent.**

[![CI](https://github.com/Continuum-AI-Corp/orcadub-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/Continuum-AI-Corp/orcadub-mcp-server/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Continuum-AI-Corp/orcadub-mcp-server?include_prereleases)](https://github.com/Continuum-AI-Corp/orcadub-mcp-server/releases)
[![npm](https://img.shields.io/npm/v/%40orcadub%2Fcli)](https://www.npmjs.com/package/@orcadub/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Website](https://orcadub.orcarouter.ai) · [API Docs](https://orcadub.orcarouter.ai/api-docs) · [Get an API key](https://www.orcarouter.ai/console) · [Config examples](examples/) · [Releases](https://github.com/Continuum-AI-Corp/orcadub-mcp-server/releases)

</div>

---

Give any MCP-capable agent — Claude Code, Claude Desktop, Codex CLI, Cursor, Windsurf — the ability to dub a video into another language: upload a file or pass a URL, submit a job to the `orca/dub` model through the [OrcaRouter](https://www.orcarouter.ai) gateway, poll progress, and download the finished MP4.

## CLI usage

Every operation is a one-shot subcommand (no resident server needed):

```bash
export ORCADUB_API_KEY=sk-orca-...   # from https://www.orcarouter.ai/console

npx -y @orcadub/cli health
npx -y @orcadub/cli upload --path ./clip.mp4
npx -y @orcadub/cli create --source-lang en --target-lang ja \
  --url https://youtu.be/... --opt preserve_bgm=true
npx -y @orcadub/cli get --video-id <id>
npx -y @orcadub/cli download --video-id <id> --dest ./out.mp4
```

Optional `create` parameters use repeatable `--opt key=val` (e.g.
`--opt watermark=false --opt resolution=1080p --opt glossary.OrcaDub=虎鲸配音`).
Results print as JSON on stdout; errors go to stderr with a non-zero exit.

With no subcommand the same binary runs as an MCP stdio server (`npx -y @orcadub/cli`).

## Install the agent Skill

The [`dub-video` Skill](https://github.com/Continuum-AI-Corp/orcadub-plugin)
teaches an agent when to dub, which billed parameters it must confirm, and how
to run the upload → create → poll → download workflow. Install it interactively:

```bash
npx -y @orcadub/cli skill install
```

The guided installer opens with a compact 15-row `ORCADUB` wordmark, then lets you:

- Choose `简体中文` or `English`; the system locale selects the initial default.
- Choose Project or Global installation with the arrow keys.
- Select one or more platforms from a checkbox list. Detected platforms appear
  first and are already checked.
- Press `Space` to toggle, `/` to filter, `a` to select all, `n` to clear all,
  and `Enter` to confirm.

It supports all 33 platforms in its Comet-compatible catalog. Use
`--lang zh` or `--lang en` to skip the language screen. For unattended or
agent-driven setup, select targets explicitly:

```bash
# Current project
npx -y @orcadub/cli skill install \
  --platform claude --platform codex --scope project --lang en --yes

# User-wide Codex installation
npx -y @orcadub/cli skill install \
  --platform codex --scope global --lang zh --yes
```

Use `--yes` to accept detected platforms (or all platforms when none are
detected) without opening the installer. Use `--json` for decoration-free
structured output. Set `NO_COLOR` (or use `TERM=dumb`) for a color-free
installer. Existing identical content is left unchanged; an existing different
Skill is preserved unless `--force` is provided. Skill installation needs
network access to the canonical `orcadub-plugin` repository, but it does not
require `ORCADUB_API_KEY` and does not contact OrcaRouter.

## Quick start

```bash
claude mcp add orcadub -e ORCADUB_API_KEY=sk-orca-... -- npx -y @orcadub/cli
```

Then just ask your agent: *"Dub this into Chinese: https://www.youtube.com/watch?v=…"*

## Prerequisites

An **OrcaRouter API key** is required — every tool call is authenticated:

1. Register / log in at the [OrcaRouter console](https://www.orcarouter.ai/console).
2. Create an API key on the token management page (it looks like `sk-orca-...`).
3. Provide it via the `ORCADUB_API_KEY` environment variable (see below).

Dubbing jobs are billed per minute of source video. Without a key the server still starts, but every tool call returns a sign-up redirect to the console.

## Installation

### Claude Code

```bash
claude mcp add orcadub -e ORCADUB_API_KEY=sk-orca-... -- npx -y @orcadub/cli
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "orcadub": {
      "command": "npx",
      "args": ["-y", "@orcadub/cli"],
      "env": { "ORCADUB_API_KEY": "sk-orca-..." }
    }
  }
}
```

### Codex CLI

```bash
codex mcp add orcadub --env ORCADUB_API_KEY=sk-orca-... -- npx -y @orcadub/cli
```

### Cursor / Windsurf / other MCP hosts

Any host that launches stdio MCP servers works with the same shape: command `npx`, args `["-y", "@orcadub/cli"]`, env `ORCADUB_API_KEY`. Ready-to-copy configuration files for Claude Code, Codex, Cursor and Windsurf live in [`examples/`](examples/).

### Docker

Images are published to GHCR for linux/amd64 and linux/arm64:

```bash
docker pull ghcr.io/continuum-ai-corp/orcadub-mcp-server:latest
claude mcp add orcadub -e ORCADUB_API_KEY=sk-orca-... -- \
  docker run --rm -i -e ORCADUB_API_KEY -v "$PWD:$PWD" -w "$PWD" \
  ghcr.io/continuum-ai-corp/orcadub-mcp-server:latest
```

Or as host JSON config:

```json
{
  "mcpServers": {
    "orcadub": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-e", "ORCADUB_API_KEY",
               "-v", "/path/to/videos:/work", "-w", "/work",
               "ghcr.io/continuum-ai-corp/orcadub-mcp-server:latest"],
      "env": { "ORCADUB_API_KEY": "sk-orca-..." }
    }
  }
}
```

Note: `dub_upload` and `dub_download` read/write local paths, so mount the
directory you upload from / download to (the `-v` flag) — paths you pass to
the tools must be valid inside the container.

### Prebuilt binaries

Download the binary for your platform from the
[releases page](https://github.com/Continuum-AI-Corp/orcadub-mcp-server/releases)
(verify with `checksums.txt`), then register it directly:

```bash
# example: Linux amd64 via gh CLI
gh release download v0.1.0 -R Continuum-AI-Corp/orcadub-mcp-server \
  -p 'orcadub-mcp-server_*_linux_amd64' -O ~/bin/orcadub-mcp-server && chmod +x ~/bin/orcadub-mcp-server
claude mcp add orcadub -e ORCADUB_API_KEY=sk-orca-... -- ~/bin/orcadub-mcp-server
```

### From source

```bash
git clone https://github.com/Continuum-AI-Corp/orcadub-mcp-server && cd orcadub-mcp-server && make build
# binary lands in bin/orcadub-mcp-server
```

## Configuration

| Environment variable | Required | Description |
|---|---|---|
| `ORCADUB_API_KEY` | yes | OrcaRouter `sk-orca-...` key, sent as `Authorization: Bearer` on every request. The gateway resolves your workspace and billing from it. |

The gateway address (`https://api.orcarouter.ai`) is fixed in the binary — there is nothing else to configure.

## Tools

| Tool | Description |
|---|---|
| `dub_health` | End-to-end probe of gateway → `orca/dub` routing. No job is created, nothing is billed. |
| `dub_upload` | Upload a local video file; returns the `file_id` for `dub_create`. Files above 64 MiB are chunked automatically (up to 8 GiB). |
| `dub_create` | Submit a dubbing job (**billed**). Source is exactly one of `file_id` or `url` (YouTube and direct links are fetched server-side). Required: `source_lang`, `target_lang` (never `auto`), and `video_name` when using `file_id`. Optional knobs cover content profiles, glossaries, translation style, background-music preservation, lipsync, watermarking, resolution/ratio and more. |
| `dub_get` | Poll a job: `queued → in_progress → completed \| failed` with integer progress. On completion the response includes `content_url` (Bearer-authenticated delivery address, never expires) and `job_id`. |
| `dub_download` | Save a completed job's MP4 to a local path (refuses to overwrite existing files). |

Supported languages: `en zh ja ko fr de es pt ru ar it hi tr th vi id bn pl nl uk fil el cs sv da no fi sk`.

## Example

A typical agent conversation:

> **User:** Dub this into Chinese: https://www.youtube.com/watch?v=xxxx
>
> **Agent:** *(dub_create with source_lang=en, target_lang=zh, url=…)* → job queued → *(polls dub_get every ~30 s)* → completed → *(asks: download locally? default: current directory)* → *(dub_download)* → "Saved to ./My-Video-zh.mp4 — re-download any time: `curl -H "Authorization: Bearer sk-orca-..." <content_url> -o out.mp4`"

Direct tool usage from an agent prompt works too — for example `dub_create {"source_lang":"en","target_lang":"zh","url":"https://…","glossary":{"OrcaRT":"鲸鸣实时"}}`.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| Tool calls return `not authorized … www.orcarouter.ai/console` | `ORCADUB_API_KEY` is unset — register, export the key, restart the session. |
| `401` | Invalid or expired key — re-issue on the OrcaRouter console. |
| `402 insufficient_credit` | Top up your OrcaRouter balance (per-minute billing). |
| `429 free_quota_exceeded` | Free-tier limit reached. |
| `task_not_exist` on `dub_get` | The id wasn't created through this gateway — use the id exactly as returned by `dub_create`. |

## Development

```bash
go build ./...
go test ./... -count=1
```

Layout: `cmd/` (entrypoint) · `internal/` (HTTP client, Skill installer, tool layer, wire types) · `npm/` (npx launcher that downloads the platform binary from GitHub Releases) · `server.json` (MCP Registry manifest).

## Releasing

1. Tag: `git tag v0.1.0 && git push origin v0.1.0`.
2. GitHub Actions runs goreleaser: linux/darwin/windows × amd64/arm64 binaries + archives + checksums land on the Release.
3. The `npm-publish` job stamps the tag version into `npm/package.json` and publishes `@orcadub/cli` to npm (requires the `NPM_TOKEN` repo secret).
4. The registry job stamps the same tag version into `server.json` and publishes it to the [MCP Registry](https://registry.modelcontextprotocol.io).

## License

[MIT](LICENSE)
