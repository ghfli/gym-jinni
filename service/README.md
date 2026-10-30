# Gym-Jinni backend (`service`)

Single Go module and binary: **gRPC :8080**, **HTTP gateway :8081**.

| Area | Package / paths |
|------|-----------------|
| Users | `user/`, `gen/go/user/v1alpha` |
| Classes and bookings | `class/`, `booking/` — registered on both transports |
| RBAC | `rbac/`, `middleware/`, [RBAC.md](RBAC.md) |

## Build (local)

Requires [buf](https://buf.build), protoc plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `protoc-gen-govalidators`), and [sqlc](https://sqlc.dev).

```bash
make bufmup bufgen
make dbsql
go build -o cmd/service .
```

Schema for sqlc is committed under `schema/*.sql`. Optional: `make dbsch` refreshes from DBML if `dbml2sql` is installed.

## Migrations

```bash
export DBURL="postgresql://..."
make migup
```

Includes all domain schemas and RBAC seeds through migration `000013`. Before upgrading, review duplicate confirmed bookings and invalid class times/capacities; `000013` refuses invalid existing records. [RBAC.md](RBAC.md) describes first-administrator provisioning and permission grants.

## Docker

Image build runs protobuf lint, buf + sqlc generation, and `go test ./...` inside the container (see `k8s-setup/container/Dockerfile.service`).


## Run and verify

Set `DBURL` and a dedicated `JWT_SECRET` of at least 32 bytes through your normal environment/secret mechanism before running `./cmd/service`. Missing, short, and old default signing keys are rejected at startup. For explicit local development only, `APP_ENV=development` enables the fallback key. Tokens issued before session/purpose enforcement require login again.

From the repository root, run the assertion-based checks against disposable PostgreSQL:

```bash
service/scripts/validate-business.sh
```

This builds canonical bindings in an isolated image, runs package checks, verifies administrator bootstrap and HTTP workflows, and tests the new migration down/up pairs. Docker access (or passwordless sudo), Go, curl and build dependencies/image access are required. It does not use the deployed database. See the [HTTP inventory and validation report](../docs/business-process-validation.md) for all access points, workflow cases and results.
