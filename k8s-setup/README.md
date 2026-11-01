# Kubernetes setup and deployment

Reviewed 2026-10-31 against the scripts/manifests. Start with the [quick start](QUICKSTART.md), [development guide](../docs/development.md) and [specification](../spec.md). The manifests are local-development templates and need the configuration below before a successful deployment is expected.

## Architecture and prerequisites

The current path is Ansible + k3d (K3s in containers) using the Docker API. Docker is the default: `unix:///var/run/docker.sock`. `group_vars/all.yml` accepts `DOCKER_HOST`; the setup wrapper can use the rootful Podman API socket when no Docker socket exists. The installed `docker` CLI must reach the same engine used by k3d. Local build roles call `sudo docker build`, then `sudo -E k3d image import --mode direct`. Do not mix image stores between engines.

Install Docker or an explicitly configured compatible engine/CLI, Ansible, kubectl and Python 3 on the host. Confirm `docker info` (or `sudo docker info`) works. k3d is installed by the cluster role if absent; its version/default topology are in [defaults](roles/k3d/defaults/main.yml). WSL requires a running compatible daemon and an environment that supports the required networking/cgroups; the repository does not guarantee every nested-container configuration works.

By default the cluster has one server and two agents. PostgreSQL is a StatefulSet with persistent storage; one backend Deployment serves all ten domains including RBAC, and one UI Deployment serves Flutter via nginx. The main-service/UI manifests request two replicas; this does not establish an availability SLO. Traefik is supplied by K3s.

| Access | Host endpoint | In-cluster/NodePort |
| --- | --- | --- |
| UI | http://localhost:39300 | Service 80 / NodePort 30000 |
| HTTP gateway | http://localhost:39081/v1/ | 8081 / 30081 |
| gRPC | localhost:39080 | 8080 / 30080 |
| Kubernetes API | localhost:6443 by default | Configured kubeconfig endpoint |
| Ingress HTTP / HTTPS | Host ports 80 / 443 | Traefik; domain/TLS must be configured |
| PostgreSQL | Not published by default | Service 5432 |

Optional `kubectl port-forward` can use local ports 8080/8081 or 3000 without conflicting with the 39xxx load-balancer mappings. A browser on another computer needs the deployment hostname/IP rather than its own localhost. The five post-login UI tabs are customer-oriented; administrative management is currently API-only ([RBAC guide](../service/RBAC.md)).

## Configure before deploying

1. Set the target engine and domain. `group_vars/all.yml` reads `DOCKER_HOST` and `GYM_JINNI_DOMAIN`; it also defines ingress class and ACME email. Review these values for your environment. Ansible and remote CI use the shared release renderer; raw `manifests/ingress.yml` remains a compatibility template whose placeholder kubectl does not expand. Local rendering uses HTTP; remote TLS requires explicit cluster bootstrap.
2. Replace development database values in `manifests/secrets.yml` / `configmaps.yml` using your deployment secret/configuration process. The checked-in values are not production credentials.
3. The backend references the required `gym-jinni-auth` Secret, key `JWT_SECRET`. Ansible, GitHub and GitLab deployment paths run the shared provisioning helper before applying the Deployment. It generates a private random key only when the Secret is absent and reuses a valid existing key. An optional `JWT_SECRET` environment variable supplies an operator-managed key on first creation. Key bytes, including trailing newlines, are preserved and compared exactly. Keys must be at least 32 bytes, contain no NUL bytes and cannot be the development default; deployed environments do not use `APP_ENV=development`.
4. Review all numbered database migrations, including session-retirement migration `000014`. Resolve duplicate confirmed bookings or invalid class ranges/capacities before `000013`. A preceding, bounded `pg_isready` init container waits for PostgreSQL using its configured host/port/database/user. The migration init then runs `exec migrate ... up` with failure propagation: migration errors leave the pod in an init failure state and prevent the application from starting. An already current schema is a successful no-change run.
5. Match manifest image references/pull policy to the image distribution method. Local roles import `gym-jinni/service:latest` and `gym-jinni/ui:latest`; the manifests use `imagePullPolicy: Never`. GitHub publishes GHCR tags, so a remote deployment needs matching registry references, pull policy and private-registry access. Both CI providers render immutable published image references and wait for a release migration Job before applying application workloads; see [releases](RELEASES.md).

