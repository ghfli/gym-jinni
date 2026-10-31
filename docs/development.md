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
python3 -m unittest discover -s k8s-setup/tests -v
```

Run `flutter pub get` first if dependencies are absent. The business runner requires Docker access or passwordless `sudo -n docker`, Go and curl, plus network/cache access for build dependencies/images. It creates uniquely named containers, a private network and random loopback ports; applies migrations; tests bootstrap and HTTP workflows; round-trips migrations `000012`/`000013`; and removes only its own resources. Logs are retained in the printed temporary directory. Never point its validation variables at a shared database.

`make test` is a legacy smoke harness that creates local data and runs clients with privileged-operation assumptions. Its compilation is checked, but a fresh customer cannot perform all administrative steps. Prefer the assertion-based isolated runner. Historical and current evidence is in the [validation report](business-process-validation.md); this documentation revision does not claim a new live deployment or full runtime test run.

## Accounts and management

Register via the UI or API. A database operator can then run the first-admin command from `service/`:

```bash
go run ./cmd/bootstrap-admin -user-id 123
```

Set `DBURL` for that application database and replace `123` with the registered ID. The command refuses once an active super administrator exists; further role grants use authenticated RBAC APIs. See [RBAC instructions](../service/RBAC.md). The current UI provides customer journeys; administrative management is API-only.

## Optional Node.js tooling

Node.js is optional workstation tooling; backend and Flutter builds do not require a Node application/package installation. GitHub actions run on their own native Node.js 24 runtimes, independent of the workstation version.

Observed workstation baseline on 2026-10-31:

| Tool | Installed version |
| --- | --- |
| Node.js | 26.13.0 LTS |
| npm / npx | 11.20.0 |
| Corepack | 0.36.0 |

The active installation is `/opt/node-v26.13.0-linux-x64`; `/usr/local/bin/node`, `npm`, `npx`, and `corepack` select it. `/opt/node-v20.10.0-linux-x64` remains as a rollback copy, not the active runtime. Other machines may use a version manager or their own installation paths. Verify the selected commands with:

```bash
node --version
npm --version
npx --version
corepack --version
command -v node
readlink -f /usr/local/bin/node
```

When upgrading, consult official release metadata, exclude prereleases, verify archive digests, test in temporary storage and preserve existing global tools before changing system links. The last upgrade verified the Node archive against official SHA-256 sums and Corepack against its official GitHub release digest. This host's npm registry TLS validation failed, so Corepack was installed offline from that verified release asset; certificate verification was kept enabled. Resolve the registry trust issue before relying on networked npm/package-manager operations.

## CI and deployment

GitHub Actions uses checkout v7, build-push v7, login v4, setup-buildx v4 and setup-kubectl v5, all verified as native Node.js 24 actions during migration. The runtime-forcing environment flag has been removed. Pushes/PRs on `main`, `master`, `csr` build images; non-PR runs publish. Deployment is opt-in through a manual run on `main`/`master` with `deploy` enabled and a valid base64 `KUBECONFIG` secret for a runner-reachable cluster. GitLab uses Podman and its own manual deploy job.

See [deployment guide](../k8s-setup/README.md) for prerequisites, local image versus GHCR differences, secret wiring, manifest rendering and cert-manager readiness.
