# Migration startup and recovery

The pod has two ordered init containers: `wait-for-postgres` uses `pg_isready` against the configured host/port/database/user, retrying with a bounded wait; `run-migrations` then executes the real migration CLI. Database unavailability waits first. Migration/data errors remain failures and prevent application startup. An already-current schema is a successful no-change run.

## Diagnose without altering data

Use the intended kubeconfig and namespace:

```bash
kubectl get pods -n gym-jinni -l app=main-service
POD=replace-with-failing-pod
kubectl logs -n gym-jinni "$POD" -c wait-for-postgres
kubectl logs -n gym-jinni "$POD" -c run-migrations
kubectl describe pod -n gym-jinni "$POD"
```

Check host/port/database configuration and PostgreSQL events when the wait times out. `pg_isready` checks server availability, not password validity; credential failures still appear in migration logs. Do not replace either step with `|| true` or start the application against a partial schema.

## Migration 000013 recovery decision

Pause competing deployment/migration operations and take a verified database backup before data repair. Use an authorized database connection to inspect:

```sql
SELECT version, dirty FROM schema_migrations;
SELECT user_id, class_id, count(*) FROM booking.booking
WHERE status = 'confirmed' GROUP BY user_id, class_id HAVING count(*) > 1;
SELECT id, start_time, end_time, min_hdcnt, max_hdcnt FROM "class"."class"
WHERE start_time >= end_time OR coalesce(min_hdcnt, 0) < 0
   OR coalesce(max_hdcnt, 0) < 0
   OR (coalesce(max_hdcnt, 0) > 0 AND coalesce(min_hdcnt, 0) > max_hdcnt);
SELECT to_regclass('booking.booking_confirmed_user_class');
SELECT conname FROM pg_constraint
WHERE conrelid = '"class"."class"'::regclass
  AND conname IN ('class_time_range', 'class_capacity');
```

Have the data owner approve how duplicate bookings or invalid class values are repaired. There is no automatic cancellation/deletion of customer records. Restore a backup or follow a reviewed schema-repair procedure if migration state or installed objects differ from expectations.

Only for a failed **000013** whose recorded state is **version 13, dirty=true**, after confirming all migration-13 DDL rolled back (no index and neither constraint above), correcting the affected data and verifying migration 12 is otherwise intact, an operator may reset the marker to the known prior version and retry. From the repository root, with `DBURL` supplied through the normal protected environment:

```bash
migrate -path service/mig -database "$DBURL" force 12
migrate -path service/mig -database "$DBURL" up
migrate -path service/mig -database "$DBURL" version
```

`force` changes only the migration marker; it never repairs data or schema. Do not use this recipe for an unknown failure, partial DDL, another migration version or an unreviewed database. After clean migration 13 is confirmed, allow/restart the blocked rollout and inspect application readiness. Do not bypass a failed migration merely to make pods ready.

## Reproduce the checks safely

Install the pinned Python test dependencies in your test environment, build/select a service image containing the migration CLI, and run from the repository root:

```bash
python3 -m pip install -r k8s-setup/tests/requirements.txt
python3 -m unittest discover -s k8s-setup/tests -v
docker build -f k8s-setup/container/Dockerfile.service -t gym-jinni/service:migration-check .
python3 k8s-setup/tests/verify_migrations.py --service-image gym-jinni/service:migration-check
```

The committed harness reads YAML structurally, executes the actual wait/init commands, cold-starts a disposable PostgreSQL instance, and tests fresh/no-change/upgrade plus duplicate/range failure and verified rollback-state recovery. It mounts the current migrations read-only and changes only synthetic fixture records. It creates uniquely named containers/network and removes them on exit; it never uses a supplied production DBURL or the current Kubernetes context. Docker access (or passwordless `sudo -n docker`), images and network/cache access are required. `--postgres-image` can explicitly select a compatible test PostgreSQL image; the default matches the manifest wait image.

The unit suite covers simulated exit propagation and bounded waits. The Docker harness supplies reproducible real-database evidence; neither claims a live cluster rollout.
