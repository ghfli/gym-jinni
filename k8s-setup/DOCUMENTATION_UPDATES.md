# Documentation index and maintenance notes

Refreshed 2026-10-31 from repository source. This file indexes current guides rather than claiming that generated templates or historical experiments have been deployed successfully.

| Document | Purpose |
| --- | --- |
| [Repository README](../README.md) | Product overview, entry points and build/test navigation |
| [AGENTS.md](../AGENTS.md) / [CLAUDE.md](../CLAUDE.md) | Contributor and coding-agent instructions |
| [Development](../docs/development.md) | Generation, credentials, validation and optional Node.js tooling |
| [Backend](../service/README.md) / [RBAC](../service/RBAC.md) | Service operation and first-administrator/API management |
| [UI](../ui/README.md) | Real screens, API_BASE, sessions and widget tests |
| [Deployment](README.md) / [quick start](QUICKSTART.md) | Source-grounded cluster/CI instructions and required customization |
| [Implementation](IMPLEMENTATION_SUMMARY.md) / [migration notes](K3D_IMPLEMENTATION.md) | Current tooling structure and historical rationale |
| [Database recovery](MIGRATION_RECOVERY.md) | Readiness, dirty migration decisions and reproducible isolated checks |
| [Troubleshooting](SETUP_ISSUES.md) | Concrete configuration, image, JWT, migration and TLS failures |
| [Specification](../spec.md) / [validation evidence](../docs/business-process-validation.md) | Requirement status, API inventory and dated results |
| [Historical VM notes](../memo.md) | Unimplemented proposals kept as historical reference |

## Changes in this refresh

Documented Go/Flutter/deployment tests, canonical code generation and schema constraint overlays; explicit JWT configuration and administrator bootstrap; current API-only management limitations; Docker-by-default cluster behavior; manual deployment and kubeconfig/registry/rendering prerequisites; cert-manager rollout/admission checks; native Node.js 24 actions; and portable optional Node.js tooling guidance.

The previous text referred to a nonexistent TEST_RESULTS.md. Verification evidence is now linked to the existing workflow report. Historical failure tables remain dated evidence rather than current pass/fail claims. Old unverified timing/SLO and production-ready assertions were removed from active guidance. AGENTS.md is the canonical contributor guide; CLAUDE.md is a short entry point that links it rather than importing a second copy.

## Maintenance checks

Check repository-relative Markdown links and heading anchors, documented paths/commands against source, and the distinction between implemented/partial/planned behavior. Keep docs-only changes separate from runtime changes. Do not rerun destructive setup/teardown or mutate a live cluster merely to check prose; a syntax/contract/link check is sufficient for this refresh. Re-run relevant runtime acceptance checks when actual code changes.


## Review follow-up

The deployment review is addressed in executable configuration: the backend has a required Secret reference, all deployment paths provision/reuse that key, and migration errors propagate out of the init container. The guides above were individually reviewed for working navigation and operational content. They retain distinct purposes: quick-start sequence, architecture/execution map, cluster-engine rationale, diagnostic/recovery runbooks, and this maintenance index. Current procedures were expanded where concrete commands were missing; obsolete unverified claims were not restored to increase line count.

Shared guides no longer contain a workstation's installation directories, resolved package versions or its TLS incident. Validation commands use `service/scripts/validate-business.sh` from the repository root. Active deployment guidance points to test output rather than a hardcoded test count.

A later review verified that a TCP wait already existed, strengthened it to PostgreSQL readiness, added explicit rotation/recovery tooling, fixed byte/namespace/logging issues, and added GitLab/structural-YAML checks. Agent essentials remain in AGENTS.md rather than duplicated tool guides. The [follow-up plan](../docs/deployment-improvement-plan.md) keeps further improvements separate from this correction.
