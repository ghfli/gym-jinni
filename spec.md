# GymJinni Software Specification

Documentation reviewed 2026-10-31. [Development/tooling](docs/development.md), [deployment](k8s-setup/README.md) and [UI guide](ui/README.md) contain current commands. Initial audit failures in section 15 are historical; section 16 records delivered remediation.

## 1. Purpose and scope

GymJinni helps gym owners, trainers, and customers manage accounts, classes, bookings, facilities, and daily gym activities. This specification consolidates the repository's implemented behavior, declared API contracts, and documented plans. It does not assume feature parity with the commercial products mentioned in the README.

**Evidence:** [README](README.md), [system design diagram](GymJinni_system_design.png), `service/` Go source, protobuf contracts, DBML/SQL schemas, migrations, `ui/` Flutter source, [RBAC notes](service/RBAC.md), and `k8s-setup/`. Local code takes precedence where prose disagrees. The externally linked Notion document was not used.

**Status definitions:**

- **Implemented in source:** Handler or supporting code exists; this does not establish that it builds or passes runtime tests.
- **Partial:** Some contracts, schemas, or handlers exist, but the complete workflow does not.
- **Planned:** Represented in design material without a working module.
- **Proposed:** A requirement introduced here to make a workflow implementable; it requires product/design confirmation.

The 2026-10-29 validation built the service with fresh container-generated bindings, applied all migrations to disposable PostgreSQL, and ran API integration and Flutter widget tests. Existing source statuses remain distinct from runtime results. See [HTTP inventory and business-process validation](docs/business-process-validation.md) for routes, workflow cases, evidence, and limitations.

## 2. Actors and authorization model

| Actor | Intended responsibilities | Seeded access |
| --- | --- | --- |
| Super administrator | Manage the system, roles, and permissions | Every permission present when the seed runs |
| Gym owner | Manage gym operations, trainers, classes, bookings, and reports | Explicitly assigned subset |
| Trainer | Manage own classes; view relevant customers/bookings | Class create/read/update; user and booking read |
| Customer | Browse classes and manage own bookings | Book/read/cancel bookings; read classes |
| Guest | Read public class information | Read classes |

Roles are additive and may expire. Migration `000004` seeds five roles and 21 permissions (user, class, booking, report, and role/permission groups); registration assigns the customer role transactionally; the seed itself does not assign roles to users. Migration `000012` adds eight module permissions and reviewed grants. `manage_*` is an ordinary exact-match permission, not a wildcard. Operation permission and handler ownership checks are separate. Private resources are restricted to their owner or an active super administrator; class creators can inspect their own class bookings. Broader gym-scoped staff access remains pending.

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
- **USER-02:** Never return or log passwords or hashes. Connection strings and credential-bearing registration payloads are no longer logged by service constructors/registration.
- **USER-03:** Define explicit self-service versus administrator rules for profile lookup, update, and deletion, and verify them with denial tests.
- **USER-04:** Define contact uniqueness (not enforced in the schema), verification, account status, and deletion/retention policies.

**Acceptance:** Valid registration persists a verifiable hash and returns only public fields. Invalid input fails without writes. Unauthorized profile access is rejected.

## 5. Authentication and sessions

`LoginUser` verifies the bcrypt hash, issues a session-bound JWT access token (15 min) and refresh token (7 days) with distinct token purposes, and persists a `user.session` row (UUID, refresh token, user agent, client IP, blocked flag, expiry). `RenewAccessToken` verifies the persisted refresh token and active session before issuing access. `LogoutUser` checks session ownership and blocks the session; subsequent access and renewal fail. Signing requires HS256, the expected issuer/subject/expiry/purpose, and a session ID. Startup requires a dedicated `JWT_SECRET` of at least 32 bytes unless `APP_ENV=development` explicitly enables the development fallback. Existing tokens without the new claims require login again.

The interceptor reads the token from `authorization`, `grpcgateway-authorization`, or `grpc-metadata-authorization` metadata. Public methods: `LoginUser`, `CreateUser`, `RenewAccessToken`, `GetRoleByName`. Customer provisioning occurs inside the account transaction; administrative role grants require authentication and `manage_roles`. The database-operator bootstrap command is documented in [RBAC notes](service/RBAC.md).

