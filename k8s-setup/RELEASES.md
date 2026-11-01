# Rendered releases

Run commands from the repository root with Python and the pinned dependencies in `k8s-setup/tests/requirements.txt`. Rendering is offline and does not contact a cluster.

```sh
python3 k8s-setup/scripts/render-release.py --profile local --domain localhost --output /tmp/gym-jinni-release
```

Local output uses imported images and development database credentials. Remote rendering requires `--profile remote`, `--domain`, and `--service-image` / `--ui-image` tagged with the published commit SHA or pinned by digest. Add `--pull-secret NAME` for a private registry. Remote output contains no database Secret and references `postgres-secret/DBURL` instead of committing credentials. The DBURL must point to the provisioned database; POSTGRES_USER, POSTGRES_PASSWORD and POSTGRES_DB are also required by the bundled PostgreSQL workload. Keep POSTGRES_DB consistent with the ConfigMap's database name (`gj`).

The renderer emits foundation, migration and application artifacts. The migration Job uses the release service image and bounded database readiness; its name hashes the image and migration sources. Application replicas do not run migrations in rendered releases. The checked-in base Deployment retains its init migration for compatibility with direct local use.

Before remote deployment a cluster operator provisions the namespace, database Secret, optional registry pull Secret, storage, and cert-manager/`letsencrypt-prod` issuer. Routine CI does not install cluster-wide resources or overwrite database credentials. Configure GYM_JINNI_DOMAIN and optional REGISTRY_PULL_SECRET in CI; GitHub uses repository variables. Existing JWT_SECRET provisioning and explicit rotation rules still apply.

```sh
bash k8s-setup/scripts/deploy-release.sh /tmp/gym-jinni-release
```

This command targets the configured cluster: choose an operator-approved kubeconfig explicitly. It waits for PostgreSQL, applies one release Job, and waits for success before changing application workloads. A failed Job blocks rollout; inspect its logs and follow [migration recovery](MIGRATION_RECOVERY.md). After approved recovery, delete only that failed migration Job and rerun the same release to retry it. Successful release Jobs are retained for evidence. After migrations succeed, the same Job provisions the first super administrator with `bootstrap-admin -from-env` if the optional Secret `gym-jinni-admin` exists. Without that Secret the step is skipped. Create the Secret once from a `KEY=VALUE` file (`ADMIN_EMAIL`, `ADMIN_PHONE`, `ADMIN_PASSWORD` of at least 12 bytes, optional `ADMIN_NAME`) with `chmod 600 admin.conf && kubectl create secret generic gym-jinni-admin -n gym-jinni --from-file=config=admin.conf`. The step is idempotent: once any super_admin exists it changes nothing, and an existing account with the same email keeps its password. Delete the Secret after the first release if you don't want the credentials kept in the cluster. Existing application replicas remain running during migration, so migrations must remain compatible with the preceding release; rollback never automatically reverses schema changes.

Fast verification: `python3 -m unittest discover -s k8s-setup/tests -v`. Tests cover both profiles, invalid configuration, matching release/migration images, preserved signing-key references, and failure/success orchestration. These tests do not establish real cluster rollout or concurrent release safety; disposable cluster acceptance remains in the delivery ledger.

The operator can apply `manifests/deployer-rbac.yml` once to create the namespace-scoped deployer account and binding. It cannot administer namespaces, ClusterIssuers or cert-manager CRDs. Namespace-wide Secret access is required by the current create-once key provisioning; this account is trusted within the application namespace. API readiness uses the cluster's normal authenticated discovery permissions. Configure one CI provider as deployment authority per cluster; provider-local serialization does not coordinate independent pipelines across GitHub and GitLab.

Local Ansible also uses the renderer and release gate, keeping its imported image policy and cleaning temporary rendered credentials. Its default ingress is HTTP. For public TLS the cluster operator runs `ACME_EMAIL=operator@example.com bash k8s-setup/scripts/bootstrap-cluster.sh` with cluster-administrator credentials, then deploys the remote profile. This explicit bootstrap installs cert-manager, the issuer, namespace and deployer RBAC; normal releases cannot perform those cluster-wide operations.

Disposable rollout acceptance (requires Docker access, k3d, kubectl, Python dependencies and prebuilt current images):

```sh
python3 k8s-setup/tests/verify_rollout.py --service-image gym-jinni/service:mvp --ui-image gym-jinni/ui:latest
```

The harness allocates a unique cluster and private temporary kubeconfig without reading or changing the active context. It exercises startup/reapply, namespace RBAC, old-token rejection/new login after key rotation, and an actual failed Job blocking a proposed application change. Cleanup deletes only that generated cluster. These checks passed locally against the specified cached images; the harness tests backend deployment behavior rather than browser rendering. Re-rendering the local profile changes pod-template annotations so newly imported builds restart even when their local tags are unchanged. Applying the same artifact remains idempotent.
