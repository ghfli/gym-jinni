#!/usr/bin/env python3
"""Run manifest readiness/migrations/recovery in disposable Docker resources.

From the repository root, after installing tests/requirements.txt:
  python3 k8s-setup/tests/verify_migrations.py --service-image gym-jinni/service:latest
The image needs the migrate CLI; the current repository migrations are mounted read-only.
"""
import argparse
from pathlib import Path
import shutil
import subprocess
import time
import uuid

from manifest_utils import ROOT, named_container


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--service-image", default=named_container("run-migrations", init=True)["image"])
    parser.add_argument("--postgres-image", default=named_container("wait-for-postgres", init=True)["image"])
    args = parser.parse_args()
    docker = ["docker"]
    if not shutil.which("docker"):
        parser.error("Docker is required")
    if subprocess.run(docker + ["info"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode:
        docker = ["sudo", "-n", "docker"]
    subprocess.run(docker + ["info"], stdout=subprocess.DEVNULL, check=True)
    run_id = "gj-migrations-" + uuid.uuid4().hex[:12]
    wait = named_container("wait-for-postgres", init=True)
    migration = named_container("run-migrations", init=True)
    waiter = None

    def run(argv, *, sql=None, timeout=60, check=True):
        result = subprocess.run(docker + argv, input=sql, text=True,
                                capture_output=True, timeout=timeout)
        if check and result.returncode:
            raise RuntimeError(f"Isolated Docker operation failed: {result.stderr[-1200:]}")
        return result

    def dsn(database):
        return f"postgresql://validation:validation@postgres:5432/{database}?sslmode=disable&search_path=public"

    def migrate(database, operation="up"):
        # Use the exact manifest command for up; explicit commands set up/recover fixtures.
        script = migration["args"] if operation == "up" else [
            'exec migrate -path /app/service/mig -database "$DBURL" ' + operation]
        return run(["run", "--rm", "--network", run_id, "-e", "DBURL=" + dsn(database),
                    "--mount", f"type=bind,src={ROOT}/service/mig,dst=/app/service/mig,readonly",
                    "--entrypoint", migration["command"][0], args.service_image]
                   + migration["command"][1:] + script, check=False)

    def sql(database, statement):
        return run(["exec", "-i", run_id, "psql", "-v", "ON_ERROR_STOP=1", "-At",
                    "-U", "validation", "-d", database], sql=statement).stdout.strip()

    try:
        # Pull before creating resources if needed, so readiness measures startup, not download time.
        for image in {args.service_image, args.postgres_image}:
            if run(["image", "inspect", image], check=False).returncode:
                run(["pull", image], timeout=300)
        run(["network", "create", run_id])
        # Start the real readiness command before PostgreSQL exists, then cold-start the DB.
        waiter = subprocess.Popen(docker + [
            "run", "--rm", "--name", run_id + "-wait", "--network", run_id,
            "-e", "PGHOST=postgres", "-e", "PGPORT=5432", "-e", "PGUSER=validation",
            "-e", "PGDATABASE=fresh", "--entrypoint", wait["command"][0],
            args.postgres_image] + wait["command"][1:] + wait["args"],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        time.sleep(1)
        run(["run", "-d", "--name", run_id, "--network", run_id,
             "--network-alias", "postgres", "-e", "POSTGRES_USER=validation",
             "-e", "POSTGRES_PASSWORD=validation", "-e", "POSTGRES_DB=fresh", args.postgres_image])
        output, errors = waiter.communicate(timeout=250)
        if waiter.returncode:
            raise RuntimeError("Manifest readiness failed during cold start: " + errors[-800:])
        print("PASS manifest PostgreSQL readiness across cold start", flush=True)
        for label in ["fresh install", "no-change"]:
            result = migrate("fresh")
            if result.returncode:
                raise RuntimeError(result.stderr)
            print("PASS manifest migration: " + label, flush=True)
        for database, problem in [("upgrade", None), ("duplicate", "duplicate"), ("invalidrange", "range")]:
            sql("fresh", "CREATE DATABASE " + database)
            result = migrate(database, "goto 12")
            if result.returncode:
                raise RuntimeError(result.stderr)
            if problem:
                end = "2030-01-01 09:00+00" if problem == "range" else "2030-01-01 11:00+00"
                sql(database, 'INSERT INTO "user"."user"(id,name,hashed_passwd) VALUES(1,\'fixture\',\'unused\');'
                    'INSERT INTO "class"."class"(id,created_by,description,start_time,end_time) '
                    f"VALUES(1,1,'fixture','2030-01-01 10:00+00','{end}');")
                if problem == "duplicate":
                    sql(database, "INSERT INTO booking.booking(user_id,class_id) VALUES(1,1),(1,1);")
            result = migrate(database)
            if not problem:
                if result.returncode:
                    raise RuntimeError(result.stderr)
                print("PASS valid migration 12 to latest upgrade", flush=True)
                continue
            constraint = "booking_confirmed_user_class" if problem == "duplicate" else "class_time_range"
            if not result.returncode or constraint not in result.stderr:
                raise RuntimeError("Expected migration 13 constraint failure did not occur")
            # Prove the documented recovery preconditions: dirty 13, all 13 DDL rolled back.
            if sql(database, "SELECT version || ':' || dirty FROM schema_migrations;") != "13:true":
                raise RuntimeError("Unexpected failed migration state")
            if sql(database, "SELECT to_regclass('booking.booking_confirmed_user_class') IS NULL;") != "t":
                raise RuntimeError("Migration 13 index was not rolled back")
            if sql(database, "SELECT count(*) FROM pg_constraint WHERE conname IN ('class_time_range','class_capacity');") != "0":
                raise RuntimeError("Migration 13 constraints were not rolled back")
            print("PASS migration 13 blocks startup for " + problem, flush=True)
            # Repair only synthetic disposable fixtures; never apply these changes to customer data.
            repair = ("DELETE FROM booking.booking WHERE id=(SELECT max(id) FROM booking.booking);"
                      if problem == "duplicate" else
                      'UPDATE "class"."class" SET end_time=\'2030-01-01 11:00+00\' WHERE id=1;')
            sql(database, repair)
            for operation in ["force 12", "up"]:
                result = migrate(database, operation)
                if result.returncode:
                    raise RuntimeError(result.stderr)
            if sql(database, "SELECT version || ':' || dirty FROM schema_migrations;") != str(max(int(p.name.split("_")[0]) for p in (ROOT / "service/mig").glob("*.up.sql"))) + ":false":
                raise RuntimeError("Recovery did not restore the latest clean migration")
            print("PASS verified rollback-state recovery for " + problem, flush=True)
    finally:
        run(["rm", "-fv", run_id + "-wait", run_id], check=False)
        if waiter and waiter.poll() is None:
            waiter.communicate(timeout=10)
        run(["network", "rm", run_id], check=False)


if __name__ == "__main__":
    main()
