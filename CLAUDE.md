# Claude Code entry point

Read [AGENTS.md](AGENTS.md) for the canonical repository rules, layout, change conventions and verification requirements. Do not maintain a second copy of those instructions here.

Use these task-specific sources:

- [Development guide](docs/development.md) for prerequisites and build/test commands. Its validation commands run from the repository root; other blocks explicitly state their working directory.
- [Backend guide](service/README.md) and [RBAC guide](service/RBAC.md) for service contracts, authentication and administrator provisioning.
- [UI guide](ui/README.md) for supported screens and client tests.
- [Deployment guide](k8s-setup/README.md) for cluster/CI operation, signing-secret lifecycle and migration failure handling.
- [Specification](spec.md) and [validation report](docs/business-process-validation.md) for implemented versus planned behavior and dated evidence.

For code changes, use the matching source contracts and relevant checks from these guides. For documentation changes, preserve dated evidence and verify links, commands and implementation claims against source.
