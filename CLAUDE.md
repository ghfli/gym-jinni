# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

See `AGENTS.md` for style, testing, and commit/PR conventions, and `spec.md` for per-module status and requirements.

## What this is

Gym-Jinni is a gym-management platform (think Vagaro / GymMaster / Strava). It uses a **modular monolith** backend in Go and a Flutter frontend, deployed to Kubernetes (k3d locally) via Podman-built container images.

Three top-level areas:
- `service/` — Go backend, a single module and single binary serving gRPC (`:8080`) + an HTTP/JSON gateway (`:8081`).
- `ui/` — Flutter app (web is the primary target; android/ios/desktop scaffolding also present).
- `k8s-setup/` — k3d + Ansible cluster automation and the Dockerfiles/manifests used by CI.

## Backend architecture (`service/`)

The backend is one Go module (`github.com/ghfli/gym-jinni/service`) built into one binary, but organized into **per-domain modules**: `user`, `class`, `rbac`, `booking`, `gym`, `schedule`, `payment`, `notification`, `activity`, `report`. Each module follows the same layout and toolchain — this is the single most important pattern to understand before adding features.

For a module `$MOD` (e.g. `user`):
- `$MOD/pb/` — Protobuf definitions + that module's `buf.yaml`. The actual `.proto` files live one level deeper (e.g. `user/pb/user/...`).
- `$MOD/db/` — `$MOD.dbml` (schema in DBML), `$MOD.sqlc.sql` (queries), and `sqlc.yaml` (codegen config).
- `$MOD/$MOD.go` — business logic. Implements the generated gRPC server interface as a struct named `Im<Service>ServiceServer` with a `New...()` constructor that opens the DB via the `DBURL` env var (see `user/user.go` for the canonical example).

**Generated code is NOT committed** and lives under `gen/` (`gen/go/...`, `gen/sql/...`). It must be regenerated before building. `service.go` imports from `gen/go/<mod>/v1alpha`, so a fresh checkout won't compile until you run codegen.

Two codegen pipelines:
- **buf** turns `.proto` into Go gRPC + grpc-gateway + go-proto-validators code. Workspace is `service/buf.work.yaml`; output config is `service/buf.gen.yaml`. Validators come from the `validator/pb` module imported by other protos.
- **sqlc** turns committed `schema/*.sql` + `$MOD/db/$MOD.sqlc.sql` into typed Go DB code. The schema files in `service/schema/` are the sqlc *source of truth* and are committed; they are regenerated from DBML via `make dbsch` (needs `dbml2sql`, optional).

`service.go` wires everything: `runGRPCServer()` registers each module's server and chains gRPC middleware (`grpc_validator` then `AuthInterceptor`); `runGatewayServer()` exposes the HTTP/JSON gateway. New modules must be registered in both functions. The gateway wraps all routes with `withCORS()` which allows `*` origin and the `Grpc-Metadata-user-id` header.

### Auth & RBAC

`service/auth/` provides JWT token generation and validation (`token.go`) and config (`config.go`). Access tokens expire in 15 min, refresh tokens in 7 days; roles are embedded in JWT claims. The secret is read from `JWT_SECRET` env var (default: `gym-jinni-dev-secret-change-in-prod`).

RBAC lives in `service/rbac/` (a normal module) plus `service/middleware/` (`auth.go`, `permissions.go`). `middleware/permissions.go` maps gRPC method paths → required permission names and maintains the public-method allowlist (LoginUser, CreateUser, RenewAccessToken). `AuthInterceptor(rbacClient)` is **already wired** in `service.go` as the second interceptor — it extracts `user_id` from the JWT via the `authorization` or `grpcgateway-authorization` metadata header and calls `rbacClient.CheckPermission()`. See `service/RBAC.md`. RBAC is part of the `service` module.

### Database & migrations

PostgreSQL, one database `gj` with a schema per domain. Migrations are golang-migrate files in `service/mig/` (`NNNNNN_name.up.sql` / `.down.sql`), applied in numeric order. The `SVCS` variable in the Makefile (currently all modules except `report`) drives which modules sqlc/dbml loop over — add new modules there. Note: the Dockerfile also only runs sqlc for the modules it has schema SQL for; if you add a module to `SVCS`, update the Dockerfile too.

