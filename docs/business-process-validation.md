# HTTP and business-process validation

Audit date: 2026-10-29. Contracts and source establish declared capabilities; only executed checks establish runtime behavior. Test failures are requirement failures and remain enabled. This audit adds tests and implementation tasks rather than changing application business behavior.

## Access points and web screens

| Address | Purpose | Read-only observation |
| --- | --- | --- |
| http://localhost:39300/ | Flutter web application | HTTP 200, HTML shell; browser rendering not verified |
| http://localhost:39300/v1/classes | Same-origin nginx API proxy | HTTP 401 without a token |
| http://localhost:39081/v1/classes | Direct JSON gateway | HTTP 401 without a token |
| http://localhost:39081/v1/roles/by-name/customer | Public role lookup | HTTP 200, customer role returned |
| http://localhost:39081/v1/users/1/activities | Protected activity API | HTTP 401 without a token |
| http://localhost:39081/v1/reports/dashboard | Protected reports API | HTTP 401 without a token |
| localhost:39080 | gRPC transport | Configured/listening; not an HTTP page; transport not probed |
| http://localhost:8081/ | Default gateway for standalone service | Connection refused during audit |

The deployed UI uses one URL and internal navigation, with no named/deep-link routes:

| Screen | Access/navigation | Business actions |
| --- | --- | --- |
| Register / Log in | Initial unauthenticated screen, two tabs | Create customer account; obtain tokens |
| Home | Authenticated default | Welcome and links to four tabs; no report API metrics |
| Classes | Classes tab / Browse Classes | List, view class dialog, book |
| My Bookings | Bookings tab / My Bookings | List own bookings and cancel confirmed bookings |
| Activity | Activity tab | Log workout/run/class activity; list and show totals |
| Profile | Profile tab | Show in-memory name/ID; local logout only |

The nginx image also maps `/api/` and `/rbac/` to the gateway root; e.g. `/api/v1/classes` rewrites to `/v1/classes`. These are configured aliases, not additional business APIs, and were not probed. Ingress depends on a configured domain/TLS and its rewrite behavior is unverified. `API_BASE` defaults to `http://127.0.0.1:39081` in source; the UI image explicitly builds with an empty value for same-origin requests.

## Complete HTTP contract inventory

70 operations across 10 services, derived from protobuf annotations. `JWT` means authentication without a mapped permission. `JWT (unmapped)` exposes a policy gap. Public entries describe current policy, not approval of that policy. Permission checks do not establish ownership enforcement. Body/query shapes remain defined by protobuf.

