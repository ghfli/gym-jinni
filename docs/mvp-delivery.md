# Multi-gym MVP delivery ledger

The accepted boundary is a multi-gym internal MVP. Live payment charges and external message delivery stay disabled. Existing CRUD methods do not establish completed business workflows. Each increment includes its focused tests and documentation; generated bindings are never committed.

| Order | Requirements | Completion evidence required | State |
| --- | --- | --- | --- |
| 1 | OPS-05 deployment configuration | Local/remote render tests, immutable images, migration gate, namespace privileges, disposable rollout | Delivered; real isolated rollout passed, one CI authority required |
| 2 | AUTH-04 rotation/replay/password change | Atomic refresh rotation, stored hashes, replay revocation, Flutter replacement-token tests | Delivered; disposable API and Flutter tests passed |
| 3 | AUTHZ-01 gym tenancy | Explicit legacy mapping, membership scopes, cross-gym denial tests | Planned |
| 4 | UI-01/04 administration | Browser role/permission/gym/member workflows and last-admin protection | Partial: global administration delivered; gym staff flows remain |
| 5 | BOOK-03, CLASS-03/04 | FIFO concurrency, cancellation, attendance and notification intent | Planned |
| 6 | ROOM-01/02 | Room/trainer conflicts and bounded timezone-aware recurrence | Planned |
| 7 | EVENT-01, NOTIFY-01 | Durable inbox outbox, retries, preferences, reminder suppression | Planned |
| 8 | PAY-02, REPORT-02 | Manual ledger/idempotency, settlement/refund events, reconciled scoped/currency reports | Partial: global currency/date corrections delivered; ledger/tenant workflows remain |
| 9 | API-01/02, QUALITY-02, UI-02 | Consistent validation/errors, audited staff actions and complete actor journeys | Partial: activity validation and management error states delivered |

Implementation defaults below are requirements for the planned milestones, not guarantees of current behavior. Active gym membership is required for booking; trainer scope is assigned classes; managers/owners operate within their gym. Booking/cancellation/promotion close at class start. Minimum attendance warns staff rather than cancelling automatically. Attendance starts at class start; completion requires class end. Capacity increases promote waitlisted users in FIFO order. Class cancellation creates refund-review work, never an automatic financial refund.

Gym timezone defaults to UTC. Recurrence supports bounded daily/weekly series with an end date and at most 365 occurrences; previews disclose skipped nonexistent DST times and the earlier choice for ambiguous times. Reminders default to 24 hours and respect current booking state/preferences. Audit history is retained without automatic deletion. Sessions remain memory-based in the browser.

Historical records with ambiguous gym ownership require explicit operator mapping; never infer a tenant. Extend protobufs additively and use numbered migrations. No live cluster changes, customer-data repairs, pushes, real charges, or external delivery are implicit in local implementation.
