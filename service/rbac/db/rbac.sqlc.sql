-- Role CRUD Operations

-- name: CreateRole :one
INSERT INTO rbac.role (
  name, description
) VALUES (
  $1, $2
)
RETURNING *;

-- name: GetRole :one
SELECT * FROM rbac.role
WHERE id = $1 LIMIT 1;

-- name: GetRoleByName :one
SELECT * FROM rbac.role
WHERE name = $1 LIMIT 1;

-- name: ListRoles :many
SELECT * FROM rbac.role
ORDER BY name;

-- name: UpdateRole :exec
UPDATE rbac.role
SET name = $2, description = $3
WHERE id = $1;

-- name: DeleteRole :exec
DELETE FROM rbac.role
WHERE id = $1;

-- Permission CRUD Operations

-- name: CreatePermission :one
INSERT INTO rbac.permission (
  name, resource, action, description
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetPermission :one
SELECT * FROM rbac.permission
WHERE id = $1 LIMIT 1;

-- name: GetPermissionByName :one
SELECT * FROM rbac.permission
WHERE name = $1 LIMIT 1;

-- name: ListPermissions :many
SELECT * FROM rbac.permission
ORDER BY resource, action;

-- name: UpdatePermission :exec
UPDATE rbac.permission
SET name = $2, resource = $3, action = $4, description = $5
WHERE id = $1;

-- name: DeletePermission :exec
DELETE FROM rbac.permission
WHERE id = $1;

-- Role-Permission Operations

-- name: AssignPermissionToRole :exec
INSERT INTO rbac.role_permission (
  role_id, permission_id
) VALUES (
  $1, $2
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: RevokePermissionFromRole :exec
DELETE FROM rbac.role_permission
WHERE role_id = $1 AND permission_id = $2;

-- name: GetRolePermissions :many
SELECT p.* FROM rbac.permission p
JOIN rbac.role_permission rp ON p.id = rp.permission_id
WHERE rp.role_id = $1
ORDER BY p.resource, p.action;

-- name: ListRolePermissions :many
SELECT 
  r.id as role_id,
  r.name as role_name,
  p.id as permission_id,
  p.name as permission_name,
  p.resource,
  p.action
FROM rbac.role r
JOIN rbac.role_permission rp ON r.id = rp.role_id
JOIN rbac.permission p ON rp.permission_id = p.id
ORDER BY r.name, p.resource, p.action;

-- User-Role Operations

-- name: AssignRoleToUser :exec
INSERT INTO rbac.user_role (
  user_id, role_id, granted_by, expires_at
) VALUES (
  $1, $2, $3, $4
)
ON CONFLICT (user_id, role_id) DO UPDATE
SET granted_by = $3, granted_at = now(), expires_at = $4;

-- name: RevokeRoleFromUser :exec
DELETE FROM rbac.user_role
WHERE user_id = $1 AND role_id = $2;

-- name: GetUserRoles :many
SELECT r.* FROM rbac.role r
JOIN rbac.user_role ur ON r.id = ur.role_id
WHERE ur.user_id = $1
  AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY r.name;

-- name: GetUsersWithRole :many
SELECT 
  ur.user_id,
  ur.granted_by,
  ur.granted_at,
  ur.expires_at
FROM rbac.user_role ur
WHERE ur.role_id = $1
  AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY ur.granted_at DESC;

-- name: GetUserRoleDetails :many
SELECT 
  r.id as role_id,
  r.name as role_name,
  r.description as role_description,
  ur.granted_by,
  ur.granted_at,
  ur.expires_at
FROM rbac.role r
JOIN rbac.user_role ur ON r.id = ur.role_id
WHERE ur.user_id = $1
  AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY r.name;

-- Authorization Operations

-- name: GetUserPermissions :many
SELECT DISTINCT p.* 
FROM rbac.permission p
JOIN rbac.role_permission rp ON p.id = rp.permission_id
JOIN rbac.user_role ur ON rp.role_id = ur.role_id
WHERE ur.user_id = $1
  AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY p.resource, p.action;

-- name: CheckUserPermission :one
SELECT EXISTS(
  SELECT 1
  FROM rbac.permission p
  JOIN rbac.role_permission rp ON p.id = rp.permission_id
  JOIN rbac.user_role ur ON rp.role_id = ur.role_id
  WHERE ur.user_id = $1
    AND p.name = $2
    AND (ur.expires_at IS NULL OR ur.expires_at > now())
) as has_permission;

-- name: CheckUserPermissionByResourceAction :one
SELECT EXISTS(
  SELECT 1
  FROM rbac.permission p
  JOIN rbac.role_permission rp ON p.id = rp.permission_id
  JOIN rbac.user_role ur ON rp.role_id = ur.role_id
  WHERE ur.user_id = $1
    AND p.resource = $2
    AND p.action = $3
    AND (ur.expires_at IS NULL OR ur.expires_at > now())
) as has_permission;

-- name: RemoveExpiredUserRoles :exec
DELETE FROM rbac.user_role
WHERE expires_at IS NOT NULL AND expires_at <= now();

