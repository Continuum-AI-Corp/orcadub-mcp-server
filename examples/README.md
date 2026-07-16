# Configuration Examples

Pick the file matching your MCP client and copy it (or merge it) into the
client's config file. Then replace `sk-orca-REPLACE_ME` with your real
[OrcaRouter API key](https://www.orcarouter.ai/console) and restart the
client.

| Client | Example | Client config file | How to apply |
|---|---|---|---|
| Claude Desktop | [claude-desktop.json](claude-desktop.json) | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) / `%APPDATA%\Claude\claude_desktop_config.json` (Windows) | Replace contents (or merge under `mcpServers`) |
| Claude Code | [claude-code.json](claude-code.json) | `.mcp.json` in the project root, or `~/.claude.json` for all projects | Merge under the top-level `mcpServers` key |
| Codex CLI | [codex.toml](codex.toml) | `~/.codex/config.toml` | Merge the `[mcp_servers.orcadub]` tables |
| Cursor | [cursor.json](cursor.json) | `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (per project) | Replace contents (or merge under `mcpServers`) |
| Windsurf | [windsurf.json](windsurf.json) | `~/.codeium/windsurf/mcp_config.json` | Replace contents (or merge under `mcpServers`) |

For Zed and other MCP clients, consult your client's documentation for the
exact config schema and merge the `orcadub` entry under whatever key the
client uses (`mcpServers`, `context_servers`, etc.).

## One-command alternatives

Claude Code (user scope, all projects):

```bash
claude mcp add --scope user orcadub -e ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp-server
```

Codex CLI:

```bash
codex mcp add orcadub --env ORCADUB_API_KEY=sk-orca-... -- npx -y orcadub-mcp-server
```

## Docker variant

Every example above launches the server with `npx -y orcadub-mcp-server`. To run it
from the container image instead, swap the command in any of the files:

```json
{
  "command": "docker",
  "args": ["run", "--rm", "-i", "-e", "ORCADUB_API_KEY",
           "-v", "/path/to/videos:/work", "-w", "/work",
           "ghcr.io/continuum-ai-corp/orcadub-mcp-server:latest"],
  "env": { "ORCADUB_API_KEY": "sk-orca-REPLACE_ME" }
}
```

Mount the directory you upload from / download to (`-v`) — paths passed to
`dub_upload` / `dub_download` must be valid inside the container.

## Notes

- **Keep real keys out of version control.** If you commit `.mcp.json` to a
  project, Claude Code supports environment expansion — use
  `"ORCADUB_API_KEY": "${ORCADUB_API_KEY:-}"` and export the key in your
  shell instead of hardcoding it. Other clients store the literal key.
- Without a valid key the tools respond with a sign-up redirect to the
  OrcaRouter console instead of working — that is expected; set the key and
  restart the client.
- Codex CLI shows an approval prompt per MCP call in interactive sessions —
  choose "approve for session". Headless `codex exec` on codex-cli ≤ 0.142
  auto-cancels MCP approvals; use an interactive session there.
