#!/usr/bin/env bash

set -euo pipefail

tag="${1:?usage: stamp-mcp-version.sh vX.Y.Z [server.json]}"
manifest="${2:-server.json}"
version="${tag#v}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid release tag: $tag" >&2
  exit 2
fi

tmp="$(mktemp "${manifest}.tmp.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

jq --arg version "$version" '
  .version = $version
  | .packages |= map(.version = $version)
' "$manifest" >"$tmp"

mv "$tmp" "$manifest"
trap - EXIT
