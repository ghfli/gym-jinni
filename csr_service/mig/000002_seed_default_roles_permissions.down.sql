-- Remove all role-permission assignments
DELETE FROM rbac.role_permission;

-- Remove default permissions
DELETE FROM rbac.permission WHERE name IN (
  'manage_users', 'create_user', 'view_user', 'update_user', 'delete_user',
  'manage_classes', 'create_class', 'view_class', 'update_class', 'delete_class',
  'manage_bookings', 'book_class', 'view_booking', 'cancel_booking',
  'view_reports', 'generate_reports', 'manage_reports',
  'manage_roles', 'manage_permissions', 'view_roles', 'view_permissions'
);

-- Remove default roles
DELETE FROM rbac.role WHERE name IN (
  'super_admin', 'gym_owner', 'trainer', 'customer', 'guest'
);

