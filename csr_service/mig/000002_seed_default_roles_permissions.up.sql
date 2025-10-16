-- Insert default roles
INSERT INTO rbac.role (name, description) VALUES
  ('super_admin', 'Super administrator with full system access'),
  ('gym_owner', 'Gym owner who can manage trainers, classes, and view reports'),
  ('trainer', 'Trainer who can manage their own classes and view customers'),
  ('customer', 'Customer who can book and manage their class bookings'),
  ('guest', 'Guest with read-only access to public information')
ON CONFLICT (name) DO NOTHING;

-- Insert default permissions

-- User Management Permissions
INSERT INTO rbac.permission (name, resource, action, description) VALUES
  ('manage_users', 'user', 'manage', 'Full user management capabilities'),
  ('create_user', 'user', 'create', 'Create new users'),
  ('view_user', 'user', 'read', 'View user information'),
  ('update_user', 'user', 'update', 'Update user information'),
  ('delete_user', 'user', 'delete', 'Delete users')
ON CONFLICT (name) DO NOTHING;

-- Class Management Permissions
INSERT INTO rbac.permission (name, resource, action, description) VALUES
  ('manage_classes', 'class', 'manage', 'Full class management capabilities'),
  ('create_class', 'class', 'create', 'Create new classes'),
  ('view_class', 'class', 'read', 'View class information'),
  ('update_class', 'class', 'update', 'Update class information'),
  ('delete_class', 'class', 'delete', 'Delete classes')
ON CONFLICT (name) DO NOTHING;

-- Booking Permissions
INSERT INTO rbac.permission (name, resource, action, description) VALUES
  ('manage_bookings', 'booking', 'manage', 'Full booking management capabilities'),
  ('book_class', 'booking', 'create', 'Book a class'),
  ('view_booking', 'booking', 'read', 'View booking information'),
  ('cancel_booking', 'booking', 'delete', 'Cancel a booking')
ON CONFLICT (name) DO NOTHING;

-- Report Permissions
INSERT INTO rbac.permission (name, resource, action, description) VALUES
  ('view_reports', 'report', 'read', 'View system reports'),
  ('generate_reports', 'report', 'create', 'Generate new reports'),
  ('manage_reports', 'report', 'manage', 'Full report management capabilities')
ON CONFLICT (name) DO NOTHING;

-- System/RBAC Permissions
INSERT INTO rbac.permission (name, resource, action, description) VALUES
  ('manage_roles', 'role', 'manage', 'Manage roles and role assignments'),
  ('manage_permissions', 'permission', 'manage', 'Manage permissions and permission assignments'),
  ('view_roles', 'role', 'read', 'View roles'),
  ('view_permissions', 'permission', 'read', 'View permissions')
ON CONFLICT (name) DO NOTHING;

-- Assign permissions to super_admin (ALL permissions)
INSERT INTO rbac.role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM rbac.role r
CROSS JOIN rbac.permission p
WHERE r.name = 'super_admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign permissions to gym_owner
INSERT INTO rbac.role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM rbac.role r, rbac.permission p
WHERE r.name = 'gym_owner'
  AND p.name IN (
    'manage_users', 'create_user', 'view_user', 'update_user',
    'manage_classes', 'create_class', 'view_class', 'update_class', 'delete_class',
    'manage_bookings', 'view_booking', 'cancel_booking',
    'view_reports', 'generate_reports', 'manage_reports',
    'view_roles', 'view_permissions'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign permissions to trainer
INSERT INTO rbac.role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM rbac.role r, rbac.permission p
WHERE r.name = 'trainer'
  AND p.name IN (
    'create_class', 'view_class', 'update_class',
    'view_user', 'view_booking'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign permissions to customer
INSERT INTO rbac.role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM rbac.role r, rbac.permission p
WHERE r.name = 'customer'
  AND p.name IN (
    'book_class', 'view_booking', 'cancel_booking', 'view_class'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign permissions to guest
INSERT INTO rbac.role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM rbac.role r, rbac.permission p
WHERE r.name = 'guest'
  AND p.name IN ('view_class')
ON CONFLICT (role_id, permission_id) DO NOTHING;

