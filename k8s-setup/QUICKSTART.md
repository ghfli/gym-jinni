# Cluster quick start

Reviewed 2026-10-31. Use a disposable local cluster and read [deployment prerequisites](README.md#configure-before-deploying) first. The base manifests need a signing-secret reference and environment-specific configuration; a setup script alone does not supply those values.

## Create the cluster

Install Ansible, kubectl and a working Docker daemon/CLI (or the explicitly configured compatible engine described in the full guide). Confirm `docker info` or `sudo docker info` works. From the repository root:

```bash
cd k8s-setup
./setup.sh --skip-build --skip-deploy
export KUBECONFIG="$HOME/.kube/gym-jinni-config"
kubectl get nodes
```

Defaults: one server/two agents; Docker API socket `/var/run/docker.sock`; Kubernetes API port 6443. The cluster role installs k3d if needed. Configure `DOCKER_HOST` consistently if using another engine. No fixed setup duration is guaranteed.

## Configure, build and deploy

Review database credentials, `JWT_SECRET` (at least 32 bytes), domain/ACME settings and migration integrity requirements in the full guide. Add the signing-secret reference to the backend Deployment before starting it; the current template supplies only `DBURL`. For local imported images:

```bash
ansible-playbook playbooks/deploy-services.yml --tags build
ansible-playbook playbooks/deploy-services.yml --tags deploy
ansible-playbook playbooks/deploy-services.yml --tags verify
kubectl get pods,svc,ingress -n gym-jinni
```

Local build roles use `sudo docker build` and import into k3d. The normal combined command is `./setup.sh` after configuration. `--skip-build` and `--skip-deploy` allow separate stages. Migration `000013` requires valid class data and unique confirmed user/class bookings. Confirm migrations succeeded; the base init-container currently masks errors.

## Access and administration

| Purpose | Local endpoint |
| --- | --- |
| Flutter UI | http://localhost:39300 |
| JSON API | http://localhost:39081/v1/ |
| gRPC | localhost:39080 |

```bash
curl --fail http://localhost:39081/v1/roles/by-name/customer
```

This lookup is public. Most APIs require a login access token and an appropriate permission/scope; `GET /v1/roles` is not an unauthenticated health check. Register in the UI, then use [first-admin provisioning](../service/RBAC.md) for administrator access. There are no administrator management pages yet; use the API inventory for available operations.

## Diagnose and clean up

```bash
kubectl logs deployment/main-service -n gym-jinni
kubectl describe deployment/main-service -n gym-jinni
kubectl get events -n gym-jinni --sort-by=.lastTimestamp
sudo -E k3d cluster list
```

Missing signing configuration causes backend startup refusal. Missing CI kubeconfig credentials fail before cert-manager installation. See [troubleshooting](SETUP_ISSUES.md) for both and for image/port issues. `./teardown.sh` deletes cluster/application resources after confirmation; back up needed data first.

For tests without a deployed cluster, use [business validation](../docs/business-process-validation.md) and the [development guide](../docs/development.md). Full deployment configuration, manual GitHub dispatch and GitLab differences are in [README.md](README.md). Historical migration rationale is in [K3D_IMPLEMENTATION.md](K3D_IMPLEMENTATION.md).
