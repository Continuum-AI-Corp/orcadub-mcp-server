# OrcaDub CLI mode — design

Date: 2026-07-22

## Problem

The `dub-video` skill (in the `orcadub-plugin` repo) currently drives a
long-running MCP server (`npx -y @orcadub/mcp`, launched at session start and
kept alive over stdio). This forces every consuming agent to:

- register and maintain an MCP server connection, and
- keep a resident process alive for the whole session.

We want the skill to stop depending on a resident MCP service. Instead, the
same underlying dubbing logic should be reachable as **one-shot CLI
subcommands** the skill invokes on demand. The process runs only while a
command executes, then exits — no server, no connection, no MCP config.

## Goal

1. Add a CLI mode to the existing Go binary that reuses `internal/client.go`
   verbatim (no changes to the HTTP/upload/download/error logic).
2. Rewrite the `dub-video` skill to call the CLI subcommands instead of MCP
   tools.
3. Rename the npm package `@orcadub/mcp` → `@orcadub/cli` (the binary is no
   longer primarily an MCP server) and update all references.

Both repos change: `orcadub-mcp-server` (the binary + npm packaging) and
`orcadub-plugin` (the skill + plugin metadata + docs).

## Non-goals

- No change to the OrcaRouter gateway integration, wire types, or the
  `internal.Client` behavior.
- No new capabilities beyond the existing 5 operations
  (health/upload/create/get/download).
- The binary keeps its MCP server mode — this is additive, not a replacement.

## Design

### Mode dispatch (`cmd/main.go`)

The binary distinguishes MCP mode from CLI mode by the first argument:

| Invocation | Behavior |
|---|---|
| `orcadub` (no args) | Run the MCP stdio server (unchanged — preserves `npx -y @orcadub/cli` MCP usage and backward compatibility) |
| `orcadub --version` / `-v` / `version` | Print version (unchanged) |
| `orcadub --help` / `-h` / `help` | Print help (updated to list CLI subcommands) |
| `orcadub health\|upload\|create\|get\|download …` | Run the CLI subcommand |

Rationale for "no args = MCP server": it preserves the existing MCP launch
path (agents that still want MCP just run the binary with no args) and keeps
the MCP stdio contract clean — MCP mode owns stdout for JSON-RPC only, while
CLI subcommands are always explicitly named, so the two never collide.

`main.go` gains one branch: when `os.Args[1]` is a known CLI subcommand,
delegate to `dub.RunCLI(os.Args[1:])` and exit with its return code.

### CLI layer (`internal/cli.go`, new)

`RunCLI(args []string) int` parses the subcommand and flags, builds a
`Client` from `LoadConfig()` (same as MCP mode — API key comes ONLY from the
`ORCADUB_API_KEY` environment variable), calls the matching `Client` method,
and prints the result.

Output contract:
- **Success** → the result object as JSON on **stdout** (same shape the MCP
  tools return via `jsonResult`), pretty-printed.
- **Failure** → error message on **stderr**, non-zero exit code (return 1
  from `RunCLI`). A missing API key surfaces the existing `errNotAuthorized`
  message (sign-up redirect) on stderr.

Each subcommand uses its own `flag.FlagSet` so `-h` per subcommand works and
unknown flags error clearly.

| Subcommand | Flags | Client method | Notes |
|---|---|---|---|
| `health` | (none) | `Health` | |
| `upload` | `--path` (required), `--purpose` (optional) | `UploadFile` | prints the FileObject (its `id` feeds create) |
| `create` | `--source-lang`, `--target-lang` (required); exactly one of `--url` / `--file-id`; `--video-name` (required with `--file-id`); `--opt key=val` (repeatable) | `CreateVideo` | same required-field validation as `dubCreate` today |
| `get` | `--video-id` (required) | `GetVideo` | prints GatewayVideo (status/progress/content_url) |
| `download` | `--video-id`, `--dest` (both required) | `DownloadContent` | refuses to overwrite (existing behavior); prints `{video_id, dest, bytes}` |

### `create` options: `--opt key=val` (repeatable)

`dub_create` has a large optional surface (a dozen tri-state booleans plus
glossary / profile / translation_style / resolution / ratio / project_id /
etc.). Rather than ~30 paired `--flag`/`--no-flag` entries, all optional
parameters are expressed through a single repeatable `--opt key=val`:

```
--opt preserve_bgm=true --opt watermark=false --opt resolution=1080p
--opt profile=podcast --opt glossary.OrcaDub=虎鲸配音
```

Semantics:
- `key` is the wire/JSON field name (`preserve_bgm`, `translation_style`, …) —
  the same names the skill already documents, so nothing new to memorize.
- **Tri-state booleans**: `key=true` / `key=false` map through the existing
  `boolStr` convention; a key that is not passed stays `nil` (= deploy
  default). This preserves the three states MCP had (default / true / false).
- **String fields** (`profile`, `translation_style`, `resolution`, `ratio`,
  `tts_backend`, `project_id`, `bed_reverb_preset`): set directly.
