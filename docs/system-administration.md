# System administration

Register an ordinary account first. A database operator provisions the first administrator from `service/` using `DBURL=... go run ./cmd/bootstrap-admin -user-id N`. The command refuses to create another administrator after one exists; subsequent grants use authenticated administration. Never pass a database URL containing credentials into shared logs.

Sign in through the Flutter UI. An active `super_admin` grant adds **Management** to navigation. The screen provides:

- **Users:** paged account directory, role details, grants with optional UTC expiry, and revocation.
- **Roles:** create/edit/delete custom roles and manage their permission assignments.
- **Permissions:** create/edit/delete permissions. Referenced records may require removing assignments first.

The reserved `super_admin` role cannot be renamed/deleted, and its permissions cannot be revoked through the management API. Permissions assigned to it cannot be deleted or have their name/resource/action changed; descriptions remain editable. Before removing, expiring or deleting an active administrator, grant another account permanent administrator access. Operations serialize with first-admin bootstrap and preserve a permanent replacement. Direct database operators remain trusted and can bypass application guards.

Role expiry uses `YYYY-MM-DDTHH:MM:SSZ`; an empty expiry means permanent. Invalid or past dates are rejected. Permission changes take effect on subsequent protected requests. If management-access discovery fails, the UI shows a retry action. After receiving a new administrator grant, sign out/in to reload navigation. UI visibility never replaces backend authorization: authorization mutations and `GET /v1/users` independently require system administrator scope.

`GET /v1/users?page_size=50&offset=0` returns public profile fields and `nextOffset`; zero means the final page. Page size is capped at 200. Password credentials are never populated. Existing user/role/permission endpoints remain available for API clients.

These screens provide global system administration. Gym-scoped staff management, scheduling, attendance, payment and operational-report screens remain separate milestones in the [delivery ledger](mvp-delivery.md).
