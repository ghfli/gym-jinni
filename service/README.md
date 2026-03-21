# Gym-Jinni backend (`service`)

Single Go module and binary: **gRPC :8080**, **HTTP gateway :8081**.

| Area | Package / paths |
|------|-----------------|
| Users | `user/`, `gen/go/user/v1alpha` |
| Classes (proto) | `class/pb` — server wiring TBD |
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

Includes user, class, RBAC schema, and RBAC seed data.

## Docker

Image build runs buf + sqlc inside the container (see `k8s-setup/docker/Dockerfile.service`).
