# Focused deployment improvement plan

This plan follows the lifecycle review. The delivered fixes cover PostgreSQL readiness/recovery, explicit byte-exact signing-key rotation, namespace/logging guards, shared CI tests and structural YAML parsing. The changes below are **planned**, not claims of completed deployment support. Implement and verify each group before starting the next; keep behavior changes and their focused tests in separate commits.

## 1. Render deployable image and domain configuration

Replace raw local-image/domain assumptions in remote CI with one rendering path shared by GitHub/GitLab. Keep local k3d imported tags and Never pull policy as the local profile. The remote profile must use the image SHA built by that run, registry pull configuration and an explicit domain/ACME configuration. Fail before apply if required values/placeholders remain unresolved.

Acceptance: render both profiles without touching a cluster; assert image references/policies, domain values and absence of unresolved placeholders; validate Kubernetes resources structurally; retain existing namespace/Secret/startup contracts. This closes the remaining image/domain portion of OPS-05 without changing application APIs.

## 2. Coordinate release migrations independently of replica startup

Introduce a release-scoped migration operation so the rollout has one explicit migration result before app replicas update. Preserve fail-closed startup/rollback decisions and PostgreSQL locking; do not hide dirty state or automatically repair customer data. Keep older deployments compatible until the release orchestration is verified.

Acceptance: disposable PostgreSQL tests for concurrent release attempts, no-change, rejected migration data and retry/recovery; assert the application rollout does not begin after migration failure. Reuse the committed harness and document the release recovery boundary.

## 3. Separate cluster bootstrap from normal deployment privileges

Keep cluster-wide cert-manager installation/issuer administration separate from routine application updates. Define the namespace-scoped permissions needed for Secret read/create/patch, Deployment rollout and manifest application. Retain clear configuration/tool checks and private kubeconfig handling.

Acceptance: review the rendered RBAC resources; verify allowed/denied operations in a disposable cluster, including failure visibility and no credential contents in logs. Do not replace the operator's repository/cluster credentials or upload kubeconfigs during implementation.

## 4. Add disposable rollout acceptance and plan key rollover policy

Add an opt-in test that creates a uniquely named k3d cluster using a temporary kubeconfig, builds/imports local images and exercises real startup, reapply, failed migration gating and explicit key rotation. Keep it separate from the fast mock suite and never target the active Kubernetes context.

Acceptance: old tokens fail after explicit single-key rotation, new logins work after rollout, Secret data survives reapply, startup is blocked after migration errors, and cleanup removes only test resources. Use this evidence to decide a later multi-key/grace-period design if uninterrupted token validity during rotation is required; do not imply the current single-key operator tool provides it.

## Verification and delivery rule

Each change should report its exact test commands, actual results and limitations. Run the fast suite in both CI providers; use the real PostgreSQL harness for schema/readiness changes and disposable cluster checks for rollout/RBAC changes. Production rollout, customer-data repair, irreversible migrations and new credential publication remain distinct actions, not side effects of reviewing or testing these improvements.
