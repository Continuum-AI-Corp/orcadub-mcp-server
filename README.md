# OrcaDub MCP Server

An [MCP](https://modelcontextprotocol.io) server for [OrcaDub](https://orcadub.orcarouter.ai) — AI video dubbing. Give any MCP-capable agent (Claude Code, Claude Desktop, Codex CLI, Cursor, …) the ability to dub a video into another language: upload a file or pass a URL, submit a job to the `orca/dub` model through the [OrcaRouter](https://www.orcarouter.ai) gateway, poll progress, and download the finished MP4.

## Prerequisites

An **OrcaRouter API key** is required — every tool call is authenticated:

1. Register / log in at the [OrcaRouter console](https://www.orcarouter.ai/console).
2. Create an API key on the token management page (it looks like `sk-orca-...`).
3. Provide it via the `ORCADUB_API_KEY` environment variable (see below).

Dubbing jobs are billed per minute of source video. Without a key the server still starts, but every tool call returns a sign-up redirect to the console.

## Installation

### Claude Code

```bash
claude mcp add orcadub -e ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "orcadub": {
      "command": "npx",
      "args": ["-y", "orcadub-mcp"],
      "env": { "ORCADUB_API_KEY": "sk-orca-..." }
    }
  }
}
```

### Codex CLI

```bash
codex mcp add orcadub --env ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp
```

### Cursor / other MCP hosts

Any host that launches stdio MCP servers works with the same shape: command `npx`, args `["-y", "orcadub-mcp"]`, env `ORCADUB_API_KEY`.

### From source / prebuilt binaries

```bash
go install github.com/Continuum-AI-Corp/orcadub-mcp-server/cmd/dub-mcp@latest
# or grab a binary from https://github.com/Continuum-AI-Corp/orcadub-mcp-server/releases
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

Layout: `cmd/dub-mcp` (entrypoint) · `internal/dub` (HTTP client, tool layer, wire types) · `npm/` (npx launcher that downloads the platform binary from GitHub Releases) · `server.json` (MCP Registry manifest).

## Releasing

1. Tag: `git tag v0.2.0 && git push origin v0.2.0`.
2. GitHub Actions runs goreleaser: linux/darwin/windows × amd64/arm64 binaries + archives + checksums land on the Release.
3. The `npm-publish` job stamps the tag version into `npm/package.json` and publishes `orcadub-mcp` to npm (requires the `NPM_TOKEN` repo secret).
4. Optionally publish `server.json` to the [MCP Registry](https://registry.modelcontextprotocol.io) with `mcp-publisher publish`.

## License

[MIT](LICENSE)
