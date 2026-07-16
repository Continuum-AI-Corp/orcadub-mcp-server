# AGENTS.md — orcadub-mcp-server

Guidance for AI coding agents (Codex, Claude Code, Cursor, …) working in this
repository. Read this before changing code.

## What this repo is

The **public distribution repo** for the OrcaDub MCP server: a standalone Go
stdio MCP server that exposes the OrcaDub video-dubbing service (OrcaRouter
model `orca/dub`) as five tools — `dub_health` / `dub_upload` / `dub_create` /
`dub_get` / `dub_download`.

It is a **mirror for release purposes**. The development home is the private
`Continuum-AI-Corp/orca-rt-dubbing` repo (`cmd/dub-mcp`), which shares wire
types with the dub-server itself. Feature work generally lands there first and
is synced here manually. Keep the two in behavioural lockstep.

## Layout

```
cmd/                   # entrypoint; serverVersion stamped via goreleaser ldflags
internal/              # HTTP client (package dub), MCP tool layer, vendored wire types, tests
npm/                   # npx launcher: postinstall downloads the platform binary
.goreleaser.yaml       # 6-platform release (bare binaries + tar.gz/zip + ghcr.io docker images)
Dockerfile             # standalone build; Dockerfile.goreleaser is used at release time
.github/workflows/     # ci.yml (fmt/vet/build/test/goreleaser check), release.yml (tag → release + npm publish)
server.json            # MCP Registry manifest
```

## Architecture invariants — do not break these

1. **All submit/query traffic goes through the OrcaRouter gateway**
   (`https://api.orcarouter.ai`, hardcoded as `gatewayBaseURL`). Every
   entry-point request carries the `model=orca/dub` routing field: videos
   create body, `/v1/files` multipart form field, `/v1/uploads` create body.
2. **Delivery uses the origin content route** —
   `https://orcadub.orcarouter.ai/v1/videos/{job_id}/content` with the same
   Bearer key (`job_id` comes from the gateway's task retrieve). The gateway's
   own `/content` proxy is SSRF-blocked and its stored presigned URL expires;
   the origin re-signs per request. **Never expose presigned COS URLs**
   (`output_url`) to users — they are bearer-token-free and unrevocable.
3. **The only runtime configuration is `ORCADUB_API_KEY`.** No other env vars,
   no config files. A missing key must NOT fail startup: tools register and
   every call returns the sign-up redirect to
   `https://www.orcarouter.ai/console` so the agent can walk the user through
   registration.
4. **Tool surface = the gateway's documented dub lifecycle, nothing more.**
   Do not add list / cancel / delete / native `/api/v1/dub/*` tools — the
   gateway does not expose those routes.
5. **`dub_create` is billed** (per minute of source video). Required fields
   (`source_lang`, `target_lang`, `file_id` XOR `url`, `video_name` with
   `file_id`) carry "REQUIRED — ask the user" in their jsonschema
   descriptions — keep those annotations so agents ask instead of guessing.
   Boolean knobs are `*string` "true"/"false" on the wire (server convention);
   convert via `boolStr`.
6. **Folder names must not carry the `orcadub` brand** — directories are
   plain `cmd/`, `internal/`. The brand lives on the repo, the released
   binary, and the npm package (`orcadub-mcp`) only.
7. **Wire types in `internal/types.go` are vendored** from
   `orca-rt-dubbing/internal/quality/openaicompat` (the source of truth,
   shared with the server). Do not invent or rename JSON fields here — sync
   from the source when the server changes.
8. **Two error envelopes** exist on the wire and both must keep parsing:
   OpenAI-style `{"error":{"message":...}}` (gateway routing errors) and the
   OrcaRouter task envelope `{"code":..,"message":..,"data":..}` (dub task
   errors). See `apiError`.

## Development

```bash
make setup     # once after cloning: installs the .githooks formatting hooks
make check     # everything CI runs: fmt + lint + test + vet + build + node --check
```

Individual targets: `make fmt` (goimports+gofmt), `make lint`
(golangci-lint, config in .golangci.yml), `make test`, `make build`.

- Tests must stay hermetic — never call the real gateway from tests. The
  `Client` has `BaseURL`/`originURL` fields and `directUploadLimit` /
  `uploadPartSize` hooks specifically so httptest servers can stand in.
- Formatting is enforced in three layers: `.githooks/pre-commit`
  (goimports + gofmt + golangci-lint --fix + re-stage), `.githooks/pre-push`
  (blocks unformatted/lint-failing pushes), and CI (gofmt check +
  golangci-lint). Never commit unformatted Go code.
- Manual smoke test of the stdio surface:

  ```bash
  go build -o /tmp/dub-mcp ./cmd
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}' \
    | ORCADUB_API_KEY=sk-orca-... /tmp/dub-mcp
  ```

## Release process

1. Bump `version` in `server.json` if publishing to the MCP Registry.
2. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. GitHub Actions: goreleaser builds linux/darwin/windows × amd64/arm64
   (bare binaries feed the npm launcher; tar.gz/zip for humans), then the
   `npm-publish` job stamps the tag version into `npm/package.json` and
   publishes when the `NPM_TOKEN` secret is set.
4. `npm/package.json` stays at `0.0.0` in git — the workflow stamps it.
   `install.js` treats `0.0.0` as "dev install, skip download".
5. The repo must be public for external users: the npm postinstall downloads
   release assets anonymously.

## Git Workflow

### Branch Naming Convention

```
feat/<feature-name>         # New features (e.g., feat/http-transport)
fix/<bug-name>              # Bug fixes (e.g., fix/upload-part-retry)
refactor/<target>           # Code refactoring (e.g., refactor/error-envelopes)
test/<test-name>            # Add/update tests
docs/<doc-name>             # Documentation (e.g., docs/update-README)
chore/<task-name>           # Maintenance, CI/CD, dependencies
perf/<optimization>         # Performance improvements
```

### Commit Message Convention

**Format**: `<type>: <description>`

**Types**: `feat` · `fix` · `refactor` · `test` · `docs` · `chore` · `perf` · `style`

### Language Convention

- **Everything in this repository is written in English**: commit messages
  (subject AND body), PR titles and bodies, in-code comments, docs, and
  script output. This is a public distribution repo — no Chinese comments.

### PR Convention

- **Title**: same `<type>: <description>` convention, in English.
- **Body** (English), covering:
  - **What** changed and **why** — the problem and the approach, not a file list.
  - **How it was verified** — the commands you ran and their results:
    `go build ./...`, `go test ./... -count=1`, `gofmt -l .`, `go vet ./...`,
    and `node --check npm/*.js npm/bin/*.js` when the launcher changed.
  - **Behaviour impact** — especially anything touching the invariants above
    (auth, gateway routing, tool surface, billing-relevant paths).
  - Link related issues (`Closes #123`).
- Keep diffs **small and single-purpose**; update tests and docs (this file /
  README) in the same PR.
