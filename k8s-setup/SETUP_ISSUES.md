# Deployment troubleshooting and historical issues

Reviewed 2026-10-31. Start with [configuration prerequisites](README.md#configure-before-deploying). The active path is Docker + k3d by default; historical nested-container experiments are summarized at the end.

## kubectl tries localhost:8080 in GitHub Actions

The recorded cert-manager job failed before installation because its missing KUBECONFIG secret decoded to an empty configuration. GitHub deployment is now manual/opt-in and validates the secret, current context, selected API endpoint and API access before installation. Provide a base64-encoded self-contained kubeconfig for a runner-reachable cluster. A local k3d localhost endpoint cannot reach the developer's machine from a GitHub-hosted runner. Do not log the config or disable TLS verification to work around it.

## Backend refuses to start

Inspect `kubectl logs deployment/main-service -n gym-jinni`. Startup requires a dedicated JWT_SECRET of at least 32 bytes; the base manifest currently supplies only DBURL. Wire a secret reference as described in README before accepting the deployment. `APP_ENV=development` is an explicit local-only fallback, not a deployed fix. Old JWTs without session/purpose claims require login again.

## Migration or data-integrity failure

Review the `run-migrations` init-container logs and migration version. Migration 000013 rejects duplicate active bookings and invalid class time/capacity values; review existing records instead of automatically deleting/cancelling them. The current init-container command masks failures with `|| true`, so a started pod does not prove migrations succeeded. Do not force migration versions or run destructive rollback against shared data as a routine check.

## Missing images or ImagePullBackOff

The local build roles call `sudo docker build` and import into k3d. The manifests use local tags and `imagePullPolicy: Never`; remote/GHCR deployment requires adjusted image references, pull policy and registry access. Keep DOCKER_HOST consistent across build/import/cluster operations. Check pod events and the selected engine's image store. Rebuilding a tag alone does not necessarily restart existing pods; the local build roles request a rollout restart if a Deployment exists.

## cert-manager installation or admission not ready

The shared installer waits for issuer CRDs, all three deployments and a server-side dry-run Issuer admission probe. A running webhook pod alone does not prove its serving certificate/CA is ready. Inspect `kubectl get deployments,pods,services -n cert-manager` and events when bounded waits fail. Verify API access and installation privileges first. GitLab deploy does not install these prerequisites automatically.

## Ingress or browser/API connectivity

Ansible renders domain/issuer templates using group variables; raw manifests contain placeholders that kubectl will not expand. Check DNS, ingress class, ACME email and reachability of host ports 80/443. Access local UI/API through ports 39300/39081; protected endpoints normally return 401 without a token. In a remote browser, localhost means the browser's machine. Use a reachable API_BASE or the nginx same-origin proxy. No current admin management page is hidden behind a special URL; management APIs are documented in [RBAC notes](../service/RBAC.md).

## Container engine, WSL and kubeconfig ownership

Check `docker info` or `sudo docker info`, the chosen DOCKER_HOST socket, and host port availability. Docker is the default. If deliberately using Podman's Docker-compatible API, enable its rootful socket and ensure the docker CLI/build/import commands use that engine. For WSL/world-writable workspaces, setup.sh exports ANSIBLE_CONFIG explicitly. Deployment plays run as the invoking user so they find that user's kubeconfig; avoid changing HOME to redirect tools.

## Native Node.js 24 action warnings

The workflow has migrated checkout/build-push/login/setup-buildx/setup-kubectl to verified native Node.js 24 major releases, and removed the forcing override. Existing completed workflow runs keep their old annotations; verify a run using the updated workflow commit. Workstation Node.js upgrades do not change GitHub actions' embedded runtimes. See [development tooling](../docs/development.md#optional-nodejs-tooling).

## Historical experiments

Generic Podman containers running MicroK8s/snap encountered AppArmor install-hook failures. Direct K3s experiments encountered overlay snapshotter, /dev/kmsg and cgroup permission failures. Archived roles/playbooks preserve those experiments, but they are not current setup instructions. k3d is the active lifecycle wrapper; host kernel/engine support still matters. Earlier unmeasured two-minute setup and blanket reliability claims are not validation evidence.

For test commands and dated outcomes, see the [validation report](../docs/business-process-validation.md). Changes to application behavior and remaining product/operations work belong in [spec.md](../spec.md).
