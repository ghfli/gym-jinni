#!/usr/bin/env bash
# Create once and compare opaque key bytes exactly; rotation is a separate command.
set -euo pipefail
export LC_ALL=C
namespace=gym-jinni
secret_name=gym-jinni-auth
if [[ "${KUBE_NAMESPACE:-$namespace}" != "$namespace" ]]; then
  echo 'KUBE_NAMESPACE must be gym-jinni to match the checked-in manifests.' >&2
  exit 1
fi

umask 077
temp_dir=$(mktemp -d)
secret_file="$temp_dir/candidate"
stored_file="$temp_dir/stored"
trap 'rm -f "$secret_file" "$stored_file"; rmdir "$temp_dir"' EXIT
valid_key_file() {
  local size non_nul_size
  size=$(wc -c < "$1")
  non_nul_size=$(tr -d '\000' < "$1" | wc -c)
  [[ "$size" -ge 32 && "$size" -eq "$non_nul_size" ]] || return 1
  ! cmp -s "$1" <(printf '%s' 'gym-jinni-dev-secret-change-in-prod')
}
if [[ -n "${JWT_SECRET:-}" ]]; then
  printf '%s' "$JWT_SECRET" > "$secret_file"
  if ! valid_key_file "$secret_file"; then
    echo 'JWT_SECRET must be a non-development signing key of at least 32 bytes without NUL bytes.' >&2
    exit 1
  fi
fi

# Return 2 only when absent. Decode into a file: $(...) would strip key newlines.
read_existing() {
  local record encoded
  record=$(kubectl get secret "$secret_name" -n "$namespace" --ignore-not-found \
    -o 'jsonpath={.metadata.name}{"\n"}{.data.JWT_SECRET}') || return 1
  [[ -n "$record" ]] || return 2
  [[ "$record" == *$'\n'* ]] || return 1
  encoded=${record#*$'\n'}
  printf '%s' "$encoded" | base64 --decode > "$stored_file" 2>/dev/null || return 1
  valid_key_file "$stored_file" || return 1
  if [[ -n "${JWT_SECRET:-}" ]] && ! cmp -s "$secret_file" "$stored_file"; then
    echo 'JWT_SECRET differs from the cluster key. Use rotate-jwt-secret.py --key-file FILE, then align the CI secret.' >&2
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
if [[ ! -s "$secret_file" ]]; then
  head -c 48 /dev/urandom | base64 | tr -d '\n' > "$secret_file"
fi
if kubectl create secret generic "$secret_name" -n "$namespace" \
    --from-file="JWT_SECRET=$secret_file" >/dev/null; then
  echo 'JWT signing secret created.'
  exit 0
fi
# Concurrent initial deployments may create the Secret first.
if read_existing; then
  echo 'JWT signing secret already configured.'
  exit 0
fi
echo 'Failed to provision the JWT signing secret.' >&2
exit 1
