#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${KUBECONFIG_BASE64:-}" ]]; then
  echo '::error::KUBECONFIG secret is missing. Add a base64-encoded kubeconfig for the deployment cluster to repository Actions secrets.' >&2
  exit 1
fi
if [[ -z "${KUBECONFIG:-}" || "$KUBECONFIG" == *:* ]]; then
  echo '::error::Set KUBECONFIG to a single output file.' >&2
  exit 1
fi
umask 077
mkdir -p "$(dirname "$KUBECONFIG")"
if ! printf '%s' "$KUBECONFIG_BASE64" | base64 --decode > "$KUBECONFIG" 2>/dev/null; then
  rm -f "$KUBECONFIG"
  echo '::error::KUBECONFIG secret is not valid base64.' >&2
  exit 1
fi
chmod 600 "$KUBECONFIG"
if [[ ! -s "$KUBECONFIG" ]] || ! kubectl config current-context >/dev/null 2>&1; then
  echo '::error::KUBECONFIG must contain a current context.' >&2
  exit 1
fi
if ! server=$(kubectl config view --minify -o 'jsonpath={.clusters[0].cluster.server}' 2>/dev/null) || [[ -z "$server" ]]; then
  echo '::error::KUBECONFIG current context must select a cluster with an API server.' >&2
  exit 1
fi
if ! kubectl --request-timeout=15s get --raw=/readyz >/dev/null 2>&1; then
  echo '::error::Cannot reach or authenticate to the configured Kubernetes API. The API must be reachable from the Actions runner; a local k3d/localhost endpoint requires a suitable self-hosted runner or network connection.' >&2
  exit 1
fi
