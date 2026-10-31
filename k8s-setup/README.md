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

1. Set the target engine and domain. `group_vars/all.yml` reads `DOCKER_HOST` and `GYM_JINNI_DOMAIN`; it also defines ingress class and ACME email. Review these values for your environment. Ansible renders ingress/issuer templates; raw `manifests/ingress.yml` contains a domain placeholder and kubectl does not expand it.
2. Replace development database values in `manifests/secrets.yml` / `configmaps.yml` using your deployment secret/configuration process. The checked-in values are not production credentials.
3. The backend references the required `gym-jinni-auth` Secret, key `JWT_SECRET`. Ansible, GitHub and GitLab deployment paths run the shared provisioning helper before applying the Deployment. It generates a private random key only when the Secret is absent and reuses a valid existing key. An optional `JWT_SECRET` environment variable supplies an operator-managed key on first creation. Keys must be at least 32 bytes and cannot be the development default; deployed environments do not use `APP_ENV=development`.
4. Review database migrations through `000013`. Resolve duplicate confirmed bookings or invalid class ranges/capacities before `000013`. The init container runs `exec migrate ... up` with failure propagation: migration errors leave the pod in an init failure state and prevent the application from starting. An already current schema is a successful no-change run.
5. Match manifest image references/pull policy to the image distribution method. Local roles import `gym-jinni/service:latest` and `gym-jinni/ui:latest`; the manifests use `imagePullPolicy: Never`. GitHub publishes GHCR tags, so a remote deployment needs matching registry references, pull policy and private-registry access. The current workflow does not substitute these references for you.

For direct manifest deployment, provision the namespace/signing Secret first (commands from the repository root):

```bash
kubectl apply -f k8s-setup/manifests/namespace.yml
bash k8s-setup/scripts/ensure-jwt-secret.sh
```

The helper never prints key material or overwrites an existing key. Invalid/missing existing key data, access failures and a conflicting explicit `JWT_SECRET` fail closed. Concurrent initial deployments reuse the valid winning Secret. Key files used during creation are private and removed on exit. Do not commit a shared signing key to `manifests/secrets.yml`; that file contains only the development database Secret.

Remaining image/registry and raw-ingress rendering gaps are tracked in [OPS-05](../spec.md#12-deployment-operations-and-quality).

## Signing-key lifecycle

All replicas read the same required Secret through `secretKeyRef`. Ordinary redeployments must preserve it; deleting it causes the next deployment to generate a different key. If an operator provides an optional CI `JWT_SECRET`, it must match an existing key or the helper stops rather than silently rotating it.

For deliberate rotation, back up/protect the current key through your secret-management system, update the Secret in the target cluster, then restart all backend replicas with `kubectl rollout restart deployment/main-service -n gym-jinni` and wait for the rollout. Environment variables in existing containers do not refresh automatically. Old tokens will no longer validate; users must log in again. The helper performs initial provisioning and validation, not automatic rotation/replay handling.

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

- Pushes/PRs to `main`, `master`, `csr` build service/UI images. The service build runs canonical generation, protobuf lint and Go package tests. The service matrix job also runs deployment-script regressions. PRs do not publish images; pushes/manual runs do.
- Actions use native Node.js 24: checkout v7, setup-buildx v4, login v4, build-push v7, setup-kubectl v5. There is no runtime-forcing environment flag. This runtime is independent of optional workstation Node.js tooling.
- Deployment is opt-in: select **Actions → Build and Deploy to K8s → Run workflow**, choose `main` or `master`, and enable **Deploy to the configured Kubernetes cluster after building**. Other manual runs build only.
- Add a base64-encoded `KUBECONFIG` Actions secret. It must be self-contained (CA/credentials embedded or available on the runner), have a current context, and select an API reachable from the runner. `localhost` in a developer's kubeconfig does not reach that developer's cluster from a GitHub-hosted runner. Configure an appropriate connected runner/network for private/local clusters.
- `scripts/configure-kubectl.sh` validates base64, context, selected server and API access before installation. It writes a private temporary file without printing credentials; the workflow removes it in an always-run cleanup step.
- The optional Actions `JWT_SECRET` is passed only to the provisioning helper; an absent value generates on first use or reuses the cluster Secret. Review the image and raw-manifest rendering requirements above. The job's existence is not evidence of a production-ready deployment.

### GitLab CI

[Pipeline](../.gitlab-ci.yml) builds/pushes with Podman and offers a manual main/master deployment using base64 `KUBECONFIG_CONTENT`. Its integration job builds the service image (package checks), not the isolated HTTP suite. The deploy job provisions/reuses the signing Secret before applying the backend; an optional protected/masked `JWT_SECRET` variable can supply the initial key. The GitLab deploy path does not install cert-manager or apply a ClusterIssuer; provision those prerequisites separately and review its raw manifests/image references. It does not share the GitHub kubeconfig helper automatically.

## cert-manager and deployment checks

Local Ansible and GitHub share [install-cert-manager.sh](scripts/install-cert-manager.sh), pinned to the release in that file. It applies the manifest, waits for issuer CRDs and controller/CA-injector/webhook Deployment rollouts, then retries a server-side dry-run Issuer admission check. The dry run creates no issuer. Retries are bounded and failures print deployment/pod/service diagnostics before a real ClusterIssuer can be applied.

From the repository root:

```bash
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

A missing required Secret prevents the application container from being created; rerun the helper against the intended namespace. A failed migration keeps the app blocked; investigate the error and database state before retrying. For duplicate/class-constraint failures, back up the database and review the affected records rather than automatically cancelling bookings or forcing the migration version. See [troubleshooting runbooks](SETUP_ISSUES.md).
