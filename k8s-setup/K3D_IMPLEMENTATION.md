# k3d implementation and migration notes

Reviewed 2026-10-31. For commands and prerequisites use [README.md](README.md) / [quick start](QUICKSTART.md); this page explains structure and historical decisions.

## Current structure

`roles/k3d/` installs/configures k3d and exports the invoking user's kubeconfig to `~/.kube/gym-jinni-config`. `playbooks/setup-cluster-k3d.yml` runs that role without global privilege escalation. `setup.sh` checks the daemon, explicitly sets Ansible configuration, and selects cluster/build/deploy stages. `playbooks/deploy-services.yml` builds/imports local images and applies manifests; the deployment plays use the invoking user's kubeconfig.

Docker is the default engine; `group_vars/all.yml` and `DOCKER_HOST` configure the Docker API endpoint. A rootful Podman socket can be an explicit alternative with a compatible CLI, but is not the default. Match the engine used by image builds and imports. The setup wrapper's Podman fallback is conditional on a missing Docker socket and an available Podman socket.

Defaults publish host 39080/39081/39300 to NodePorts 30080/30081/30000, plus 80/443 to ingress. The default Kubernetes API host port is 6443. The backend serves RBAC alongside the other domains; there is no separate RBAC service process.

## Historical migration

Earlier experiments attempted MicroK8s/snap and direct K3s inside generic Podman containers. AppArmor, overlay/cgroup and device-access limitations blocked those experiments. The active implementation moved cluster lifecycle to k3d; old roles/playbooks are retained under [archive/](archive/) for reference. This history does not prove every host's networking/cgroup configuration works, nor is it a request to run archived setup commands.

## Current deployment changes

The service image now performs canonical protobuf/sqlc generation, lint and Go package checks. Local images use Docker/import; GitHub uses Buildx/native Node.js 24 actions and an explicit manual deploy input. Missing/invalid CI kubeconfig credentials fail before cert-manager installation. Ansible/GitHub share deployment rollout and webhook-admission readiness checks.

The templates require a provisioned signing Secret and propagate migration failures. Ansible/GitHub/GitLab now provision/reuse the key before backend apply. Deployment-specific database/domain values and image distribution alignment still need configuration. Review [configuration gaps](README.md#configure-before-deploying). Destructive teardown remains opt-in; no fixed performance/availability guarantees are established by this migration.

See [validation evidence](../docs/business-process-validation.md) and [troubleshooting](SETUP_ISSUES.md) for dated checks and failure cases.


## Engine and deployment tracing

From `k8s-setup/`, the default cluster path is `setup.sh` → `playbooks/setup-cluster-k3d.yml` → `roles/k3d/`. Build/import/deploy stages are in `playbooks/deploy-services.yml`. The shared key helper is invoked by the deployment role after namespace/database Secret application; its generated value is cluster state, not a rendered checked-in manifest.

```bash
sudo -E k3d cluster list
sudo -E k3d node list
kubectl get nodes
kubectl get pods -n gym-jinni
```

Use the same engine socket for these commands and for image imports. To diagnose image/startup failure, follow [SETUP_ISSUES.md](SETUP_ISSUES.md); to change configuration, update [group variables](group_vars/all.yml) and review the relevant manifests. The kubeconfig context, signing Secret and migration state are separate prerequisites and should be checked independently.
