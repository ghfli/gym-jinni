# RBAC (roles & permissions)

RBAC lives in this repo under **`service/rbac/`** and **`service/middleware/`**. It is served by the same process as user (and future class) APIs:

- **gRPC**: port `8080` (default)
- **HTTP gateway**: port `8081` — e.g. `GET /v1/roles`, `GET /v1/permissions`

## Database

- Schema: `rbac` (tables: `role`, `permission`, `role_permission`, `user_role`)
- Migrations: `service/mig/000003_init_schema_rbac.*`, `000004_seed_default_roles_permissions.*`
- SQLC schema source: `service/schema/rbac.sql`

## Middleware

`service/middleware/` maps gRPC methods to permission names and provides `AuthInterceptor(rbacClient)` for optional enforcement (wire in `service.go` when you have stable auth metadata).

## Codegen

From `service/`:

```bash
make bufmup bufgen   # protobuf + gateway
make dbsql           # sqlc (needs committed schema/*.sql)
```

Refresh `schema/*.sql` from DBML when you change `rbac/db/rbac.dbml` (requires `dbml2sql`): `make dbsch`.