## Common commands (run from `service/`)

```bash
# Full local build (regenerate code, tidy, compile the binary into cmd/service)
make service          # = bufgen + dbsql + go mod tidy + go build

# Codegen only
make bufgen           # protobuf -> gen/go (gRPC, gateway, validators)
make dbsql            # sqlc -> gen/go (per module in $SVCS)
make buflint          # lint protos
make bufmcc           # clear buf module cache (use when buf resolves stale deps)
make clean            # remove built binaries and gen/ directory

# Local Postgres lifecycle (Docker)
make setup            # create postgres:alpine container, create db gj, migrate up
make migup / migdn    # apply / roll back all migrations
make migup1 / migdn1  # one step
make migvn            # show current migration version
make migfr            # force-set migration version (use after manual fix)
make dbps             # psql into gj
make teardn           # migrate down + drop db + remove container

# Tests
go test ./...                              # unit tests (needs gen/ — run make bufgen first)
go test -run TestName ./middleware         # single test
scripts/validate-business.sh               # isolated integration suite (see below)
make test                                  # smoke test: start postgres + cmd/service, run cmd/grpcclt and cmd/httpclt.sh
```

Testing layers:
- **Unit tests** live beside the code (`auth/`, `class/`, `middleware/`, `cmd/bootstrap-admin/`). Some import `gen/go/...`, so codegen must run before `go test`.
- **Integration suite** `integration/business_test.go` skips unless `VALIDATION_HTTP_BASE` / `VALIDATION_DBURL` / `VALIDATION_JWT_SECRET` are set. Run it through `scripts/validate-business.sh`. The script builds the service image from `Dockerfile.service`, starts a throwaway Postgres + service on a private Docker network, applies `mig/*.up.sql`, runs the `bootstrap-admin` and `integration` tests, round-trips the latest migrations, and then removes everything. It never touches the dev DB. Requirements are traced in `docs/business-process-validation.md`.
- **Smoke clients** `cmd/grpcclt.go` and `cmd/httpclt.sh` drive a running server; extend them for API changes and cover authorization denials as well as successes.

`cmd/bootstrap-admin` promotes an existing user to the first administrator by writing to the DB directly (`DBURL=... go run ./cmd/bootstrap-admin -user-id N`). On purpose, there is no HTTP way to do this.

Docker targets in the Makefile use `sudo`. Requires on PATH: `buf`, `sqlc`, `migrate`, and protoc plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `protoc-gen-govalidators`). `DBURL` defaults to `postgresql://root:gj@127.0.0.1:5432/gj?sslmode=disable&search_path=public`. `JWT_SECRET` defaults to `gym-jinni-dev-secret-change-in-prod`.

Note: `buf dep update` is intentionally a no-op (`bufmup`) because per-module update fails when protos import sibling workspace modules. BSR deps stay pinned in `buf.lock`.

## Frontend (`ui/`)

Standard Flutter app; most logic is in `lib/main.dart`, backend URL config in `lib/api_config.dart`.

```bash
cd ui
flutter pub get
flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:39081   # against the k3d cluster
flutter test
flutter analyze
```

`API_BASE` is injected via `--dart-define`; the deployed image builds with an empty `API_BASE` (same-origin) because nginx proxies `/v1/` to the backend. The default for local dev is the k3d LB port `39081`.

## Deployment & ports

Container images are built with **Podman** from `k8s-setup/container/Dockerfile.service` and `Dockerfile.ui` (the service Dockerfile runs buf + sqlc inside the build). Both GitHub Actions (`.github/workflows/deploy.yml`) and GitLab CI (`.gitlab-ci.yml`) build/push images on `main`, `master`, and `csr`; k8s deploy is manual.

