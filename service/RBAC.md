# RBAC (roles & permissions)

RBAC lives in this repo under **`service/rbac/`** and **`service/middleware/`**. It is served by the same process as all domain APIs:

- **gRPC**: port `8080` (default)
- **HTTP gateway**: port `8081` — e.g. `GET /v1/roles`, `GET /v1/permissions`

## Database

- Schema: `rbac` (tables: `role`, `permission`, `role_permission`, `user_role`)
- Migrations: `service/mig/000003_init_schema_rbac.*`, `000004_seed_default_roles_permissions.*`
- SQLC schema source: `service/schema/rbac.sql`

## Middleware

`service/middleware/` maps gRPC methods to permission names and provides `AuthInterceptor(rbacClient, usersvc.ValidateSession)`, wired in `service.go`. Every declared RPC has an explicit public, authenticated-only, or permission policy; unknown methods are denied. Access/refresh JWTs have distinct purposes and carry a persisted session ID. Logout revokes access and renewal for that session. Pre-upgrade tokens without these claims must log in again.

Registration assigns only the customer role in the account transaction. Role assignment is authenticated, requires `manage_roles`, and records the verified grantor. Private resources require ownership; cross-user access is restricted to active super administrators except class booking access for the class creator. Global reports require system-admin scope until tenant reports exist.

## First administrator

Register the intended account, then a database operator can run from `service/`:

```bash
# Supply DBURL through your normal environment/secret mechanism.
go run ./cmd/bootstrap-admin -user-id 123
```

The command takes a transaction lock, validates the existing account and role, and refuses once an active super administrator exists. Subsequent grants use authenticated role administration. There is no public bootstrap endpoint.

Migration `000012` adds missing module permissions and grants them to super administrators; gym owners receive gym read/manage and trainers receive profile-update permission. Schedule, payment and notification administration remain system-admin operations by default. Migration `000013` adds booking uniqueness and class constraints. Review duplicate confirmed bookings and invalid class rows before applying it; migration failure preserves existing data instead of silently cancelling bookings.

## Codegen

From `service/`:

```bash
make bufmup bufgen   # protobuf + gateway
make dbsql           # sqlc (needs committed schema/*.sql)
```

Refresh `schema/*.sql` from DBML when you change `rbac/db/rbac.dbml` (requires `dbml2sql`): `make dbsch`.

## Signing configuration

Set a unique `JWT_SECRET` of at least 32 bytes before starting the service. Startup rejects the old default key, missing keys, and short keys. `APP_ENV=development` explicitly enables the local fallback; never set it in a deployed environment. `make test` uses that explicit development mode.
