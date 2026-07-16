# CLAUDE.md

@AGENTS.md

The imported AGENTS.md is the canonical guidance for this repo (project
overview, architecture invariants, dev workflow, release process, git
conventions). Claude-specific notes below.

## Claude Code specifics

- **Before every `git commit`**: run `goimports -w` and `gofmt -w` on staged
  `.go` files, then re-stage them. Never commit unformatted Go code.
- **Testing the MCP server end-to-end from Claude Code**: register a local
  build and call the tools directly —

  ```bash
  go build -o ~/bin/dub-mcp ./cmd/dub-mcp
  claude mcp add --scope user orcadub -e ORCADUB_API_KEY=sk-orca-... -- ~/bin/dub-mcp
  ```

  then invoke `mcp__orcadub__dub_health` (free, no job created) to verify the
  gateway → orca/dub chain. `dub_create` bills real money — only submit real
  jobs when the user explicitly asks, and confirm the required parameters
  (source_lang / target_lang / source / video_name) with the user first.
- **Companion skill**: the Claude Code skill for the dubbing workflow lives in
  the private `orca-rt-dubbing` repo (`.claude/skills/dub/`) and is not part
  of this distribution repo by design.
- **Sync direction**: behaviour changes usually land in
  `orca-rt-dubbing/cmd/dub-mcp` first (wire types are shared with the server
  there), then get mirrored here. When editing here directly, flag in the PR
  body that the private repo needs the same change.
