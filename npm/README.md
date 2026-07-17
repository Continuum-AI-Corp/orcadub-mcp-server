<div align="center">
  <img src="https://raw.githubusercontent.com/Continuum-AI-Corp/orcadub-mcp-server/main/docs/assets/orcadub.png" alt="OrcaDub" width="300">
</div>

# orcadub-mcp-server

**Translate any video. Keep the voice.**

MCP server for [OrcaDub](https://orcadub.orcarouter.ai) — AI video dubbing via
[OrcaRouter](https://www.orcarouter.ai) (model `orca/dub`): upload a video or
pass a URL, submit a dubbing job, poll progress, download the finished MP4.

This package is a launcher: on install it downloads the platform-matching
prebuilt binary from the GitHub release of the same version and verifies its
SHA-256 against the release checksums.

## Usage

```bash
claude mcp add orcadub -e ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp-server
```

Get an API key at https://www.orcarouter.ai/console. Full documentation,
per-host configuration examples (Claude Code / Codex / Cursor / Windsurf) and
tool reference:

https://github.com/Continuum-AI-Corp/orcadub-mcp-server
