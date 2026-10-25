# GymJinni Software Specification

## 1. Purpose and scope

GymJinni helps gym owners, trainers, and customers manage accounts, classes, bookings, facilities, and daily gym activities. This specification consolidates the repository's implemented behavior, declared API contracts, and documented plans. It does not assume feature parity with the commercial products mentioned in the README.

**Evidence:** [README](README.md), [system design diagram](GymJinni_system_design.png), `service/` Go source, protobuf contracts, DBML/SQL schemas, migrations, `ui/` Flutter source, [RBAC notes](service/RBAC.md), and `k8s-setup/`. Local code takes precedence where prose disagrees. The externally linked Notion document was not used.

**Status definitions:**

- **Implemented in source:** Handler or supporting code exists; this does not establish that it builds or passes runtime tests.
- **Partial:** Some contracts, schemas, or handlers exist, but the complete workflow does not.
- **Planned:** Represented in design material without a working module.
- **Proposed:** A requirement introduced here to make a workflow implementable; it requires product/design confirmation.

No application builds or runtime tests were performed for this revision; statuses come from reading source.

## 2. Actors and authorization model

| Actor | Intended responsibilities | Seeded access |
| --- | --- | --- |
| Super administrator | Manage the system, roles, and permissions | Every permission present when the seed runs |
| Gym owner | Manage gym operations, trainers, classes, bookings, and reports | Explicitly assigned subset |
| Trainer | Manage own classes; view relevant customers/bookings | Class create/read/update; user and booking read |
| Customer | Browse classes and manage own bookings | Book/read/cancel bookings; read classes |
| Guest | Read public class information | Read classes |

Roles are additive and may expire. Migration `000004` seeds five roles and 21 permissions (user, class, booking, report, and role/permission groups); it does not assign roles to users. `manage_*` is an ordinary exact-match permission, not a wildcard. Ownership restrictions are intended behavior, not enforced by the global permission model.

**AUTHZ-01:** Resource handlers shall check both operation permission and ownership/scope. A trainer's class permission shall not permit editing another trainer's class; customers shall not access another customer's bookings.

**Open decision:** A `gym` module with membership exists (see section 8), but permissions are not scoped to a gym. Define tenant boundaries before exposing owner workflows in multi-gym deployments.

## 3. Architecture and module inventory

The system is a **modular monolith**: one Go module (`service/`) and one binary serving gRPC (`:8080`) and an HTTP/JSON gateway (`:8081`, with permissive CORS) over one PostgreSQL database with a schema per domain. The former separate `csr_service/` module no longer exists; RBAC is a normal module. The design image sketches a future distributed architecture; extraction into services is an option, not a started migration.

| Module | RPCs / handlers | Status | Location |
| --- | --- | --- | --- |
| User, sessions, trainer profiles | 10 / 10 | Implemented in source | `service/user/`, `service/auth/` |
| Class | 5 / 5 | Implemented in source | `service/class/` |
| RBAC | 23 / 23 | Implemented in source | `service/rbac/` |
| Booking | 5 / 5 | Partial (no waitlist) | `service/booking/` |
| Gym and membership | 6 / 6 | Implemented in source | `service/gym/` |
| Schedule | 5 / 5 | Implemented in source (no conflict rules verified) | `service/schedule/` |
| Payment | 4 / 4 | Implemented in source | `service/payment/` |
| Notification and preferences | 5 / 5 | Partial (stored/listed; no delivery channel or event bus) | `service/notification/` |
| Activity logging and stats | 4 / 4 | Implemented in source | `service/activity/` |
| Report | 3 / 3 | Partial (no DB layer of its own) | `service/report/` |
| Authorization middleware | n/a | Implemented and wired | `service/middleware/` |
| Flutter client | n/a | Partial | `ui/` |
| Kubernetes deployment, CI | n/a | Implemented tooling | `k8s-setup/`, `.github/workflows/`, `.gitlab-ci.yml` |
| Event bus, CDC, caches, audit log | n/a | Planned | Design image |

Request flow: client → HTTP gateway (loopback to gRPC) or gRPC → validator interceptor → `AuthInterceptor` (JWT identity + RBAC `CheckPermission`) → handler → sqlc bindings → PostgreSQL. Handlers embed the generated `Unimplemented*Server`, so any RPC without a handler returns Unimplemented.

