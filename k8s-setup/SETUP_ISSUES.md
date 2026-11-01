# Deployment troubleshooting and historical issues

Reviewed 2026-10-31. Start with [configuration prerequisites](README.md#configure-before-deploying). The active path is Docker + k3d by default; historical nested-container experiments are summarized at the end.

## kubectl tries localhost:8080 in GitHub Actions

The recorded cert-manager job failed before installation because its missing KUBECONFIG secret decoded to an empty configuration. GitHub deployment is now manual/opt-in and validates the secret, current context, selected API endpoint and API access before installation. Provide a base64-encoded self-contained kubeconfig for a runner-reachable cluster. A local k3d localhost endpoint cannot reach the developer's machine from a GitHub-hosted runner. Do not log the config or disable TLS verification to work around it.

## Backend refuses to start

Inspect the failing pod and its app/init logs as appropriate. Startup requires a dedicated JWT_SECRET of at least 32 bytes; the base manifest reads the required gym-jinni-auth/JWT_SECRET Secret. Ansible/CI provision it automatically; direct manifest users must run the shared helper first. `APP_ENV=development` is an explicit local-only fallback, not a deployed fix. Old JWTs without session/purpose claims require login again.

## Migration or data-integrity failure

Review the `run-migrations` init-container logs and migration version. Migration 000013 rejects duplicate active bookings and invalid class time/capacity values; review existing records instead of automatically deleting/cancelling them. The init container now propagates errors: a failed or dirty migration prevents backend startup, while an already current schema succeeds without changes. Do not force migration versions or run destructive rollback against shared data as a routine check.

## Missing images or ImagePullBackOff

The local build roles call `sudo docker build` and import into k3d. The manifests use local tags and `imagePullPolicy: Never`; remote/GHCR deployment requires adjusted image references, pull policy and registry access. Keep DOCKER_HOST consistent across build/import/cluster operations. Check pod events and the selected engine's image store. Rebuilding a tag alone does not necessarily restart existing pods; the local build roles request a rollout restart if a Deployment exists.

## cert-manager installation or admission not ready

The shared installer waits for issuer CRDs, all three deployments and a server-side dry-run Issuer admission probe. A running webhook pod alone does not prove its serving certificate/CA is ready. Inspect `kubectl get deployments,pods,services -n cert-manager` and events when bounded waits fail. Verify API access and installation privileges first. GitLab deploy does not install these prerequisites automatically.

## Ingress or browser/API connectivity

The shared release renderer resolves the domain; explicit bootstrap configures the issuer; raw manifests contain placeholders that kubectl will not expand. Check DNS, ingress class, ACME email and reachability of host ports 80/443. Access local UI/API through ports 39300/39081; protected endpoints normally return 401 without a token. In a remote browser, localhost means the browser's machine. Use a reachable API_BASE or the nginx same-origin proxy. Active system administrators have a Management navigation destination; see [system administration](../docs/system-administration.md).

## Container engine, WSL and kubeconfig ownership

Check `docker info` or `sudo docker info`, the chosen DOCKER_HOST socket, and host port availability. Docker is the default. If deliberately using Podman's Docker-compatible API, enable its rootful socket and ensure the docker CLI/build/import commands use that engine. For WSL/world-writable workspaces, setup.sh exports ANSIBLE_CONFIG explicitly. Deployment plays run as the invoking user so they find that user's kubeconfig; avoid changing HOME to redirect tools.

## Native Node.js 24 action warnings

The workflow has migrated checkout/build-push/login/setup-buildx/setup-kubectl to verified native Node.js 24 major releases, and removed the forcing override. Existing completed workflow runs keep their old annotations; verify a run using the updated workflow commit. Workstation Node.js upgrades do not change GitHub actions' embedded runtimes. See [development tooling](../docs/development.md#optional-nodejs-tooling).

## Historical experiments

Generic Podman containers running MicroK8s/snap encountered AppArmor install-hook failures. Direct K3s experiments encountered overlay snapshotter, /dev/kmsg and cgroup permission failures. Archived roles/playbooks preserve those experiments, but they are not current setup instructions. k3d is the active lifecycle wrapper; host kernel/engine support still matters. Earlier unmeasured two-minute setup and blanket reliability claims are not validation evidence.

For test commands and dated outcomes, see the [validation report](../docs/business-process-validation.md). Changes to application behavior and remaining product/operations work belong in [spec.md](../spec.md).


## Signing Secret missing, invalid or mismatched

Use a valid kubeconfig for the intended cluster. From the repository root:

```bash
kubectl get namespace gym-jinni
bash k8s-setup/scripts/ensure-jwt-secret.sh
kubectl get secret gym-jinni-auth -n gym-jinni
```

The last command shows metadata, not key values. The helper reuses a valid existing key; a short/default key, missing JWT_SECRET data, denied access or conflicting explicit environment value returns an error. Do not delete a working Secret to make a conflict disappear: that is a signing-key rotation. Check the intended environment/secret-manager value and use the [implemented rotation command](README.md#signing-key-lifecycle) deliberately. It does not delete the Secret, uses a resource-version check, and can resume an incomplete rollout with the same key file.

## Migration 000013 blocks a rollout

```bash
kubectl get pods -n gym-jinni -l app=main-service
POD=replace-with-failing-pod
kubectl logs -n gym-jinni "$POD" -c run-migrations
kubectl describe pod -n gym-jinni "$POD"
```

Read the named database constraint/error first. Back up the target database; review duplicate confirmed user/class bookings or invalid class ranges/capacities with the data owner. Correct records through an approved recovery procedure, check the migration's dirty/version state, then retry. Do not reintroduce `|| true`, drop constraints, or force a version merely to make pods ready. A failing init is the expected protection, not a reason to start the application against a partial schema.

## Verify the documented configuration without a live deployment

From the repository root, first install `k8s-setup/tests/requirements.txt` in your Python test environment, then run `python3 -m unittest discover -s k8s-setup/tests -v`, and from `k8s-setup/` run `ansible-playbook --syntax-check -i inventory.yml playbooks/deploy-services.yml`. The test output is the authoritative current count. These checks do not replace rollout inspection or a backup/restore test against the actual deployment.


The earlier TCP wait was present; it has been strengthened to a bounded PostgreSQL availability check. For readiness configuration, dirty migration inspection, verified migration-13 recovery preconditions and a reproducible Docker harness, use [MIGRATION_RECOVERY.md](MIGRATION_RECOVERY.md). Recovery changes migration state only after data/schema preconditions are checked; it never runs automatically during deployment.

Ansible now exposes helper failures while inheriting optional operator key material from the controller environment rather than rendering it into the task's environment arguments. An API access error, conflicting key or invalid Secret should be diagnosed from the helper message; do not disable validation. Both provisioning and rotation reject `KUBE_NAMESPACE` values that disagree with the checked-in namespace.

Rendered releases use a migration Job rather than a per-replica migration init. Inspect `kubectl get jobs -n gym-jinni` and the failed Job logs before following [migration recovery](MIGRATION_RECOVERY.md). The checked-in base Deployment still has its compatibility init migration.
