# Configuration examples

Copy-paste MCP configurations for popular hosts. Every variant launches the
server with `npx -y orcadub-mcp` and needs exactly one setting: your
OrcaRouter API key (`sk-orca-...`, created at
https://www.orcarouter.ai/console → token management page).

| Host | Example file | Where it goes |
|---|---|---|
| Claude Code | [`claude-code/.mcp.json`](claude-code/.mcp.json) | Project root (`.mcp.json`) — commit it; the key is read from the `ORCADUB_API_KEY` environment variable, never from the file |
| Codex CLI | [`codex/config.toml`](codex/config.toml) | Merge into `~/.codex/config.toml` |
| Cursor | [`cursor/mcp.json`](cursor/mcp.json) | `.cursor/mcp.json` (per project) or `~/.cursor/mcp.json` (global) |
| Windsurf | [`windsurf/mcp_config.json`](windsurf/mcp_config.json) | `~/.codeium/windsurf/mcp_config.json` |

## One-command alternatives

Claude Code (user scope, all projects):

```bash
claude mcp add --scope user orcadub -e ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp
```

Codex CLI:

```bash
codex mcp add orcadub --env ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp
```

## Notes

- **Claude Code** expands `${ORCADUB_API_KEY:-}` from the host environment, so
  the example `.mcp.json` is safe to commit. For Cursor / Windsurf / Codex the
  config stores the literal key — replace `sk-orca-REPLACE_ME` and keep those
  files out of version control.
- Without a valid key the tools respond with a sign-up redirect to the
  OrcaRouter console instead of working — that is expected; set the key and
  restart the session.
- Codex CLI shows an approval prompt per MCP call in interactive sessions —
  choose "approve for session". Headless `codex exec` on codex-cli ≤ 0.142
  auto-cancels MCP approvals; use an interactive session there.