## 4. User management

Handlers exist for `CreateUser`, `GetUser`, `UpdateUser` (including password change), `DeleteUser`, `ListTrainers`, `GetTrainerProfile`, and `UpdateTrainerProfile`. Passwords are hashed with bcrypt. Schema is in `service/schema/user.sql`; trainer profiles were added in migration `000007`.

- **USER-01:** Registration uses validated name and contact information. Reconcile the design (email and/or phone) with protobuf validation before release.
- **USER-02:** Never return or log passwords or hashes. The user constructor logs `DBURL`; remove connection strings from logs.
- **USER-03:** Define explicit self-service versus administrator rules for profile lookup, update, and deletion, and verify them with denial tests.
- **USER-04:** Define contact uniqueness (not enforced in the schema), verification, account status, and deletion/retention policies.

**Acceptance:** Valid registration persists a verifiable hash and returns only public fields. Invalid input fails without writes. Unauthorized profile access is rejected.

## 5. Authentication and sessions

`LoginUser` verifies the bcrypt hash, issues a JWT access token (15 min) and refresh token (7 days), and persists a `user.session` row (UUID, refresh token, user agent, client IP, blocked flag, expiry). `RenewAccessToken` issues a new access token from a refresh token with roles in claims. `LogoutUser` blocks the session. Signing uses HS256-style secret from `JWT_SECRET`, defaulting to a development value.

The interceptor reads the token from `authorization`, `grpcgateway-authorization`, or `grpc-metadata-authorization` metadata. Public methods: `LoginUser`, `CreateUser`, `RenewAccessToken`, `GetRoleByName`, `AssignRoleToUser`.

- **AUTH-01:** Fail startup (or refuse non-dev mode) when `JWT_SECRET` is the default value.
- **AUTH-02:** Verify session state (blocked/expired) on protected requests and renewal, not only token signature and expiry. Confirm renewal rejects blocked sessions.
- **AUTH-03:** Review the public-method list: `AssignRoleToUser` being public lets unauthenticated callers assign roles, including privileged ones, unless the handler itself restricts it. Confirm intent or remove.
- **AUTH-04 (proposed):** Rotate refresh tokens on renewal with replay detection; define revocation on password change and account deletion.

**Acceptance:** Wrong credentials, expired/revoked sessions, and forged tokens fail. Logout prevents further renewal.

## 6. Class management

`class` provides `CreateClass`, `GetClass`, `ListClasses`, `UpdateClass`, and `DeleteClass` over a model with creator, description, start/end time, and minimum/maximum headcount.

- **CLASS-01:** Validate `start_time < end_time`, nonnegative headcounts, and minimum not exceeding maximum; define optional-limit semantics.
- **CLASS-02:** Derive creator from authenticated identity; enforce trainer ownership and gym scope.
- **CLASS-03 (proposed):** Define scheduled/cancelled/completed states and coordinate cancellation or capacity changes with bookings and notifications. Do not reduce capacity below confirmed bookings.
- **CLASS-04:** Specify how minimum attendance affects running or cancelling a class.

**Acceptance:** Authorized users manage permitted classes; invalid times/capacities fail; listing is chronological.

## 7. RBAC and authorization middleware

`rbac` declares 23 RPCs, all with handlers: role and permission CRUD (including by-name lookup), role-permission assign/revoke/list, user-role assign/revoke/list/details/users-with-role, and `CheckPermission`, `CheckPermissionByResourceAction`, `GetUserPermissions`. Tables are `rbac.role`, `rbac.permission`, `rbac.role_permission`, `rbac.user_role` (migration `000003`). Effective permissions are the distinct union of roles whose `expires_at` has not passed; an expired-role cleanup query exists without a scheduler.

`middleware/permissions.go` maps full gRPC method names to permission names; unmapped methods require authentication only. `AuthInterceptor` is wired in `service.go`; the service also holds an internal loopback client to RBAC, and the user module receives an RBAC client via `SetRBACClient`.

