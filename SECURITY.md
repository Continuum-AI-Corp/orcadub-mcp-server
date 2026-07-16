# Security Policy

## Reporting a vulnerability

Please do **not** open a public issue for security problems.

Use GitHub's private vulnerability reporting on this repository
(**Security → Report a vulnerability**). We will acknowledge reports within
72 hours and keep you informed of the fix progress.

## Scope notes

- This server sends your `ORCADUB_API_KEY` as an `Authorization: Bearer`
  header to `api.orcarouter.ai` and `orcadub.orcarouter.ai` only. Any code
  path that could leak the key elsewhere (logs, tool output, other hosts) is
  in scope and treated as high severity.
- Presigned storage URLs must never appear in tool output — the client blanks
  them by design; a bypass is in scope.
- The npm launcher verifies downloaded binaries against the release
  `checksums.txt` (SHA-256). Integrity bypasses are in scope.
- Release artifacts carry GitHub Artifact Attestations (SLSA build
  provenance): `gh attestation verify <file> -R Continuum-AI-Corp/orcadub-mcp-server`.
  CI/release workflows pin every action to a commit SHA; provenance or
  pipeline-integrity bypasses are in scope.

## Supported versions

Only the latest release receives security fixes.
