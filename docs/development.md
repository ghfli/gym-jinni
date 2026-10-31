# Development and tooling guide

Reviewed 2026-10-31 against repository source. Start commands from the repository root unless a block explicitly changes directory. [Specification](../spec.md), [API/workflow inventory](business-process-validation.md), [backend](../service/README.md), [UI](../ui/README.md), and [deployment](../k8s-setup/README.md) describe their respective behavior.

## Prerequisites and build

The backend is a Go module (`service/go.mod`, Go 1.21 language baseline); the canonical container builder currently uses Go 1.25.8 with pinned Buf/sqlc/protobuf plugins. Local builds need Go, Buf, sqlc and the configured protobuf plugins; database lifecycle targets also need Docker, PostgreSQL images and golang-migrate. Use the Dockerfile for the exact pinned tool versions. Flutter requires Dart `>=3.0.0 <4.0.0` per `ui/pubspec.yaml`; containers install Flutter stable, so consult their build output for the resolved SDK.

```bash
cd service
make bufgen dbsql
make buflint
go test ./...
go build -o cmd/service .
```

`make service` additionally runs `go mod tidy`. Generated `service/gen/` bindings are ignored and must come from the current contracts; handwritten or stale local bindings can conflict with source. A canonical container build avoids that state:

```bash
docker build -f k8s-setup/container/Dockerfile.service -t gym-jinni/service:check .
```

This runs protobuf lint, generation, sqlc and all Go package checks. Environment-dependent tests skip during package-only checks. `make dbsch` is optional: it regenerates committed schema SQL from DBML and appends the class/booking constraint overlays. Keep DBML, SQL, contracts and numbered up/down migrations synchronized.

## Database and signing configuration

For disposable local development, `cd service` then `make setup` creates the named `postgres` container, database `gj` and applies migrations. The Makefile defaults are development credentials; supply `make DBURL=... migup` for a different connection and review the target's Docker dependency before using a remote database. Run migrations only against the intended database.

Before launching `service/cmd/service`, set `DBURL` and a private `JWT_SECRET` of at least 32 bytes through your normal environment/secret mechanism. Explicit `APP_ENV=development` permits the development key. Migrations currently span `000001`–`000013`; `000013` refuses duplicate confirmed bookings and invalid class ranges/capacities instead of silently altering records. Older tokens without session/purpose claims require login again.

`make teardn` rolls back/drops local data and removes the database container. `make clean` removes generated bindings/binaries. Neither is part of routine validation.

## Validation

```bash
service/scripts/validate-business.sh
(cd ui && flutter test --no-pub)
(cd ui && flutter analyze --no-pub)
python3 -m pip install -r k8s-setup/tests/requirements.txt
python3 -m unittest discover -s k8s-setup/tests -v
```

Run `flutter pub get` first if dependencies are absent. Install deployment-test requirements in an active Python virtual/test environment with pip; GitHub creates an isolated environment and GitLab uses its Python test image. The business runner requires Docker access or passwordless `sudo -n docker`, Go and curl, plus network/cache access for build dependencies/images. It creates uniquely named containers, a private network and random loopback ports; applies migrations; tests bootstrap and HTTP workflows; round-trips migrations `000012`/`000013`; and removes only its own resources. Logs are retained in the printed temporary directory. Never point its validation variables at a shared database.

`make test` is a legacy smoke harness that creates local data and runs clients with privileged-operation assumptions. Its compilation is checked, but a fresh customer cannot perform all administrative steps. Prefer the assertion-based isolated runner. Historical and current evidence is in the [validation report](business-process-validation.md); this documentation revision does not claim a new live deployment or full runtime test run.

## Accounts and management

Register via the UI or API. A database operator can then run the first-admin command from `service/`:

```bash
go run ./cmd/bootstrap-admin -user-id 123
```

Set `DBURL` for that application database and replace `123` with the registered ID. The command refuses once an active super administrator exists; further role grants use authenticated RBAC APIs. See [RBAC instructions](../service/RBAC.md). The current UI provides customer journeys; administrative management is API-only.

## Optional Node.js tooling

Node.js is optional workstation tooling; backend and Flutter builds do not require a Node application/package installation. Choose a supported stable or LTS release from the [official downloads](https://nodejs.org/en/download), using your operating system's installation mechanism or a version manager. Installation paths and package-manager configuration belong to each workstation, not repository requirements.

Check the tools selected by your shell:

```bash
node --version
npm --version
npx --version
command -v node
```

If you use Corepack, check/install it according to its supported Node.js versions and your package-manager needs. Verify official release checksums when installing archives manually, preserve needed global tools, and test the new installation before switching your shell. Resolve certificate-trust problems through the environment's CA configuration; do not disable TLS verification.

GitHub actions run on their own native Node.js 24 runtimes, independent of the workstation version. The workflow is the source for action versions; a host Node.js upgrade does not migrate those actions.

## CI and deployment

GitHub Actions uses checkout v7, build-push v7, login v4, setup-buildx v4 and setup-kubectl v5, all verified as native Node.js 24 actions during migration. The runtime-forcing environment flag has been removed. Pushes/PRs on `main`, `master`, `csr` build images; non-PR runs publish. Deployment is opt-in through a manual run on `main`/`master` with `deploy` enabled and a valid base64 `KUBECONFIG` secret for a runner-reachable cluster. GitLab uses Podman and its own manual deploy job.

See [deployment guide](../k8s-setup/README.md) for prerequisites, local image versus GHCR differences, secret wiring, manifest rendering and cert-manager readiness.

For repeatable real PostgreSQL startup/upgrade/failure/recovery checks, see the [migration harness](../k8s-setup/MIGRATION_RECOVERY.md#reproduce-the-checks-safely). This harness is separate from the fake-command unit checks.