- **RBAC-01:** Preserve additive evaluation, duplicate-safe assignments, and expiry filtering; validate target users, roles, permissions, and grantors (no foreign keys to users today).
- **RBAC-02:** Document and implement an administrator bootstrap procedure.
- **RBAC-03:** Restrict permission inspection to the caller or administrators.
- **RBAC-04 (proposed):** Default-deny methods that lack a policy entry instead of allowing any authenticated user.
- **RBAC-05:** Derive grantor identity from the verified token and record administrative changes in an audit log.
- **RBAC-06:** Distinguish operational failure from denial in permission checks without exposing database internals.

**Acceptance:** Multiple roles combine correctly; expired/revoked roles stop authorizing; unauthorized role administration and cross-user introspection fail over both transports.

## 8. Booking, gyms, and scheduling

**Booking:** `CreateBooking`, `CancelBooking`, `GetBooking`, `ListUserBookings`, `ListClassBookings` (migration `000005`). No waitlist exists.

- **BOOK-01:** Booking confirmation and capacity accounting shall be atomic; concurrent requests never exceed class maximum.
- **BOOK-02:** Prevent duplicate active bookings per user/class; retried calls shall not double-count.
- **BOOK-03 (proposed):** Waitlist with FIFO promotion in one transaction on cancellation. Priority, acceptance windows, and cutoffs remain product decisions.
- **BOOK-04:** Customers see and cancel only their own bookings; staff within scope.

**Gym:** `CreateGym`, `GetGym`, `ListGyms`, `UpdateGym`, `AddMember`, `ListMembers` (migration `000006`). Gym scope is not yet used by class, booking, or RBAC checks.

**Schedule:** CRUD for schedules (migration `000008`).

- **ROOM-01:** Reject overlapping active reservations for the same room, including concurrent writes. **Proposed:** half-open intervals `[start, end)`.
- **ROOM-02:** Define recurrence timezone, occurrence generation, and conflict handling before recurring schedules are supported.

**Acceptance:** A concurrency test for the last seat confirms at most one contender; overlapping reservations cannot both succeed; cancellation frees exactly one seat.

## 9. Payments, notifications, activity, and reports

- **Payment** (`CreatePayment`, `GetPayment`, `ListPayments`, `RefundPayment`; migration `000009`). **PAY-01:** No payment-provider integration is specified; define provider, idempotency keys, currency/amount rules, and refund authority before real money moves. Never store card data.
- **Notification** (`SendNotification`, `ListNotifications`, `MarkRead`, `GetPreferences`, `UpdatePreference`; migration `000010`). **NOTIFY-01:** No delivery channel, scheduler, or event bus exists. Choose channels/providers, reminders, retries, and duplicate suppression. **EVENT-01 (proposed):** Persist event intent with domain changes (outbox) and make consumers idempotent.
- **Activity** (`LogActivity`, `GetActivity`, `ListActivities`, `GetStats`; migration `000011`) records user workout/activity data. Define activity types and units.
- **Report** (`GetDashboard`, `GetRevenueReport`, `GetAttendanceReport`). **REPORT-01:** Gate by permission and data scope; confirm metrics, timezone, and export formats; reconcile totals with source records.
- **AUDIT-01 (planned):** Record administrative actions with verified actor, target, action, timestamp, and outcome, excluding credentials and tokens.

## 10. Flutter client

`ui/lib/main.dart` implements an auth screen (register/login), app shell, dashboard, class list, my bookings, activity log, and profile pages; `auth_state.dart` holds session state and `api_config.dart` reads `API_BASE` (`--dart-define`; empty means same-origin behind the nginx `/v1/` proxy). Web is the primary target; other platform scaffolds are untested. Dart constraint: `>=3.0.0 <4.0.0`. The only test is `test/widget_test.dart`.

- **UI-01:** Add trainer/owner class and schedule management and an administrator interface for roles, permissions, assignments, and expiry.
- **UI-02:** Display loading, empty, validation, permission-denied, expired-session, and network-failure states; use secure token storage per platform.
- **UI-03:** Split `main.dart` (single large file) into feature files as screens are added.
- **UI-04:** Add widget/integration tests covering login, browsing, booking/cancellation, and permission-sensitive screens. The server remains authoritative for permissions.

## 11. API, validation, and data requirements

| Interface | Default binding |
| --- | --- |
| gRPC | `0.0.0.0:8080` (loopback `127.0.0.1:8080` used by gateway and internal RBAC client) |
| HTTP gateway | `:8081`, routes under `/v1/` |
| Kubernetes via k3d load balancer | host ports `39080` / `39081` / `39300` |

