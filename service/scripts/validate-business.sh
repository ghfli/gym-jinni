#!/usr/bin/env bash
# Isolated only: never uses the development database or existing containers.
set -euo pipefail
cd "$(dirname "$0")/.."
docker_cmd=(docker)
if ! docker info >/dev/null 2>&1; then docker_cmd=(sudo -n docker); fi
"${docker_cmd[@]}" info >/dev/null
run_id="gj-validation-$(date +%s)-$$"
artifacts=$(mktemp -d "${TMPDIR:-/tmp}/gj-validation.XXXXXX")
cleanup() {
  "${docker_cmd[@]}" logs "$run_id-service" > "$artifacts/service.log" 2>&1 || true
  "${docker_cmd[@]}" rm -fv "$run_id-service" "$run_id-db" >/dev/null 2>&1 || true
  "${docker_cmd[@]}" network rm "$run_id" >/dev/null 2>&1 || true
  "${docker_cmd[@]}" image rm "$run_id-image" >/dev/null 2>&1 || true
  echo "Validation artifacts (may contain sensitive service logs): $artifacts"
}
trap cleanup EXIT
"${docker_cmd[@]}" build -f ../k8s-setup/container/Dockerfile.service -t "$run_id-image" .. 2>&1 | tee "$artifacts/build.txt"
"${docker_cmd[@]}" network create "$run_id" >/dev/null
"${docker_cmd[@]}" run -d --name "$run_id-db" --network "$run_id" --network-alias postgres \
  -e POSTGRES_USER=validation -e POSTGRES_PASSWORD=validation -e POSTGRES_DB=validation \
  -p 127.0.0.1::5432 "${POSTGRES_TEST_IMAGE:-postgres:16-alpine}" >/dev/null
ready=false
for _ in {1..60}; do
  if "${docker_cmd[@]}" exec "$run_id-db" pg_isready -U validation -d validation >/dev/null 2>&1; then ready=true; break; fi
  sleep 1
done
$ready || { echo 'PostgreSQL readiness timed out' >&2; exit 1; }
for migration in mig/*.up.sql; do
  "${docker_cmd[@]}" exec -i "$run_id-db" psql -v ON_ERROR_STOP=1 -U validation -d validation < "$migration" > /dev/null
done
db_port=$("${docker_cmd[@]}" port "$run_id-db" 5432/tcp)
"${docker_cmd[@]}" run -d --name "$run_id-service" --network "$run_id" \
  -e DBURL='postgresql://validation:validation@postgres:5432/validation?sslmode=disable' \
  -e JWT_SECRET="$run_id-secret" -p 127.0.0.1::8081 \
  "$run_id-image" >/dev/null
api_port=$("${docker_cmd[@]}" port "$run_id-service" 8081/tcp)
ready=false
for _ in {1..60}; do
  if curl -fsS "http://$api_port/v1/roles/by-name/customer" >/dev/null 2>&1; then ready=true; break; fi
  sleep 1
done
$ready || { echo 'Service readiness timed out' >&2; exit 1; }
export VALIDATION_HTTP_BASE="http://$api_port"
export VALIDATION_DBURL="postgresql://validation:validation@$db_port/validation?sslmode=disable"
export VALIDATION_JWT_SECRET="$run_id-secret"
go test -count=1 -v ./integration 2>&1 | tee "$artifacts/results.txt"
