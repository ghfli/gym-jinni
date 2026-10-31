#!/usr/bin/env bash
set -euo pipefail

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.2/cert-manager.yaml
kubectl wait --for=condition=Established \
  crd/issuers.cert-manager.io crd/clusterissuers.cert-manager.io \
  --timeout=120s
for deployment in cert-manager cert-manager-cainjector cert-manager-webhook; do
  kubectl rollout status "deployment/$deployment" -n cert-manager --timeout=300s
done

# A running webhook pod can still be waiting for its serving certificate/CA.
# Server dry-run verifies admission is usable without creating an issuer.
for attempt in {1..30}; do
  if kubectl apply --dry-run=server -f - >/dev/null 2>&1 <<'MANIFEST'
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: gym-jinni-readiness-check
  namespace: cert-manager
spec:
  selfSigned: {}
MANIFEST
  then
    echo 'cert-manager API and webhook are ready.'
    exit 0
  fi
  if [[ "$attempt" -lt 30 ]]; then sleep 2; fi
done
echo 'cert-manager webhook did not become ready after 30 admission checks.' >&2
kubectl get deployments,pods,services -n cert-manager >&2 || true
exit 1