For direct manifest deployment, provision the namespace/signing Secret first (commands from the repository root):

```bash
kubectl apply -f k8s-setup/manifests/namespace.yml
bash k8s-setup/scripts/ensure-jwt-secret.sh
```

The helper targets the manifests' fixed `gym-jinni` namespace and rejects a conflicting `KUBE_NAMESPACE` before cluster access. It never prints key material or overwrites an existing key during normal deployment. Invalid/missing existing key data, access failures and a conflicting explicit `JWT_SECRET` fail closed. Concurrent initial deployments reuse the valid winning Secret. Key files used during creation are private and removed on exit. Do not commit a shared signing key to `manifests/secrets.yml`; that file contains only the development database Secret.

Remaining image/registry and raw-ingress rendering gaps are tracked in [OPS-05](../spec.md#12-deployment-operations-and-quality).

## Signing-key lifecycle

All replicas read the same required Secret through `secretKeyRef`. Ordinary redeployments must preserve it; deleting it causes the next deployment to generate a different key. If an operator provides an optional CI `JWT_SECRET`, it must match an existing key or the helper stops rather than silently rotating it.

For deliberate rotation, back up/protect the current key through your secret-management system and export the replacement into a private file (permissions 0600). From the repository root:

```bash
JWT_KEY_FILE=path-to-private-replacement-key
python3 k8s-setup/scripts/rotate-jwt-secret.py --key-file "$JWT_KEY_FILE"
```

The operator tool checks the namespace and target Deployment, compares exact key bytes, and updates only the signing-key field with a resource-version precondition. It preserves the Secret, unrelated data and metadata; a concurrent change fails instead of overwriting another update. It then restarts the backend and waits for rollout completion. If the key update succeeds but rollout fails, fix the underlying rollout problem and retry the **same** command/key; matching-key retries finish the rollout without replacing the Secret again.

When adding a CI secret after a cluster generated its own, either securely align CI to the existing key or explicitly rotate the cluster to the intended CI key, then align the CI value. Normal deployment deliberately refuses a mismatch and points to the rotation command. Do not delete a working Secret or silently adopt a new environment value. Do not read a key file using shell command substitution: that strips trailing newlines.

Plan a maintenance window for rotation. Existing tokens no longer validate after the replicas move to the new key; users must log in again, and mixed old/new replicas may reject requests during rollout. Seamless multi-key/grace-period rotation is a separate application change, not provided by this operator tool. Signing-key rotation is also distinct from refresh-token replay protection.

## Local setup and operations

From this directory:

```bash
./setup.sh --skip-build --skip-deploy
export KUBECONFIG="$HOME/.kube/gym-jinni-config"
kubectl get nodes
```

This provisions the cluster only. After reviewing/configuring the manifests and credentials above:

```bash
ansible-playbook playbooks/deploy-services.yml --tags build
ansible-playbook playbooks/deploy-services.yml --tags deploy
ansible-playbook playbooks/deploy-services.yml --tags verify
```

`./setup.sh` runs cluster creation, build and deploy together; `--skip-build` and `--skip-deploy` select stages. Timings depend on caches, network and host resources; there is no verified two-minute setup guarantee. Run Ansible from this directory so it uses [ansible.cfg](ansible.cfg). The wrapper explicitly sets `ANSIBLE_CONFIG` for WSL/world-writable working directories. The deployment plays use the invoking user's kubeconfig; their individual build operations invoke sudo as required.

```bash
kubectl get pods,svc,ingress -n gym-jinni
kubectl logs deployment/main-service -n gym-jinni
kubectl logs deployment/ui -n gym-jinni
kubectl get events -n gym-jinni --sort-by=.lastTimestamp
sudo -E k3d cluster list
sudo -E k3d cluster stop gym-jinni
sudo -E k3d cluster start gym-jinni
```

Keep the same `DOCKER_HOST` across setup/import/start/stop operations. For a deliberate Podman alternative on Linux, enable its rootful socket and set `DOCKER_HOST=unix:///run/podman/podman.sock`; also ensure the `docker` CLI used by the roles addresses that engine. These are environment-specific prerequisites, not a claim that Docker and Podman can share their image stores.

`./teardown.sh` prompts before deleting cluster/application resources and optionally images. Persistent database data may be lost; this is not a routine check. `kubectl scale deployment main-service --replicas=N -n gym-jinni` changes replica count but does not provide a database backup or an availability guarantee. Back up the intended PostgreSQL database and test restoration before destructive work.

## Database backup example

For the default development account, a logical backup can be taken without copying credentials into a command:

```bash
kubectl exec -n gym-jinni statefulset/postgres -- pg_dump -U root -d gj > gym-jinni-backup.sql
```

Protect the resulting file as application data. Adjust the account/database for your environment. Restore only into an explicitly prepared target and test the procedure before teardown; automated backup retention, disaster recovery and migration recovery remain operational work in the specification. Do not infer restore guarantees from a running StatefulSet or a successful image build.

## CI/CD

### GitHub Actions

[Workflow](../.github/workflows/deploy.yml) behavior:

GitHub builds/tests on configured push/PR branches and publishes images on non-PR runs. Deployment is an explicit main/master workflow dispatch with `deploy` enabled and a valid base64 `KUBECONFIG` secret. Both CI providers use the shared renderer and release gate. Configure the domain, immutable images, database and optional registry Secret as described in [releases](RELEASES.md). Cluster bootstrap is an explicit operator action, separate from routine CI privileges. Private temporary kubeconfigs are cleaned after deployment. GitHub concurrency and GitLab resource groups serialize each provider; select one deployment authority per cluster.

### GitLab CI

[Pipeline](../.gitlab-ci.yml) builds/pushes with Podman and offers a manual main/master deployment using base64 `KUBECONFIG_CONTENT`. Its integration job builds the service image (package checks), not the isolated HTTP suite. The deployment test job runs the shared structural suite with pinned dependencies. Deployment uses Python for rendering and checksum-verifies the pinned kubectl download, validates helper tools/kubeconfig, preserves the signing key and gates application rollout on its migration Job. Neither CI deploy path installs cluster-wide cert-manager prerequisites.

## cert-manager and deployment checks

Explicit operator bootstrap uses [install-cert-manager.sh](scripts/install-cert-manager.sh), pinned to the release in that file. It applies the manifest, waits for issuer CRDs and controller/CA-injector/webhook Deployment rollouts, then retries a server-side dry-run Issuer admission check. The dry run creates no issuer. Retries are bounded and failures print deployment/pod/service diagnostics before a real ClusterIssuer can be applied.

From the repository root, using an active Python test environment with pip (PyYAML is pinned in the requirements file):

```bash
python3 -m pip install -r k8s-setup/tests/requirements.txt
python3 -m unittest discover -s k8s-setup/tests -v
bash -n k8s-setup/scripts/*.sh
```

The command above reports the current regression count. Mocks cover credential/readiness failures, signing-secret lifecycle and migration command exit status; they do not install cert-manager in a real cluster. With Ansible installed, check playbook syntax from this directory using `ansible-playbook --syntax-check -i inventory.yml playbooks/deploy-services.yml`. See [validation evidence](../docs/business-process-validation.md), [troubleshooting](SETUP_ISSUES.md) and [implementation summary](IMPLEMENTATION_SUMMARY.md).


## Diagnose blocked startup

```bash
kubectl get pods -n gym-jinni -l app=main-service
kubectl get events -n gym-jinni --sort-by=.lastTimestamp
```

Choose the failing pod from the first command, then inspect its init/app logs without printing Secrets:

```bash
POD=replace-with-failing-pod
kubectl logs -n gym-jinni "$POD" -c run-migrations
kubectl describe pod -n gym-jinni "$POD"
kubectl logs -n gym-jinni "$POD" -c main-service
```

A missing required Secret prevents the application container from being created; rerun the helper against the intended namespace. A failed migration keeps the app blocked; follow the [migration recovery decision and reproduction guide](MIGRATION_RECOVERY.md) before retrying. For duplicate/class-constraint failures, back up the database and review the affected records rather than automatically cancelling bookings or forcing the migration version. See [troubleshooting runbooks](SETUP_ISSUES.md).

The committed real-database harness is `python3 k8s-setup/tests/verify_migrations.py`; its prerequisites, cold-start/upgrade/failure/recovery cases and resource cleanup are documented in the recovery guide. See the [focused follow-up plan](../docs/deployment-improvement-plan.md) for separate, bounded improvements beyond this review.

Rendered release deployment and CI prerequisites are documented in [releases](RELEASES.md). Remote CI uses immutable images and a migration Job before application rollout.
