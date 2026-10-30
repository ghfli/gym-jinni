INSERT INTO rbac.permission(name, resource, action, description) VALUES
 ('manage_gym','gym','manage','Manage owned gyms'),
 ('view_gym','gym','read','View gym information'),
 ('manage_schedule','schedule','manage','Manage schedules'),
 ('view_schedule','schedule','read','View schedules'),
 ('create_payment','payment','create','Create payment records'),
 ('view_payment','payment','read','View payment records'),
 ('manage_payment','payment','manage','Manage payment records'),
 ('send_notification','notification','create','Send notifications')
ON CONFLICT(name) DO NOTHING;
INSERT INTO rbac.role_permission(role_id,permission_id)
SELECT r.id,p.id FROM rbac.role r CROSS JOIN rbac.permission p
WHERE (r.name='super_admin' AND p.name IN ('manage_gym','view_gym','manage_schedule','view_schedule','create_payment','view_payment','manage_payment','send_notification'))
 OR (r.name='gym_owner' AND p.name IN ('manage_gym','view_gym'))
 OR (r.name='trainer' AND p.name='update_user')
ON CONFLICT DO NOTHING;