Local cluster: `k8s-setup/setup.sh` (Ansible + k3d via the rootful Podman socket). In-cluster the service listens on `8080`/`8081`; the **k3d load balancer remaps host ports to `39080` / `39081` / `39300`** to avoid clashing with local dev tools. Don't assume `8081` on the host — use `39081`.

## VM dev option

`vagrant up` provisions an Arch Linux VM via `Vagrantfile` + `setup-alx.sh`. To set the toolchain up directly on the host instead, run/read `setup-alx.sh`.

<!-- imported-from: codex:project:instructions -->
# Repository Guidelines

## Project Structure & Module Organization

- `service/`: Single Go module and binary (gRPC `:8080`, HTTP gateway `:8081`) organised into per-domain modules: `user`, `class`, `rbac`, `booking`, `gym`, `schedule`, `payment`, `notification`, `activity`, `report`. Auth helpers live in `auth/` (JWT) and `middleware/` (authorization interceptor, method-to-permission map).
- Each module keeps API definitions in `<module>/pb/`, DBML schema and SQL queries in `<module>/db/`, and business logic in `<module>/<module>.go`. Migrations are in `service/mig/`, committed sqlc schema in `service/schema/`, smoke clients in `service/cmd/`. Generated code goes to `gen/` and is not committed.
- `ui/`: Flutter application; Dart source in `lib/`, widget tests in `test/`, platform assets in the platform directories and `web/`.
- `k8s-setup/`: k3d/Ansible cluster automation, Dockerfiles (`container/`) and manifests. CI is in `.github/workflows/deploy.yml` and `.gitlab-ci.yml`.
- `Vagrantfile` and `setup-alx.sh` describe the VM development environment; `GymJinni_system_design.png` illustrates the target architecture; `spec.md` is the software specification.

## Build, Test, and Development Commands

Backend prerequisites include Go, Docker/PostgreSQL, Buf and its configured protobuf plugins, sqlc, golang-migrate, and optionally dbml2sql. Run targets inside `service/`:

- `make setup`: Create the local PostgreSQL container/database and apply migrations.
- `make service`: Generate protobuf/database code, tidy dependencies, and build `cmd/service`.
- `make bufgen dbsql`: Regenerate API and database bindings after schema/query changes.
- `make dbsch`: Regenerate `schema/*.sql` from DBML (needs `dbml2sql`).
- `make buflint`: Validate protobuf style.
- `make test`: Build binaries, start PostgreSQL and the service, run gRPC/HTTP smoke clients, then stop the service. Use disposable local data.
- Set `DBURL` before running `./cmd/service` directly; `JWT_SECRET` overrides the development signing secret.

From `ui/`, run `flutter pub get`, `flutter run -d chrome --dart-define=API_BASE=...`, and `flutter build web`. Respect the Dart SDK constraint in `pubspec.yaml`. Local cluster setup is `k8s-setup/setup.sh`; images are built with Podman.

## Coding Style & Naming Conventions

Format Go with `gofmt` (tabs) and Dart with `dart format lib test` (two-space indentation). Use lowercase Go package names, exported `PascalCase` identifiers, and Dart `snake_case.dart` filenames. Run `flutter analyze`; lint rules come from `flutter_lints`. Edit protobuf, DBML, and SQL sources rather than generated bindings. Add numbered migration pairs such as `000012_description.up.sql` and `.down.sql`. When adding a module, add it to `SVCS` in the Makefile, the Dockerfile's sqlc loop, and register it in both functions of `service.go`.

## Testing Guidelines

There are no Go unit tests yet. Use Go's standard framework with `*_test.go` files alongside implementation and run `go test ./...` in `service/` after generation. Flutter uses `flutter_test`; name tests `*_test.dart` and run `flutter test`. No coverage threshold is configured. Extend `cmd/grpcclt.go` and `cmd/httpclt.sh` for API changes and test authorization denials as well as successful requests.

## Commit & Pull Request Guidelines

History uses short, descriptive subjects such as `adjust setup-alx.sh`; no strict prefix convention is established. Keep commits focused. PRs should explain behavior changes, link relevant issues, list validation commands/results, and include screenshots for UI changes. Document migration, configuration, and API impacts.
