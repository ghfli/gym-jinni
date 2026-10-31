#!/usr/bin/env bash
# Create once, reuse on redeploy; never print or silently rotate signing keys.
set -euo pipefail
export LC_ALL=C
namespace="${KUBE_NAMESPACE:-gym-jinni}"
secret_name=gym-jinni-auth

valid_key() {
  [[ ${#1} -ge 32 && "$1" != gym-jinni-dev-secret-change-in-prod ]]
}
if [[ -n "${JWT_SECRET:-}" ]] && ! valid_key "$JWT_SECRET"; then
  echo 'JWT_SECRET must be a non-development signing key of at least 32 bytes.' >&2
  exit 1
fi

# Return 2 only when absent. Permissions, decoding and invalid key errors fail closed.
read_existing() {
  local record encoded
  record=$(kubectl get secret "$secret_name" -n "$namespace" --ignore-not-found \
    -o 'jsonpath={.metadata.name}{"\n"}{.data.JWT_SECRET}') || return 1
  [[ -n "$record" ]] || return 2
  if [[ "$record" != *$'\n'* ]]; then return 1; fi
  encoded=${record#*$'\n'}
  stored_key=$(printf '%s' "$encoded" | base64 --decode 2>/dev/null) || return 1
  valid_key "$stored_key" || return 1
  if [[ -n "${JWT_SECRET:-}" && "$JWT_SECRET" != "$stored_key" ]]; then
    echo 'JWT_SECRET differs from the existing signing key; use an explicit key-rotation procedure.' >&2
    return 1
  fi
}
if read_existing; then
  echo 'JWT signing secret already configured.'
  exit 0
else
  existing_status=$?
  if [[ "$existing_status" -ne 2 ]]; then
    echo 'Cannot read a valid existing JWT signing secret; check access and secret contents.' >&2
    exit 1
  fi
fi

umask 077
temp_dir=$(mktemp -d)
secret_file="$temp_dir/JWT_SECRET"
trap 'rm -f "$secret_file"; rmdir "$temp_dir"' EXIT
if [[ -n "${JWT_SECRET:-}" ]]; then
  printf '%s' "$JWT_SECRET" > "$secret_file"
else
  head -c 48 /dev/urandom | base64 | tr -d '\n' > "$secret_file"
fi
if kubectl create secret generic "$secret_name" -n "$namespace" \
    --from-file="JWT_SECRET=$secret_file" >/dev/null; then
  echo 'JWT signing secret created.'
  exit 0
fi
# Concurrent deploys may create the secret first. Accept a valid winning key,
# but still reject permissions/network errors or a conflicting explicit value.
if read_existing; then
  echo 'JWT signing secret already configured.'
  exit 0
fi
echo 'Failed to provision the JWT signing secret.' >&2
exit 1