**API-01:** Protobuf is the contract source; keep gateway annotations, DBML, SQL queries, schema SQL, and migrations synchronized. Validate nested payload presence, IDs, strings, time ranges, and state transitions.

**API-02:** Map invalid input, unauthenticated, forbidden, not found, conflict, and infrastructure errors to consistent gRPC/HTTP statuses without raw SQL errors. Add pagination to list RPCs.

**DATA-01:** PostgreSQL is authoritative. Capacity and room-conflict rules must be transactional.

**DATA-02:** All migrations form one ordered stream in `service/mig/` (`000001`–`000011`). `service/schema/*.sql` is the committed sqlc source regenerated from DBML via `make dbsch`; keep it in step with migrations.

## 12. Deployment, operations, and quality

Tooling: Go, Buf, sqlc, golang-migrate, PostgreSQL; Podman images from `k8s-setup/container/Dockerfile.service` and `Dockerfile.ui` (buf and sqlc run inside the build); k3d + Ansible local cluster via `k8s-setup/setup.sh`; GitHub Actions and GitLab CI build images on `main`, `master`, and `csr`, with deploys manual. Vagrant/Arch Linux (`setup-alx.sh`) is a development aid.

- **OPS-01:** Use environment-specific `DBURL` and `JWT_SECRET` from secrets; do not reuse the development credentials in the Dockerfile default, `Makefile`, or manifests. Remove connection-string and sensitive-request logging. Restrict CORS from `*` in production.
- **OPS-02:** Propagate gRPC startup failures to the process, close resources, shut down gracefully, and add readiness/liveness checks.
- **OPS-03:** Define backup/restore and migration rollback procedures. Keep destructive teardown targets (`make teardn`) out of routine validation.
- **OPS-04:** Keep the module list in sync across the Makefile `SVCS`, the Dockerfile sqlc loop, and `service.go`. `report` has no DB layer and is excluded from sqlc.
- **QUALITY-01:** High availability, consistency, and low latency are design goals; no numeric SLOs exist. Establish them before adding caches, replication, or distributed services.

## 13. Verification and delivery

**Existing checks and gaps:**

- Backend: `make bufgen dbsql`, `make buflint`, build; `make test` boots the service and runs `cmd/grpcclt.go` and `cmd/httpclt.sh` against local PostgreSQL. These smoke clients create data and mostly print responses; they are not assertion-based. There are no Go unit tests (`*_test.go`) and no coverage threshold.
- Flutter: `flutter pub get`, `flutter analyze`, `flutter test`.
- Nothing in this document has been verified by building or running.

**Proposed next steps:**

1. **Hardening:** Close the public `AssignRoleToUser`, default-deny unmapped methods, enforce `JWT_SECRET`, check session state, remove sensitive logging.
2. **Ownership and scope:** Enforce creator/owner/gym scope in class, booking, schedule, and payment handlers (AUTHZ-01).
3. **Concurrency:** Make booking capacity atomic, add waitlist, and add schedule conflict rules, each with concurrency tests.
4. **Tests:** Add assertion-based smoke tests and Go unit tests, including authorization denials.
5. **Communication and insight:** Add delivery channels with an outbox, audit logging, and agreed reports.
6. **UI:** Add admin and trainer/owner screens and tests.

Each delivered module shall include contract/schema updates, migrations, permission policy, success/failure tests, and an end-to-end scenario: registration → login → permitted class creation → booking → cancellation → notification, including permission denial and concurrent last-seat requests.

## 14. Decisions still required

- Single-gym versus multi-gym scope and ownership model.
- Contact requirements and uniqueness, verification, account deletion, and retention.
- Token transport, rotation, and administrator bootstrap.
- Class lifecycle, minimum attendance, booking cutoffs, waitlist, and cancellation policies.
- Room ownership, recurrence, scheduling timezone, and conflict resolution.
- Payment provider and refund policy; notification channels and preferences.
- Report definitions, audit retention, supported clients, production hosting, and measurable SLOs.

Subscriptions/memberships billing and social features have no concrete contracts here; references to comparable products alone do not make them planned.