- **AUTH-01 (implemented):** Startup rejects missing/short/default signing keys outside explicit `APP_ENV=development`; dedicated keys require at least 32 bytes.
- **AUTH-02 (implemented):** Protected requests and renewal verify persisted active session/account state. Logout blocks access and renewal for that session; renewal also checks the persisted refresh token.
- **AUTH-03 (implemented):** Role grants require authentication and `manage_roles`; customer provisioning is transactional and first-admin bootstrap requires database-operator access. Anonymous grants are denied.
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

`middleware/permissions.go` maps full gRPC method names to permission names; unmapped methods are denied, while deliberately authenticated-only methods have explicit empty policies. `AuthInterceptor` is wired in `service.go`; the service also holds an internal loopback client to RBAC, and the user module provisions the default customer role directly in the registration transaction; there is no unauthenticated internal role-grant endpoint.

- **RBAC-01:** Preserve additive evaluation, duplicate-safe assignments, and expiry filtering; validate target users, roles, permissions, and grantors (no foreign keys to users today).
- **RBAC-02 (implemented):** The operator command bootstraps the first active super administrator; subsequent grants use the authenticated API. See [RBAC instructions](service/RBAC.md).
- **RBAC-03 (implemented):** User-specific role/permission inspection is restricted to the caller or active super administrators, in addition to operation policy.
- **RBAC-04 (implemented):** Unmapped methods are denied; a policy-completeness regression checks declared RPCs. Authenticated-only operations have explicit empty permission entries.
- **RBAC-05:** Derive grantor identity from the verified token and record administrative changes in an audit log.
- **RBAC-06:** Distinguish operational failure from denial in permission checks without exposing database internals.

**Acceptance:** Multiple roles combine correctly; expired/revoked roles stop authorizing; unauthorized role administration and cross-user introspection fail over both transports.

## 8. Booking, gyms, and scheduling

**Booking:** `CreateBooking`, `CancelBooking`, `GetBooking`, `ListUserBookings`, `ListClassBookings` (migration `000005`). Confirmation, cancellation and capacity changes serialize on the class row; a partial unique index prevents duplicate confirmed user/class bookings. No waitlist exists.

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

`ui/lib/main.dart` implements an auth screen (register/login), app shell, dashboard, class list, my bookings, activity log, and profile pages; `auth_state.dart` holds session state and `api_config.dart` reads `API_BASE` (`--dart-define`; empty means same-origin behind the nginx `/v1/` proxy). Web is the primary target; other platform scaffolds are untested. Dart constraint: `>=3.0.0 <4.0.0`. Critical-flow widget tests are in `test/widget_test.dart`; HTTP clients are injected through Provider. The original requirement assertions remain enabled and passed after remediation; section 15 preserves historical failures and section 16 separates completed/remaining work.

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

**DATA-02:** All migrations form one ordered stream in `service/mig/` (`000001`–`000013`). `service/schema/*.sql` is the committed sqlc source regenerated from DBML via `make dbsch`; keep it in step with migrations.

## 12. Deployment, operations, and quality

Tooling: Go, Buf, sqlc, golang-migrate, PostgreSQL; Docker images locally/GitHub and Podman images in GitLab from `k8s-setup/container/Dockerfile.service` and `Dockerfile.ui` (buf and sqlc run inside the build); k3d + Ansible local cluster via `k8s-setup/setup.sh`; GitHub Actions and GitLab CI build images on `main`, `master`, and `csr`, with deploys manual. Vagrant/Arch Linux (`setup-alx.sh`) is a development aid.

