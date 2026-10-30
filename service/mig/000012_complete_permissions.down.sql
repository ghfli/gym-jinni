DELETE FROM rbac.role_permission WHERE role_id IN (SELECT id FROM rbac.role WHERE name='trainer') AND permission_id IN (SELECT id FROM rbac.permission WHERE name='update_user');
DELETE FROM rbac.permission WHERE name IN ('manage_gym','view_gym','manage_schedule','view_schedule','create_payment','view_payment','manage_payment','send_notification');
