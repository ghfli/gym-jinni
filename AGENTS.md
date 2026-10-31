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
