# Deployment implementation summary

Reviewed 2026-10-31 against current source. This summarizes implemented tooling; [README.md](README.md) is the operational guide and [spec.md](../spec.md) tracks feature/operations gaps. Earlier unmeasured timing and production-readiness claims are not acceptance criteria.

## Current implementation

| Area | Implemented behavior |
| --- | --- |
| Cluster | k3d/K3s, one server/two agents by default; Docker API/daemon by default |
| Backend | One modular Go binary for ten domains, gRPC 8080/HTTP 8081 |
| UI | Flutter web through nginx, same-origin `/v1/` proxy |
| Database | PostgreSQL StatefulSet, ordered migrations through 000013 |
| Local builds | Docker CLI, k3d image import, local image tags with Never pull policy |
| GitHub | Buildx matrix, native Node.js 24 actions, package checks and Python deployment regressions; manual opt-in deployment |
| GitLab | Podman image builds; manual deployment with separate kubeconfig variable |
| TLS setup | Shared cert-manager installer for Ansible/GitHub, rollout waits and bounded dry-run admission probe |
| Verification | Disposable API/bootstrap/migration runner, Go/Flutter tests, deployment guard tests |

The host load balancer maps 39080→30080 (gRPC), 39081→30081 (HTTP), 39300→30000 (UI), and 80/443 for ingress. The API defaults to host 6443. RBAC is part of the backend, not a separate deployment. See [role defaults](roles/k3d/defaults/main.yml).

## Prerequisites still owned by the deployer

Provide dedicated signing/database secrets, domain/ACME settings, a runner-reachable kubeconfig, image references/pull policy matching the registry/import method, and confirmed migration success. The backend now requires the shared signing Secret, provisioned once/reused by deployment helpers, and migration errors block startup; GitHub applies raw manifest placeholders and does not rewrite local image tags. GitLab does not install the cert-manager prerequisites. These are documented gaps, not delivered automation.

The first administrator is bootstrapped by a trusted database operator; current management screens are API-only. Multi-gym scope, provider settlement/delivery, recurrence/resource conflicts and management UI remain in the product backlog.

## Evidence and maintenance

[Validation report](../docs/business-process-validation.md) preserves the initial failures and successful remediation results. Deployment-script mocks exercise configuration/readiness failure paths; they are not a live cluster installation result. Build success, two replicas and readiness probes do not establish an SLO or prove failover/restore behavior.

Keep module registration, Makefile/sqlc loops and Dockerfile generation aligned. Preserve [constraint overlays](../service/schema/constraints/) during DBML refreshes. Test backups/restores before destructive operations, keep container-engine selection consistent, and verify actual resolved tool versions when upgrading. [Development guide](../docs/development.md) describes optional host tooling independently of CI runtimes, without requiring any machine-specific installation.

Historical MicroK8s/K3s-in-Podman experiments and migration reasons are in [setup issues](SETUP_ISSUES.md) and [migration notes](K3D_IMPLEMENTATION.md). No historical draft is an active setup playbook.


## Deployment execution map

| Stage | Source | Failure behavior |
| --- | --- | --- |
| Namespace/database configuration | `roles/gym-jinni/tasks/main.yml`, CI manifest steps | Apply errors stop deployment |
| Persistent signing key | `scripts/ensure-jwt-secret.sh` | Invalid/mismatched keys or access errors stop; ordinary redeploys preserve the key |
| Database readiness | `wait-for-postgres` init container with `pg_isready` | Bounded retries wait for server availability before migrations |
| Schema migration | `run-migrations` in `manifests/main-service-deployment.yml` | Nonzero migration exit blocks application startup; no-change succeeds |
| Backend environment | Required `gym-jinni-auth/JWT_SECRET` reference | Missing Secret/key prevents container creation |
| TLS readiness | `scripts/install-cert-manager.sh` | Rollout/admission failures stop before a real issuer is applied |

The deployment test command reports its current count. Regression checks cover secret preservation, explicit-key conflicts, creation races, denied access and execution of the actual init command. Dated validation evidence distinguishes mocked behavior from isolated real PostgreSQL migration checks. Use [diagnostic commands](README.md#diagnose-blocked-startup) for blocked pods rather than bypassing init failure.

The [committed migration harness and conditional recovery runbook](MIGRATION_RECOVERY.md) replace one-off verification claims. Manifest tests use structural YAML selection by resource/container name. Both CI providers run the pinned-dependency regression suite; the signing key has a separate, byte-exact operator rotation path with rollout completion checks.