- **Map fields** (`glossary`, `speaker_assignments`): dotted key form
  `glossary.<term>=<rendering>`, accumulated into the map.
- Unknown keys → error (fail loud, don't silently drop a paid-job parameter).

`RunCLI`'s create handler translates the parsed `--opt` set into the same
`CreateInput`→`CreateVideoRequest` mapping `dubCreate` performs today, so the
gateway request body is identical between MCP and CLI paths. To avoid two
copies of that mapping drifting, the shared `CreateInput → CreateVideoRequest`
construction is factored into one helper (`buildCreateRequest`) that both
`dubCreate` (MCP) and the CLI create handler call.

### npm package rename

`@orcadub/mcp` → `@orcadub/cli`:
- `npm/package.json`: `name` field, description, keywords, repository refs.
- `npm/bin/run.js` / `npm/install.js`: any embedded package-name strings.
- `README.md`, `examples/*`: MCP config examples still show the binary run
  with no args (MCP mode) but under the new package name
  (`npx -y @orcadub/cli`); add a CLI-usage section.

The binary's own name (`orcadub-mcp-server`) and the goreleaser artifact names
are out of scope for this pass unless a rename is trivial — renaming release
artifacts risks breaking `install.js`'s download URLs. The npm package rename
is the user-facing change; the internal artifact name can follow later.

## `orcadub-plugin` changes

### `skills/dub-video/SKILL.md`

Rewrite tool references from MCP tools to CLI invocations. The workflow
(sanity → upload → create → poll → download), the "ASK, never guess" required
params, poll etiquette, and download confirmation all stay. Concretely:

- Prerequisites: replace "drives the orcadub MCP server" with "calls the
  `@orcadub/cli` CLI via `npx -y @orcadub/cli <subcommand>`; requires Node for
  `npx`, or a downloaded binary on PATH." API key still comes from
  `ORCADUB_API_KEY` in the environment.
- Each `dub_*` reference → the matching CLI command:
  - `dub_health` → `npx -y @orcadub/cli health`
  - `dub_upload {path}` → `npx -y @orcadub/cli upload --path <path>`
  - `dub_create {...}` → `npx -y @orcadub/cli create --source-lang … --target-lang … (--url … | --file-id … --video-name …) [--opt k=v …]`
  - `dub_get {video_id}` → `npx -y @orcadub/cli get --video-id <id>`
  - `dub_download {video_id, dest}` → `npx -y @orcadub/cli download --video-id <id> --dest <path>`
- Parse the JSON printed on stdout; treat a non-zero exit + stderr as the
  error path (replacing "the tool returned a not-authorized error").
- The optional toggles table maps site labels → `--opt <param>=true`.
- Debugging section: "tools missing → MCP not connected" becomes "command
  fails to run → Node/npx unavailable or the package couldn't be fetched";
  keep the gateway/402/429/task_not_exist guidance.

### `.claude-plugin/plugin.json`

- Remove the `mcpServers` block.
- Remove `userConfig.orcadub_api_key`. Decision: the API key is supplied ONLY
  via the `ORCADUB_API_KEY` environment variable. With no `mcpServers.env` to
  inject a userConfig value, the userConfig prompt would collect a key the CLI
  never reads; instead the skill instructs the user to export
  `ORCADUB_API_KEY` before dubbing.
- Update the plugin `description` to drop "Bundles the @orcadub/mcp server"
  (describe it as the dub-video skill that calls the OrcaDub CLI).

### `README.md` and `.claude-plugin/marketplace.json`

Replace MCP-centric wording with the CLI model: the skill calls
`@orcadub/cli` on demand, no resident server. Update the "What's inside" table
(5 subcommands instead of 5 MCP tools) and install steps.

## Testing

- `internal/cli_test.go`: table-driven tests for `RunCLI` covering
  - each subcommand's happy path against an `httptest` server (reuse the
    patterns in `client_test.go`),
  - `create` required-field validation (missing source/target lang; both/neither
    of url/file-id; file-id without video-name),
  - `--opt` parsing (bool tri-state, string, map dotted keys, unknown key
    error),
  - missing API key → `errNotAuthorized` on stderr, non-zero return,
  - stdout carries valid JSON of the expected shape on success.
- Assert the CLI create path and MCP create path build an identical
  `CreateVideoRequest` (guards the shared `buildCreateRequest` helper).
- Keep existing `tools_test.go` / `client_test.go` green (the refactor to
  `buildCreateRequest` must not change MCP behavior).
- `make check` (fmt/lint/test/vet/build + node --check on the npm scripts).

## Risks / trade-offs

- **Two copies of the create mapping** — mitigated by extracting
  `buildCreateRequest`.
- **`--opt` is less discoverable** than native flags — mitigated by keying on
  the already-documented wire names and by the skill spelling out the common
  options; unknown keys error loudly.
- **npm rename breaks existing MCP users** on the old name — accepted per the
  decision to rename outright (no dual-publish). Docs updated in lockstep.
- **API key via env only** — the skill must guide the user to export
  `ORCADUB_API_KEY`; no `--api-key` flag (avoids leaking the key into process
  listings / shell history).
