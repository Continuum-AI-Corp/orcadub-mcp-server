# Contributing

Thanks for your interest in improving the OrcaDub MCP server!

## Getting started

```bash
git clone https://github.com/Continuum-AI-Corp/orcadub-mcp-server
cd orcadub-mcp-server
make setup     # installs the pre-commit formatting hook (once)
make check     # fmt + lint + test + vet + build + npm syntax checks
```

You need Go (version from `go.mod`), Node ≥ 18 (launcher checks only), and
[golangci-lint](https://golangci-lint.run/usage/install/).

## Development guidelines

The architecture invariants and conventions live in [AGENTS.md](AGENTS.md) —
read it before changing code. The short version:

- All submit/query traffic goes through the OrcaRouter gateway with
  `model=orca/dub`; delivery uses the origin content route; presigned URLs
  never appear in tool output.
- The only runtime configuration is `ORCADUB_API_KEY`.
- Tests must stay hermetic (httptest fakes, no network).
- Everything is written in English: code comments, commits, PRs, docs.

## Pull requests

- Branch from `main` using `feat/<name>`, `fix/<name>`, `docs/<name>`, etc.
- Commit messages follow `<type>: <description>` (see AGENTS.md).
- Fill in the PR template: what changed, why, how it was verified, and any
  behaviour impact.
- Keep diffs small and single-purpose; update tests and docs in the same PR.
- CI (fmt, vet, lint, tests, npm checks, stdio smoke test, goreleaser check)
  must pass.

## Reporting bugs / requesting features

Use the issue templates. For security problems, follow
[SECURITY.md](SECURITY.md) instead of opening a public issue.
