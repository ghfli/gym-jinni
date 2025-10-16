-- based on ../gen/sql/rbac.dbml.sql
-- update me whenever it changes

CREATE SCHEMA IF NOT EXISTS "rbac";

CREATE TABLE IF NOT EXISTS "rbac"."role" (
  "id" serial PRIMARY KEY,
  "name" varchar UNIQUE NOT NULL,
  "description" varchar,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE IF NOT EXISTS "rbac"."permission" (
  "id" serial PRIMARY KEY,
  "name" varchar UNIQUE NOT NULL,
  "resource" varchar NOT NULL,
  "action" varchar NOT NULL,
  "description" varchar,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE IF NOT EXISTS "rbac"."role_permission" (
  "role_id" int NOT NULL,
  "permission_id" int NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  PRIMARY KEY ("role_id", "permission_id")
);

CREATE TABLE IF NOT EXISTS "rbac"."user_role" (
  "user_id" int NOT NULL,
  "role_id" int NOT NULL,
  "granted_by" int,
  "granted_at" timestamptz NOT NULL DEFAULT (now()),
  "expires_at" timestamptz,
  PRIMARY KEY ("user_id", "role_id")
);

ALTER TABLE "rbac"."role_permission" ADD FOREIGN KEY ("role_id") REFERENCES "rbac"."role" ("id") ON DELETE CASCADE;
ALTER TABLE "rbac"."role_permission" ADD FOREIGN KEY ("permission_id") REFERENCES "rbac"."permission" ("id") ON DELETE CASCADE;
ALTER TABLE "rbac"."user_role" ADD FOREIGN KEY ("role_id") REFERENCES "rbac"."role" ("id") ON DELETE CASCADE;

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_user_role_user_id ON "rbac"."user_role" ("user_id");
CREATE INDEX IF NOT EXISTS idx_user_role_role_id ON "rbac"."user_role" ("role_id");
CREATE INDEX IF NOT EXISTS idx_user_role_expires_at ON "rbac"."user_role" ("expires_at");
CREATE INDEX IF NOT EXISTS idx_role_permission_role_id ON "rbac"."role_permission" ("role_id");
CREATE INDEX IF NOT EXISTS idx_role_permission_permission_id ON "rbac"."role_permission" ("permission_id");
CREATE INDEX IF NOT EXISTS idx_permission_resource_action ON "rbac"."permission" ("resource", "action");

