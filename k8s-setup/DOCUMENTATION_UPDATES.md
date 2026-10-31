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
| [Troubleshooting](SETUP_ISSUES.md) | Concrete configuration, image, JWT, migration and TLS failures |
| [Specification](../spec.md) / [validation evidence](../docs/business-process-validation.md) | Requirement status, API inventory and dated results |
| [Historical VM notes](../memo.md) | Unimplemented proposals kept as historical reference |

## Changes in this refresh

Documented Go/Flutter/deployment tests, canonical code generation and schema constraint overlays; explicit JWT configuration and administrator bootstrap; current API-only management limitations; Docker-by-default cluster behavior; manual deployment and kubeconfig/registry/rendering prerequisites; cert-manager rollout/admission checks; native Node.js 24 actions; and the observed optional Node.js 26 workstation installation.

The previous text referred to a nonexistent TEST_RESULTS.md. Verification evidence is now linked to the existing workflow report. Historical failure tables remain dated evidence rather than current pass/fail claims. Old unverified timing/SLO and production-ready assertions were removed from active guidance. Existing contributor additions in CLAUDE.md were retained and brought into alignment with AGENTS.md.

## Maintenance checks

Check repository-relative Markdown links and heading anchors, documented paths/commands against source, and the distinction between implemented/partial/planned behavior. Keep docs-only changes separate from runtime changes. Do not rerun destructive setup/teardown or mutate a live cluster merely to check prose; a syntax/contract/link check is sufficient for this refresh. Re-run relevant runtime acceptance checks when actual code changes.
