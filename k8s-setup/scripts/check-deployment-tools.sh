#!/usr/bin/env bash
set -euo pipefail
for tool in bash base64 head tr mktemp cmp wc kubectl; do
  command -v "$tool" >/dev/null || { echo "Deployment image requires $tool" >&2; exit 1; }
done
if ! printf '%s' 'dG9vbC1jaGVjaw==' | base64 --decode >/dev/null 2>&1; then
  echo 'Deployment image requires base64 --decode support.' >&2
  exit 1
fi
