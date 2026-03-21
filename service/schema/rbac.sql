CREATE SCHEMA rbac;

CREATE TABLE rbac.role (
  id serial PRIMARY KEY,
  name varchar UNIQUE NOT NULL,
  description varchar,
  created_at timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE rbac.permission (
  id serial PRIMARY KEY,
  name varchar UNIQUE NOT NULL,
  resource varchar NOT NULL,
  action varchar NOT NULL,
  description varchar,
  created_at timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE rbac.role_permission (
  role_id int NOT NULL REFERENCES rbac.role (id) ON DELETE CASCADE,
  permission_id int NOT NULL REFERENCES rbac.permission (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT (now()),
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE rbac.user_role (
  user_id int NOT NULL,
  role_id int NOT NULL REFERENCES rbac.role (id) ON DELETE CASCADE,
  granted_by int,
  granted_at timestamptz NOT NULL DEFAULT (now()),
  expires_at timestamptz,
  PRIMARY KEY (user_id, role_id)
);
