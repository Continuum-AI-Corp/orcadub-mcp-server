#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp "${TMPDIR:-/tmp}/orcadub-server-json.XXXXXX")"
trap 'rm -f "$fixture"' EXIT

cp "$repo_root/server.json" "$fixture"
"$repo_root/scripts/stamp-mcp-version.sh" v1.1.1 "$fixture"

jq -e '
  .version == "1.1.1"
  and (.packages | length > 0)
  and (.packages | all(.version == "1.1.1"))
' "$fixture" >/dev/null
