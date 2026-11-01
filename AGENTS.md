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
- `make bufgen dbsql`: Regenerate API and database bindings after schema/query changes. `service/gen/` is ignored and must be regenerated before compiling; never commit or hand-edit it. `make bufmup` is intentionally a no-op because per-module `buf dep update` cannot resolve sibling workspace imports; keep committed `buf.lock` pins.
- `make dbsch`: Regenerate `schema/*.sql` from DBML plus `schema/constraints/*.sql` overlays (optional `dbml2sql`). Preserve the partial booking index and class checks.
- `make buflint`: Validate protobuf style.
- `make test`: Build binaries, start PostgreSQL and the service, run gRPC/HTTP smoke clients, then stop the service. Use disposable local data.
- Set `DBURL` and a dedicated `JWT_SECRET` of at least 32 bytes before running `./cmd/service`. Only explicit `APP_ENV=development` permits the development fallback. Never use that mode in a deployed environment.
- From the repository root, `service/scripts/validate-business.sh` runs assertion-based API/bootstrap checks and migration round trips against disposable PostgreSQL. It requires Docker access (or passwordless `sudo -n docker`), Go, curl, and image/build dependency access.
- After registration, a database operator can provision the first administrator from `service/` using `go run ./cmd/bootstrap-admin -user-id N` with the target `DBURL`. Further grants require authenticated role administration. For automated provisioning, `go run ./cmd/bootstrap-admin -from-env` creates (or reuses) the account from `ADMIN_EMAIL`, `ADMIN_PHONE`, `ADMIN_NAME` (optional) and `ADMIN_PASSWORD` (min 12 bytes), which may instead come from a `0600` `KEY=VALUE` file in `ADMIN_CONFIG_FILE` or a password file in `ADMIN_PASSWORD_FILE`; it is idempotent and a no-op once any administrator exists.

From `ui/`, run `flutter pub get`, `flutter run -d chrome --dart-define=API_BASE=...`, and `flutter build web`. Respect the Dart SDK constraint in `pubspec.yaml`. Local cluster setup is `k8s-setup/setup.sh`; Docker is the default engine, with an optional Docker-compatible Podman socket. Local Ansible builds use the `docker` CLI; GitHub uses Docker Buildx and GitLab uses Podman. See [development guide](docs/development.md) and [deployment guide](k8s-setup/README.md).

## Coding Style & Naming Conventions

Format Go with `gofmt` (tabs) and Dart with `dart format lib test` (two-space indentation). Use lowercase Go package names, exported `PascalCase` identifiers, and Dart `snake_case.dart` filenames. Run `flutter analyze`; lint rules come from `flutter_lints`. Edit protobuf, DBML, and SQL sources rather than generated bindings. Add numbered migration pairs after the current migration sequence, for example `000015_description.up.sql` and `.down.sql`. Review existing duplicates/invalid data before applying booking-integrity migration `000013`. When adding a module, add it to `SVCS` in the Makefile, the Dockerfile's sqlc loop, and register it in both functions of `service.go`.

## Testing Guidelines

Go unit tests cover authentication, class validation, and middleware; bootstrap/API tests require an isolated database and otherwise skip. Use Go's standard framework with `*_test.go` files alongside implementation and run `go test ./...` in `service/` after both protobuf and sqlc generation. The service Dockerfile runs canonical generation, protobuf lint and package checks, avoiding stale ignored host bindings. Flutter uses `flutter_test`; name tests `*_test.dart` and run `flutter test`. No coverage threshold is configured. Extend `cmd/grpcclt.go` and `cmd/httpclt.sh` for API changes and test authorization denials as well as successful requests.

## Commit & Pull Request Guidelines

History uses short, descriptive subjects such as `adjust setup-alx.sh`; no strict prefix convention is established. Keep commits focused. PRs should explain behavior changes, link relevant issues, list validation commands/results, and include screenshots for UI changes. Document migration, configuration, and API impacts.

## Deployment and Documentation

GitHub Actions builds on pushes/PRs to `main`, `master`, and `csr`; only non-PR runs publish images. Deployment requires a manual workflow run on `main`/`master` with the `deploy` input enabled and a base64 `KUBECONFIG` secret for a reachable cluster. Native Node.js 24 actions are used; no runtime-forcing flag is needed. GitLab deployment is also manual. Deployment guard tests: install pinned `k8s-setup/tests/requirements.txt` in a Python test environment, then run `python3 -m unittest discover -s k8s-setup/tests -v` from the root. Real startup/migration checks and conditional recovery are documented in [migration recovery](k8s-setup/MIGRATION_RECOVERY.md).

The Flutter UI has Register/Login and Home, Classes, Bookings, Activity, Profile tabs. Active super administrators also have a Management destination for the paged user directory, role grants/expiry and role/permission management. See [system administration](docs/system-administration.md); gym-scoped staff screens remain planned. Keep [spec.md](spec.md) and [validation report](docs/business-process-validation.md) aligned with implemented versus planned behavior; preserve dated historical results as historical evidence.

Node.js is optional host tooling, not a backend/Flutter runtime dependency. Keep shared instructions independent of individual machines and installation paths. See [optional tooling](docs/development.md#optional-nodejs-tooling); CI action runtimes are specified in the workflow.

Local k3d host ports are `39080` (gRPC), `39081` (HTTP API), and `39300` (UI); in-cluster ports are different. Keep these distinctions in client URLs and examples.
