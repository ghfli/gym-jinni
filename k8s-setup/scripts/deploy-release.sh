#!/usr/bin/env bash
# Apply rendered artifacts; never update application replicas after a failed migration.
set -euo pipefail
release_dir=${1:?Usage: deploy-release.sh RENDERED_DIRECTORY}
namespace=gym-jinni
if [[ "${KUBE_NAMESPACE:-$namespace}" != "$namespace" ]]; then
  echo 'KUBE_NAMESPACE must be gym-jinni.' >&2
  exit 1
fi
script_dir=$(cd "$(dirname "$0")" && pwd)
# Namespace and database credentials are provisioned by the cluster operator.
kubectl get configmaps -n "$namespace" >/dev/null
bash "$script_dir/ensure-jwt-secret.sh"
kubectl apply -f "$release_dir/foundation.yml"
kubectl rollout status statefulset/postgres -n "$namespace" --timeout=300s
job_ref=$(kubectl apply -f "$release_dir/migration.yml" -o name)
[[ "$job_ref" == job.batch/gym-jinni-migrate-* && "$job_ref" != *$'\n'* ]] || {
  echo 'Expected exactly one release migration Job.' >&2; exit 1;
}
for ((attempt=0; attempt<125; attempt++)); do
  state=$(kubectl get "$job_ref" -n "$namespace" -o 'jsonpath={.status.conditions[?(@.type=="Complete")].status}{"/"}{.status.conditions[?(@.type=="Failed")].status}')
  if [[ "$state" == True/* ]]; then
    kubectl apply -f "$release_dir/application.yml"
    kubectl rollout status deployment/main-service -n "$namespace" --timeout=300s
    kubectl rollout status deployment/ui -n "$namespace" --timeout=300s
    exit 0
  fi
  if [[ "$state" == */True ]]; then
    echo "Migration $job_ref failed. Inspect its logs and follow MIGRATION_RECOVERY.md; application rollout was blocked." >&2
    exit 1
  fi
  sleep 5
done
echo "Timed out waiting for $job_ref; application rollout was blocked." >&2
exit 1