- **OPS-01:** Use environment-specific `DBURL` and `JWT_SECRET` from secrets; do not reuse the development credentials in the Dockerfile default, `Makefile`, or manifests. Remove connection-string and sensitive-request logging. Restrict CORS from `*` in production.
- **OPS-02:** Propagate gRPC startup failures to the process, close resources, shut down gracefully, and add readiness/liveness checks.
- **OPS-03:** Define backup/restore and migration rollback procedures. Keep destructive teardown targets (`make teardn`) out of routine validation.
- **OPS-04:** Keep the module list in sync across the Makefile `SVCS`, the Dockerfile sqlc loop, and `service.go`. `report` has no DB layer and is excluded from sqlc.
- **OPS-05 (remaining):** Signing-secret references/provisioning and fail-closed migration init are implemented. Remaining work: align GHCR image references/pull policy with CI output; render raw ingress placeholders before kubectl apply; define production migration recovery/backup procedures; provision cert-manager/issuer prerequisites explicitly in GitLab. The current local templates and CI configuration do not implement these production deployment steps end-to-end. See [deployment prerequisites](k8s-setup/README.md#configure-before-deploying).
- **QUALITY-01:** High availability, consistency, and low latency are design goals; no numeric SLOs exist. Establish them before adding caches, replication, or distributed services.

## 13. Verification and delivery

**Existing checks and gaps:**

- Backend: `make bufgen dbsql`, `make buflint`, build; `make test` boots the service and runs `cmd/grpcclt.go` and `cmd/httpclt.sh` against local PostgreSQL. These smoke clients create data and mostly print responses; they are not assertion-based. Assertion-based Go HTTP integration tests now live in `service/integration/`, with the isolated runner `service/scripts/validate-business.sh`; no coverage threshold is configured.
- Flutter: `flutter pub get`, `flutter analyze`, `flutter test`.
- Runtime validation and historical failures are recorded in the [validation report](docs/business-process-validation.md). The remediation adds canonical package checks, bootstrap/ownership tests, booking serialization/constraints and UI session regressions. Host ignored bindings still require canonical regeneration; the container build runs protobuf lint, generation and `go test ./...` without modifying those local files.

**Remaining delivery priorities (completed hardening is in section 16):**

1. **Hardening:** Add refresh rotation/replay detection, password-change revocation, audit history and consistent infrastructure-error sanitization; retain session/policy regression coverage.
2. **Ownership and scope:** Extend current owner/system-admin checks into an explicit gym-scoped staff/tenant model (AUTHZ-01), including schedule/payment/report workflows.
3. **Concurrency:** Preserve atomic booking/capacity constraints; add the policy-defined waitlist and schedule resource-conflict rules with concurrency tests.
4. **Tests:** Extend existing assertion-based API, Go/Flutter and deployment-script tests for newly delivered workflows; avoid treating skipped environment tests as runtime evidence.
5. **Communication and insight:** Add delivery channels with an outbox, audit logging, and agreed reports.
6. **UI:** Add admin and trainer/owner screens and tests.

Each delivered module shall include contract/schema updates, migrations, permission policy, success/failure tests, and an end-to-end scenario: registration → login → permitted class creation → booking → cancellation → notification, including permission denial and concurrent last-seat requests.

## 14. Decisions still required

- Single-gym versus multi-gym scope and ownership model.
- Contact requirements and uniqueness, verification, account deletion, and retention.
- Token transport and refresh-token rotation; first-administrator bootstrap is now implemented.
- Class lifecycle, minimum attendance, booking cutoffs, waitlist, and cancellation policies.
- Room ownership, recurrence, scheduling timezone, and conflict resolution.
- Payment provider and refund policy; notification channels and preferences.
- Report definitions, audit retention, supported clients, production hosting, and measurable SLOs.

Subscriptions/memberships billing and social features have no concrete contracts here; references to comparable products alone do not make them planned.


## 15. Validated gaps and implementation tasks (2026-10-29)

The [validation report](docs/business-process-validation.md) lists all 70 HTTP operations, six client screens, test cases, and observed outcomes. The business workflow is the acceptance boundary: an existing CRUD handler does not establish safe authorization, a valid lifecycle, or delivery of external effects. Tests intentionally assert the required behavior and fail until the corresponding work is delivered. The initial audit evidence below is historical. Completed remediation and remaining work are distinguished in the delivery status in section 16.

Priorities: **P0** authorization/session integrity; **P1** correctness and operationally usable workflows; **P2** planned capabilities. Preserve these IDs when implementing work. Existing requirement IDs below are expanded rather than superseded.

| Task / requirement | Priority and evidence | Implementation work and dependencies | Acceptance criteria | Current status |
| --- | --- | --- | --- | --- |
| AUTH-03, RBAC-02: trusted provisioning | P0; anonymous super-admin assignment returned 200 | Remove role assignment from public methods; implement explicit administrator bootstrap and trusted internal default-customer provisioning that does not need a public privilege-grant endpoint. Derive grantor from verified identity (RBAC-05). | Anonymous grants return 401; unauthorized customer grants return 403; registration still grants only customer; bootstrap is documented/tested and cannot promote arbitrary users anonymously.  | Delivered; see section 16 |
| AUTH-02, **AUTH-05: session-bound token purpose** | P0; renewal after logout, access-as-refresh and refresh-as-access all returned 200 | Bind tokens to sessions and distinguish access from refresh tokens; enforce signature algorithm/issuer/type/expiry, persisted session state and session ownership. Refuse renewal when login could not persist the session. Coordinate UI-06. | Logged-out/blocked/expired sessions cannot renew or authorize protected requests; each token type is rejected in the other context; one user cannot revoke another user's session.  | Delivered; see section 16 |
| AUTHZ-01, USER-03, CLASS-02, BOOK-04 | P0; cross-user booking/activity reads, forged creator and booking user returned 200 | Use verified actor identity for self-service writes and enforce ownership on reads, updates, cancellation, activity, notifications/preferences and payments. Define gym membership/staff scope before enabling multi-gym management. | Customer A cannot read/write/cancel B's records; trainer A cannot impersonate trainer B; explicit scoped staff exceptions have success and denial tests; rejected writes do not alter state.  | Partial; remaining scope in section 16 |
| **RBAC-07: complete permission seeds** | P1; eight required permissions absent after all migrations | Add a numbered forward/down migration for manage_gym, view_gym, manage_schedule, view_schedule, create_payment, view_payment, manage_payment and send_notification, with reviewed least-privilege role grants including super-admin. Do not edit historical seeds as the upgrade strategy. | A fresh installation and upgraded installation both have every mapped permission and intended role grants; customer/staff denial cases pass.  | Delivered; see section 16 |
| **RBAC-08: contract-policy alignment**, RBAC-04 | P0; a roleless authenticated caller listed classes with 200 | Correct ListClass policy to ListClasses; distinguish explicit authenticated-only entries from absent entries; apply default-deny with a deliberate internal authorization-check policy. Depends on trusted provisioning. | All declared RPCs have explicit policy; roleless class listing returns 403; expiry/revocation affect the next protected call; adding an unmapped RPC fails a policy-completeness check.  | Delivered; see section 16 |
| CLASS-01, API-01/02 | P1; reversed time range/minimum greater than maximum accepted | Validate nested payload presence, temporal order and capacity bounds on create/update; define optional limits and capacity reduction rules with BOOK-01/02. Return consistent invalid-argument errors. | Invalid requests return 400 with no writes; valid boundary cases pass; updates cannot reduce capacity below confirmed seats.  | Partial; remaining scope in section 16 |
| BOOK-01/02 | P1; 16 concurrent same-user requests created three confirmed bookings | Make capacity checks and inserts atomic with suitable locking/constraints; add active-booking uniqueness and deterministic duplicate/conflict mapping. Coordinate cancellation and retries. | Concurrent last-seat requests leave at most one confirmed seat; concurrent duplicates leave exactly one active user/class booking; cancellation frees one seat and rebooking succeeds.  | Delivered; see section 16 |
| **UI-05: gateway response compatibility** | P1; canonical enum and nested activity-stat widgets failed | Decode protobuf enum names and nested stats, lowerCamelCase fields and int64 strings; retain numeric/snake_case compatibility only where existing supported responses need it. Use shared response decoding. | Canonical confirmed bookings display Confirmed and allow cancellation; totals show 2/40/200; numeric booking regression test still passes.  | Delivered; see section 16 |
| UI-02, **UI-06: visible errors and server logout** | P1; 403 hidden as empty list; local logout made no server call | Display permission/network errors distinctly from empty data; retain login session ID, call server logout, handle revocation failure explicitly, clear local state and implement expiry/renewal behavior. Depends on AUTH-02/05. | Denied list shows an error; logout revokes the correct session and returns to auth; old tokens fail; token expiry/network failure states have widget and integration coverage.  | Partial; remaining scope in section 16 |
| BOOK-03, CLASS-03/04 | P2; no waitlist or lifecycle orchestration | Decide queue/cutoff/minimum-attendance policy, then implement transactional promotion, class cancellation and capacity/lifecycle transitions. Depends on atomic booking and EVENT-01. | FIFO seat promotion has concurrency coverage; cancellation updates affected bookings and records notification intent; completed/no-show states have authorized transitions.  | Planned / requires decisions |
| ROOM-01/02 | P1 conflicts, P2 recurrence; schedules have no room/resource identity or occurrence generation | Define resource ownership/model, add contract/schema/migrations and atomic overlap prevention. Decide timezone/recurrence policy before occurrence generation. | Same-resource overlaps cannot both commit; adjacent half-open intervals pass if adopted; concurrent tests and timezone/DST recurrence cases pass.  | Planned / requires decisions |
| PAY-01, **PAY-02: settlement and financial lifecycle** | P1; create inserts ledger row; refund only changes status | Define amount/currency/refund rules and provider adapter, idempotency and verified settlement events. Restrict transitions/refunds and associate payment with the purchased booking/membership where required by the chosen product policy. | Retried requests cannot charge twice; pending rows cannot be treated as settled revenue; refunds require valid state/authority; provider failures cannot falsely record success.  | Planned / requires decisions |
| NOTIFY-01, EVENT-01 | P2; inbox exists, event-triggered delivery does not | Choose channels/reminder policies; implement transactional outbox, preferences, delivery workers, retry and duplicate suppression; scope inbox/read/preferences by actor. | Booking/cancellation commits notification intent atomically; opted-out channels are respected; retries deliver at most one logical notification; users cannot inspect/modify others' inboxes.  | Partial; remaining scope in section 16 |
| REPORT-01, **REPORT-02: measurable operational reporting** | P1; source counts non-refunded pending payments and global bookings; the attendance request has no date filters | Define timezone/status/currency/tenant metrics, filter attendance by requested dates, reconcile revenue with settled/refunded source records, and implement authorized attendance transitions. Depends on PAY-02 and lifecycle decisions. | Boundary/date/scoping fixtures reconcile with database; pending payments are excluded from settled revenue; currencies are not silently summed; completed/no-show totals are reachable and correct.  | Planned / requires decisions |
| UI-01/04 | P2; no staff/admin management screens | Add trainer/owner classes, schedules, gym membership, payments/notification/report views as their workflows become usable; add admin role/permission/expiry UI after P0 authorization work. | Each supported actor completes its workflow through the browser; hidden controls do not replace server denial; screens cover loading/empty/error states.  | Planned / requires decisions |
| **QUALITY-03: smoke-client contract compatibility** | P1; canonical `go test ./...` fails in `service/cmd/grpcclt.go` | Replace the removed RenewAccessTokenClient helper with the declared RenewAccessToken RPC and use int32 IDs consistent with the protobuf contracts. Keep secrets out of client output. | The smoke client compiles after canonical generation; all packages pass compilation under `go test ./...`; renewal uses the declared request/response contract.  | Delivered; see section 16 |
| **QUALITY-02: reproducible generated-code checks**, OPS-04 | P1; host generated files fail compilation, canonical container generation builds | Document/automate clean generation and checks without committing bindings; maintain module lists; run integration requirements and UI regression checks in a disposable environment and retain sanitized results. | Clean checkout builds after canonical generation; Go checks, protobuf lint and Flutter analysis have reported results; failed requirements remain visible and link to tasks, rather than being disabled.  | Delivered; see section 16 |

Existing proposed product policies remain decisions, not implemented guarantees: gym tenancy, waitlist cutoffs, recurrence semantics, provider/channel selection and report definitions require resolution before their features are delivered. No live database was changed by this validation.


## 16. Remediation delivery status

Implemented and covered by regression tests:

- **AUTH-01/02/03/05, RBAC-02/07/08:** Explicit signing configuration; session-bound access/refresh purposes; persisted-session validation on access and renewal; owned-session logout; authenticated role grants with verified grantor; transactional customer provisioning; database-operator first-admin bootstrap; complete permission seeds and default-deny policies. A policy-completeness test checks every declared RPC.
- **CLASS-01/02, BOOK-01/02/04:** UTC calendar/time and capacity validation; creator checks; owned/admin class updates; capacity reductions checked against confirmed bookings under the same class lock used by booking and cancellation. Active user/class booking uniqueness is enforced in PostgreSQL. Broader gym tenancy, waitlists and lifecycle transitions remain open.
- **USER-03 and AUTHZ-01 (partial):** Self-service profile reads/updates and guarded account deletion work with per-handler validation. Private activities, notifications/preferences, payment records and RBAC introspection require ownership/admin scope. Class creators can inspect their own class bookings. Gym member administration checks the gym owner; global reports require system-admin scope until tenant reports are implemented. Deleting an account with retained business records fails atomically, preserving its sessions/roles. Full retention policy and gym-scoped staff delegation remain open.
- **UI-05/06 and UI-02 (partial):** Canonical enum/nested-stat decoding, visible class/booking permission errors, session-ID retention, server logout, automatic renewal with one retry, shared renewal requests and late-response protection. Loading/error handling in other screens and secure persistent token storage remain open.
- **QUALITY-02/03:** The Docker service build runs pinned protobuf lint, canonical generation, and all Go package tests; the smoke client compiles against the current RPC/ID types. The isolated runner verifies bootstrap, HTTP workflows and migration round trips. Schema generation preserves constraints not representable by DBML through explicit overlays. Smoke-client privileged workflows still require correctly provisioned test actors; compilation is not proof its entire scenario can run as an unprivileged new customer.

Upgrade requirements: apply migrations `000012` and `000013`; review duplicate confirmed bookings and invalid class ranges/capacities before `000013` (the migration refuses invalid data rather than altering customer records). Configure a dedicated signing secret; old tokens require login. Provision the first administrator through the operator command, then use authenticated grants. No live deployment or database is changed automatically by this work.

Still planned: refresh-token rotation/replay detection and password-change revocation; gym-scoped staff/tenant workflows; waitlists/class lifecycle/attendance; room/resource conflict and recurrence models; external payment settlement/refund rules; notification delivery/outbox/reminders; currency-aware/date-filtered report contracts; and missing trainer/owner/admin screens. These require the dependencies and product decisions already listed above and are not represented as completed by CRUD handlers.


## 17. Tooling and deployment documentation status (2026-10-31)

GitHub deployment is now an explicit `workflow_dispatch` run with `deploy` enabled on main/master; pushes/PRs build/test and only non-PR runs publish. Kubeconfig decoding/context/API reachability checks precede cert-manager installation. Local Ansible and GitHub use shared rollout and bounded admission-readiness checks; GitLab has a separate manual deploy path. Deployment-script regressions and workflow/Ansible syntax checks passed during that fix; no live cluster installation was claimed.

The workflow actions use native Node.js 24 and the forcing override is removed. Optional host Node.js tooling is independent of CI runtimes; the [development guide](docs/development.md#optional-nodejs-tooling) gives portable selection/verification guidance without prescribing a workstation layout.

The documentation audit corrected stale no-unit-test/default-secret/Podman-default/automatic-deploy claims and links to nonexistent test reports. It preserved earlier runtime evidence as dated history and identified OPS-05 configuration gaps from the current manifests. No application code, runtime behavior or live deployment is changed by this documentation revision.


## 18. Deployment review follow-up

The backend manifest now requires `gym-jinni-auth/JWT_SECRET`; shared deployment provisioning creates a random key only if absent, validates existing/optional explicit keys and refuses implicit rotation. Ansible, GitHub and GitLab invoke it before applying the backend. Missing required Secret data blocks container creation; deployed configuration does not enable development fallback.

The migration init command uses `exec migrate ... up` with error propagation. PostgreSQL connectivity/schema/data failures block application startup; no-change succeeds. Regression tests exercise the exact manifest command's exit status and signing-key lifecycle. OPS-05 retains the distinct registry/rendering/recovery work still outstanding.

Contributor instructions are centralized in AGENTS.md; CLAUDE.md links them. Validation script examples consistently run from the repository root. Active guides use test-command output instead of fixed counts, and shared tooling guidance is machine-independent. Shorter Kubernetes guides were reviewed individually and expanded with source/execution maps and concrete startup/secret/migration diagnosis.


## 19. Deployment lifecycle review and next plan

The existing TCP wait was strengthened to bounded `pg_isready` availability using PostgreSQL configuration fields. Migration failures still block startup. A committed, structurally parsed Docker harness reproduces cold start, fresh/no-change/upgrade, migration-13 constraint failures and verified rollback-state recovery; the operator runbook defines when marker repair is permissible.

Signing-key bytes are compared without stripping newlines; namespace drift fails before cluster access. Normal redeployment preserves the key and rejects unexpected changes. An explicit operator command applies a resource-version-checked update without deleting the Secret, restarts/waits for backend rollout and supports retries after an interrupted rollout. Single-key rotation invalidates prior tokens; zero-disruption multi-key rollover remains separate work.

Ansible helper diagnostics are visible without rendering the operator key in task arguments. GitLab now runs the same pinned-dependency deployment suite as GitHub and checks the deployment image's shell/utility prerequisites. Tests load YAML by resource/container name, including reordered formatting and extra-container fixtures. Essential generated-code, host-port and Buf dependency rules are recorded in AGENTS.md and explicitly referenced by CLAUDE.md.

The [focused deployment improvement plan](docs/deployment-improvement-plan.md) defines separate follow-up changes for registry/domain rendering, release migration coordination, scoped deployment credentials and disposable rollout acceptance. None is silently bundled into this lifecycle correction.
