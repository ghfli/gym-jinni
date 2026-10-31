#!/usr/bin/env bash
# Explicit operator action, separate from routine namespaced application deployment.
set -euo pipefail
: "${ACME_EMAIL:?Set ACME_EMAIL for the public TLS issuer}"
script_dir=$(cd "$(dirname "$0")" && pwd)
issuer=$(python3 - <<'PY'
import json, os, re
email = os.environ['ACME_EMAIL']
if not re.fullmatch(r'[^\s@]+@[^\s@]+\.[^\s@]+', email):
    raise SystemExit('ACME_EMAIL must be a valid email address')
print(json.dumps({'apiVersion':'cert-manager.io/v1','kind':'ClusterIssuer',
 'metadata':{'name':'letsencrypt-prod'},'spec':{'acme':{
 'server':'https://acme-v02.api.letsencrypt.org/directory','email':email,
 'privateKeySecretRef':{'name':'letsencrypt-prod-account-key'},
 'solvers':[{'http01':{'ingress':{'ingressClassName':'traefik'}}}]}}}))
PY
)
bash "$script_dir/install-cert-manager.sh"
kubectl apply -f "$script_dir/../manifests/namespace.yml"
kubectl apply -f "$script_dir/../manifests/deployer-rbac.yml"
printf '%s' "$issuer" | kubectl apply -f -