| Domain | HTTP operation | RPC | Current access | Screen |
| --- | --- | --- | --- | --- |
| [activity](../service/activity/pb/activity/v1alpha/activity.proto) | `POST /v1/activities` | `LogActivity` | JWT | Activity |
| [activity](../service/activity/pb/activity/v1alpha/activity.proto) | `GET /v1/activities/{id}` | `GetActivity` | JWT | API only |
| [activity](../service/activity/pb/activity/v1alpha/activity.proto) | `GET /v1/users/{user_id}/activities` | `ListActivities` | JWT | Activity |
| [activity](../service/activity/pb/activity/v1alpha/activity.proto) | `GET /v1/users/{user_id}/activity_stats` | `GetStats` | JWT | Activity |
| [booking](../service/booking/pb/booking/v1alpha/booking.proto) | `POST /v1/bookings` | `CreateBooking` | JWT + `book_class` | Classes |
| [booking](../service/booking/pb/booking/v1alpha/booking.proto) | `POST /v1/bookings/{id}/cancel` | `CancelBooking` | JWT + `cancel_booking` | My Bookings |
| [booking](../service/booking/pb/booking/v1alpha/booking.proto) | `GET /v1/bookings/{id}` | `GetBooking` | JWT + `view_booking` | API only |
| [booking](../service/booking/pb/booking/v1alpha/booking.proto) | `GET /v1/users/{user_id}/bookings` | `ListUserBookings` | JWT + `view_booking` | My Bookings |
| [booking](../service/booking/pb/booking/v1alpha/booking.proto) | `GET /v1/classes/{class_id}/bookings` | `ListClassBookings` | JWT + `view_booking` | API only |
| [class](../service/class/pb/class/v1alpha/class.proto) | `POST /v1/classes` | `CreateClass` | JWT + `create_class` | API only |
| [class](../service/class/pb/class/v1alpha/class.proto) | `GET /v1/classes/{id}` | `GetClass` | JWT + `view_class` | API only |
| [class](../service/class/pb/class/v1alpha/class.proto) | `GET /v1/classes` | `ListClasses` | JWT (unmapped) | Classes |
| [class](../service/class/pb/class/v1alpha/class.proto) | `PUT /v1/classes/{class.id}` | `UpdateClass` | JWT + `update_class` | API only |
| [class](../service/class/pb/class/v1alpha/class.proto) | `DELETE /v1/classes/{id}` | `DeleteClass` | JWT + `delete_class` | API only |
| [gym](../service/gym/pb/gym/v1alpha/gym.proto) | `POST /v1/gyms` | `CreateGym` | JWT + `manage_gym` | API only |
| [gym](../service/gym/pb/gym/v1alpha/gym.proto) | `GET /v1/gyms/{id}` | `GetGym` | JWT + `view_gym` | API only |
| [gym](../service/gym/pb/gym/v1alpha/gym.proto) | `GET /v1/gyms` | `ListGyms` | JWT + `view_gym` | API only |
| [gym](../service/gym/pb/gym/v1alpha/gym.proto) | `PUT /v1/gyms/{id}` | `UpdateGym` | JWT + `manage_gym` | API only |
| [gym](../service/gym/pb/gym/v1alpha/gym.proto) | `POST /v1/gyms/{gym_id}/members` | `AddMember` | JWT + `manage_gym` | API only |
| [gym](../service/gym/pb/gym/v1alpha/gym.proto) | `GET /v1/gyms/{gym_id}/members` | `ListMembers` | JWT + `view_gym` | API only |
| [notification](../service/notification/pb/notification/v1alpha/notification.proto) | `POST /v1/notifications` | `SendNotification` | JWT + `send_notification` | API only |
| [notification](../service/notification/pb/notification/v1alpha/notification.proto) | `GET /v1/users/{user_id}/notifications` | `ListNotifications` | JWT | API only |
| [notification](../service/notification/pb/notification/v1alpha/notification.proto) | `POST /v1/notifications/{id}/read` | `MarkRead` | JWT | API only |
| [notification](../service/notification/pb/notification/v1alpha/notification.proto) | `GET /v1/users/{user_id}/notification_preferences` | `GetPreferences` | JWT | API only |
| [notification](../service/notification/pb/notification/v1alpha/notification.proto) | `PUT /v1/notification_preferences` | `UpdatePreference` | JWT | API only |
| [payment](../service/payment/pb/payment/v1alpha/payment.proto) | `POST /v1/payments` | `CreatePayment` | JWT + `create_payment` | API only |
| [payment](../service/payment/pb/payment/v1alpha/payment.proto) | `GET /v1/payments/{id}` | `GetPayment` | JWT + `view_payment` | API only |
| [payment](../service/payment/pb/payment/v1alpha/payment.proto) | `GET /v1/users/{user_id}/payments` | `ListPayments` | JWT + `view_payment` | API only |
| [payment](../service/payment/pb/payment/v1alpha/payment.proto) | `POST /v1/payments/{id}/refund` | `RefundPayment` | JWT + `manage_payment` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `POST /v1/roles` | `CreateRole` | JWT + `manage_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/roles/{id}` | `GetRole` | JWT + `view_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/roles/by-name/{name}` | `GetRoleByName` | Public | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/roles` | `ListRoles` | JWT + `view_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `PUT /v1/roles/{id}` | `UpdateRole` | JWT + `manage_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `DELETE /v1/roles/{id}` | `DeleteRole` | JWT + `manage_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `POST /v1/permissions` | `CreatePermission` | JWT + `manage_permissions` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/permissions/{id}` | `GetPermission` | JWT + `view_permissions` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/permissions/by-name/{name}` | `GetPermissionByName` | JWT + `view_permissions` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/permissions` | `ListPermissions` | JWT + `view_permissions` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `PUT /v1/permissions/{id}` | `UpdatePermission` | JWT + `manage_permissions` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `DELETE /v1/permissions/{id}` | `DeletePermission` | JWT + `manage_permissions` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `POST /v1/roles/{role_id}/permissions/{permission_id}` | `AssignPermissionToRole` | JWT + `manage_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `DELETE /v1/roles/{role_id}/permissions/{permission_id}` | `RevokePermissionFromRole` | JWT + `manage_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/roles/{role_id}/permissions` | `GetRolePermissions` | JWT + `view_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `POST /v1/users/{user_id}/roles/{role_id}` | `AssignRoleToUser` | Public | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `DELETE /v1/users/{user_id}/roles/{role_id}` | `RevokeRoleFromUser` | JWT + `manage_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/users/{user_id}/roles` | `GetUserRoles` | JWT + `view_user` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/users/{user_id}/role-details` | `GetUserRoleDetails` | JWT + `view_user` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/roles/{role_id}/users` | `GetUsersWithRole` | JWT + `view_roles` | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `POST /v1/check-permission` | `CheckPermission` | JWT | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `POST /v1/check-permission-by-resource-action` | `CheckPermissionByResourceAction` | JWT | API only |
| [rbac](../service/rbac/pb/rbac/v1alpha/rbac.proto) | `GET /v1/users/{user_id}/permissions` | `GetUserPermissions` | JWT | API only |
| [report](../service/report/pb/report/v1alpha/report.proto) | `GET /v1/reports/dashboard` | `GetDashboard` | JWT + `view_reports` | API only |
| [report](../service/report/pb/report/v1alpha/report.proto) | `GET /v1/reports/revenue` | `GetRevenueReport` | JWT + `view_reports` | API only |
| [report](../service/report/pb/report/v1alpha/report.proto) | `GET /v1/reports/attendance` | `GetAttendanceReport` | JWT + `view_reports` | API only |
| [schedule](../service/schedule/pb/schedule/v1alpha/schedule.proto) | `POST /v1/schedules` | `CreateSchedule` | JWT + `manage_schedule` | API only |
| [schedule](../service/schedule/pb/schedule/v1alpha/schedule.proto) | `GET /v1/schedules/{id}` | `GetSchedule` | JWT + `view_schedule` | API only |
| [schedule](../service/schedule/pb/schedule/v1alpha/schedule.proto) | `GET /v1/schedules` | `ListSchedules` | JWT + `view_schedule` | API only |
| [schedule](../service/schedule/pb/schedule/v1alpha/schedule.proto) | `PUT /v1/schedules/{id}` | `UpdateSchedule` | JWT + `manage_schedule` | API only |
| [schedule](../service/schedule/pb/schedule/v1alpha/schedule.proto) | `DELETE /v1/schedules/{id}` | `DeleteSchedule` | JWT + `manage_schedule` | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `POST /v1/create_user` | `CreateUser` | Public | Register |
| [user](../service/user/pb/user/v1alpha/user.proto) | `POST /v1/login_user` | `LoginUser` | Public | Log in |
| [user](../service/user/pb/user/v1alpha/user.proto) | `GET /v1/users/{user.id}` | `GetUser` | JWT + `view_user` | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `PUT /v1/users/{user.id}` | `UpdateUser` | JWT + `update_user` | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `POST /v1/renew_token` | `RenewAccessToken` | Public | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `DELETE /v1/users/{user.id}` | `DeleteUser` | JWT + `delete_user` | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `POST /v1/logout_user` | `LogoutUser` | JWT | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `GET /v1/users/{user_id}/trainer_profile` | `GetTrainerProfile` | JWT + `view_user` | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `PUT /v1/trainer_profile` | `UpdateTrainerProfile` | JWT + `update_user` | API only |
| [user](../service/user/pb/user/v1alpha/user.proto) | `GET /v1/trainers` | `ListTrainers` | JWT + `view_user` | API only |

## Workflow test matrix

Use fresh customer A/B, trainer A/B, gym owner, administrator, expired-role user, and unauthenticated caller. Privileged roles are assigned through test-only SQL fixtures. Run normal seed checks before supplemental fixtures; no test should silently invent permissions to make a planned flow pass.

| Workflow / requirements | Preconditions and steps | Expected outcome and failure cases | Coverage |
| --- | --- | --- | --- |
| Registration / USER-01/02, RBAC-02 | Register unique email, verify DB hash/customer role, log in | Public response excludes credentials; wrong password and invalid contact rejected; duplicate-contact policy needs decision | Automated hash, role, login; manual invalid/duplicate contacts |
| Sessions / AUTH-02/05, UI-06 | Log in, renew, log out, renew again; interchange token purposes; submit expired/forged token | Valid renewal succeeds; revoked, expired, forged and wrong-purpose tokens fail; logout records blocked session | Automated security cases; manual normal renewal and cross-session logout |
| Role administration / AUTH-03, RBAC-01/07 | Anonymous role grant; admin grant/revoke/expire; compare seeded permission names to policy | Anonymous grant denied; grant/revoke/expiry affect next request; every required permission exists | Automated anonymous grant, missing seeds, expiry/revoke; manual additive roles and admin CRUD |
| Profiles / USER-03, AUTHZ-01 | A reads/updates/deletes own profile; B tries A; trainer edits profile | Only self or authorized staff can access/change records; not-found is distinct from denial | Source-only; manual matrix; no profile-edit UI |
| Classes / CLASS-01/02, RBAC-08 | Trainer creates/lists; customer tries create; spoof creator; submit reversed times/capacities; revoke viewing | Chronological visible list; proper deny; identity cannot be spoofed; invalid values create no record | Automated create/list, denial, spoofing, invalid range; manual ordering/update/delete |
| Booking / BOOK-01/02/04 | Customer books one-seat class; repeats; cancels; rebooks; another customer books; missing ID | One confirmed seat, duplicate 409, full 429, persisted cancellation and released seat, missing 404 | Automated journey; read/list ownership and forged booking-user tests |
| Concurrency / BOOK-01/02 | Release 16 contenders simultaneously for one seat; repeat same user on roomy class | Exactly one HTTP success and exactly one confirmed DB record | Automated; a passing race run alone does not prove atomicity |
| Waitlist and lifecycle / BOOK-03, CLASS-03/04 | Fill class, join queue, cancel seat; cancel class; complete attendance | FIFO transactional promotion, cancellation notifications and defined cutoff/minimum policy | Unimplemented; policy decisions and tasks in spec |
| Gym membership / AUTHZ-01, RBAC-07 | Owner creates gym/adds member; unrelated owner lists or modifies it | Permission and gym-scoped access; duplicate membership behavior defined | Source-only; manual after permission seeds; no management UI |
| Schedule / ROOM-01/02 | Create/update reservations, overlap same resource, adjacent times, concurrent overlap, recurrence over DST | Overlaps rejected atomically; half-open boundaries and timezone policy defined | Source-only CRUD; resource model/conflict/occurrence workflows unimplemented |
| Payments / PAY-01/02 | Create pending payment, retry, authorize settlement, refund twice, cross-user read, reconcile revenue | Provider-confirmed state; no duplicate charges; authorized valid refunds; currency-separated totals | Source-only ledger; provider/settlement/idempotency unimplemented |
| Notifications / NOTIFY-01, EVENT-01 | Booking/cancel event, read inbox, mark own read, change preferences, retry delivery | Transactional event intent; ownership; preferences respected; retry without duplicates | Source-only inbox/preferences; event delivery unimplemented |
| Activity / AUTHZ-01, UI-05 | Log two workouts, list, compare totals, B requests A | Totals 2/40/200; private records; stats envelope/field names decoded | Automated totals, cross-user read, widget stats; manual invalid units and all write ownership |
| Reports / REPORT-01/02 | Seed date-boundary bookings and pending/settled/refunded payments across gyms/currencies | Authorized scoped totals reconcile with source; attendance dates honored; completed/no-show transitions exist | Source-only; manual fixture reconciliation; no report UI |
| Web / UI-01/02/04/05/06 | Login, navigate, view/cancel booking, stats, deny/error states, logout; refresh browser | Correct gateway JSON; explicit errors; server logout; documented storage/session expiry behavior | Automated widgets for critical states; manual browser/network/expiry and missing staff screens |

For manual cases record actor, fixture IDs, request/status, before/after DB values, actual result, and requirement/task. Assert rejected writes leave state unchanged. Use service contracts for payload shapes; test missing nested payloads and invalid identifiers across CRUD operations (API-01/02). Payment/provider and notification-delivery cases remain blocked until adapters exist; do not represent stored rows as completed payments or delivered messages.

## Reproduction and evidence

Run from the repository root:

```bash
service/scripts/validate-business.sh
(cd service && go test ./...)
(cd ui && flutter test --no-pub)
(cd ui && flutter analyze --no-pub)
```

The API runner requires Docker access (or passwordless `sudo -n docker`), Go, curl and image/build dependency access. It regenerates protobuf/sqlc in a fresh service image using the existing Dockerfile, creates a uniquely named network and PostgreSQL database, applies migrations in order with stop-on-error, and exposes randomized loopback ports. It never uses the deployed database. `POSTGRES_TEST_IMAGE` can override the default PostgreSQL 16 Alpine image. It removes only its own containers/network/image and preserves logs under the printed `/tmp/gj-validation.*` directory. Existing service logging can include fixture credentials; keep those logs local. Expected requirement failures produce a nonzero exit code.

`service/integration` skips when its two validation environment variables are absent; a skipped run is not API validation. Do not point those variables at a shared database. The runner supplies both. Existing `make test` changes local data and prints responses without asserting correctness, so it is not used here.

The source build regenerates bindings inside the container because ignored local `service/gen/` files are stale: duplicate helper definitions, incompatible ID/pointer types, and incorrect activity aggregate types prevent a host build. Canonical generated wire JSON uses enum names and lowerCamelCase fields; handwritten local bindings may emit numeric enums and snake_case. The widgets test both relevant representations. Generated files are neither edited nor committed.

### Executed results (2026-10-29)

| Check | Result | Evidence / linked task |
| --- | --- | --- |
| Fresh container protobuf/sqlc generation and service compilation | Passed | Existing Dockerfile built current source successfully; host ignored bindings are incompatible (QUALITY-02) |
| Migrations 000001–000011 on PostgreSQL 16 Alpine | Passed | Isolated runner applied every up migration with `ON_ERROR_STOP=1` |
| Seed permission completeness | Failed, eight missing | manage_gym, view_gym, manage_schedule, view_schedule, create_payment, view_payment, manage_payment, send_notification (RBAC-07) |
| Customer journey | Passed | Public-field/hash/default role assertions, trainer create/customer browse, booking duplicate 409, persisted cancellation, rebooking, full-class 429, customer create denial 403 and missing booking 404 |
| Activity totals | Passed | Two 20-minute/100-calorie activities aggregate to 2/40/200 |
| Ordinary authentication, role expiry/revoke, expired token | Passed | Missing/forged/expired credentials denied; removing or expiring all roles denies GetClass |
| Anonymous privileged role grant | Failed | HTTP 200 instead of 401 (AUTH-03/RBAC-02) |
| Cross-user activity and booking lists, forged creator and booking user | Failed | Each returned HTTP 200 instead of 403 (AUTHZ-01) |
| Roleless class listing | Failed | HTTP 200 instead of 403; ListClasses is unmapped (RBAC-08) |
| Renewal after logout and token-purpose interchange | Failed | All three returned HTTP 200 instead of 401 (AUTH-02/05) |
| Invalid class time/capacity | Failed | HTTP 200 instead of 400 (CLASS-01) |
| Last-seat race, 16 different users | Passed this run only | Exactly one success/confirmed row; source check/count/insert is still non-transactional (BOOK-01) |
| Duplicate race, 16 requests for same user/class | Failed | Three successes and three confirmed records (BOOK-02) |
| Flutter widgets | Three passed, four failed | Registration/login/navigation and numeric booking cancellation pass; canonical enum, stats envelope, visible 403 and server logout fail (UI-02/05/06) |
| Flutter analysis | Passed | No issues after behavior-preserving braces/formatting |
| Integration package without explicit environment | Compiles and skips | Safety guard only; not a runtime test result |
| Canonical `go test ./...` | Failed in existing smoke client | `service/cmd/grpcclt.go` calls undefined RenewAccessTokenClient and passes uint32 where int32 is required (QUALITY-03); other application packages compile, integration tests skip without environment |
| Canonical `buf lint` | Passed | Exit 0; existing DEFAULT-category deprecation warnings |
| Runner syntax / patch whitespace | Passed | `bash -n service/scripts/validate-business.sh`; `git diff --check` |

The initial isolated run completed the HTTP suite in 8.94 seconds after building/migrating. Failure assertions remain active and make both requirement suites exit nonzero. No test failure was reclassified as success. Raw output was retained locally at `/tmp/gj-validation.4x6ktU/results.txt`; the durable sanitized evidence is the table above. Existing deployment probes were read-only. No end-to-end browser automation, external payment/delivery integration, or manual cases above have been claimed as passed.

Canonical package/lint checks were run in the Dockerfile builder stage (rather than modifying local ignored bindings):

```bash
sudo docker build --target builder -f k8s-setup/container/Dockerfile.service -t gj-validation-checks .
sudo docker run --rm gj-validation-checks go test ./...
sudo docker run --rm gj-validation-checks buf lint
sudo docker image rm gj-validation-checks
```

The `go test ./...` command is expected to remain nonzero until QUALITY-03 is delivered. The image tag above is temporary validation tooling; use an unused tag if running concurrently. Implementation evidence is in the per-domain protobuf links in the inventory, the matching `service/<domain>/<domain>.go` handlers and DB queries, `service/middleware/{auth,permissions}.go`, migration `000004`, and Flutter `main.dart`/`auth_state.dart`. Report calculations and schedule/payment limitations were reviewed in source, not executed as complete business workflows.
