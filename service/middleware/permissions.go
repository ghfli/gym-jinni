package middleware

// PermissionMapping maps gRPC method names to required permissions
var PermissionMapping = map[string]string{
	// User service methods (from existing service)
	"/user.v1alpha.UserService/CreateUser":  "create_user",
	"/user.v1alpha.UserService/GetUser":     "view_user",
	"/user.v1alpha.UserService/UpdateUser":  "update_user",
	"/user.v1alpha.UserService/DeleteUser":  "delete_user",
	"/user.v1alpha.UserService/LoginUser":   "", // Public method
	"/user.v1alpha.UserService/LogoutUser":  "", // Authenticated but no specific permission
	
	// Class service methods (from existing service)
	"/class.v1alpha.ClassService/CreateClass": "create_class",
	"/class.v1alpha.ClassService/GetClass":    "view_class",
	"/class.v1alpha.ClassService/ListClass":   "view_class",
	"/class.v1alpha.ClassService/UpdateClass": "update_class",
	"/class.v1alpha.ClassService/DeleteClass": "delete_class",
	
	// RBAC service methods - role management
	"/rbac.v1alpha.RBACService/CreateRole":      "manage_roles",
	"/rbac.v1alpha.RBACService/GetRole":         "view_roles",
	"/rbac.v1alpha.RBACService/GetRoleByName":   "view_roles",
	"/rbac.v1alpha.RBACService/ListRoles":       "view_roles",
	"/rbac.v1alpha.RBACService/UpdateRole":      "manage_roles",
	"/rbac.v1alpha.RBACService/DeleteRole":      "manage_roles",
	
	// RBAC service methods - permission management
	"/rbac.v1alpha.RBACService/CreatePermission":      "manage_permissions",
	"/rbac.v1alpha.RBACService/GetPermission":         "view_permissions",
	"/rbac.v1alpha.RBACService/GetPermissionByName":   "view_permissions",
	"/rbac.v1alpha.RBACService/ListPermissions":       "view_permissions",
	"/rbac.v1alpha.RBACService/UpdatePermission":      "manage_permissions",
	"/rbac.v1alpha.RBACService/DeletePermission":      "manage_permissions",
	
	// RBAC service methods - role-permission assignment
	"/rbac.v1alpha.RBACService/AssignPermissionToRole":   "manage_roles",
	"/rbac.v1alpha.RBACService/RevokePermissionFromRole": "manage_roles",
	"/rbac.v1alpha.RBACService/GetRolePermissions":       "view_roles",
	
	// RBAC service methods - user-role assignment
	"/rbac.v1alpha.RBACService/AssignRoleToUser":   "manage_roles",
	"/rbac.v1alpha.RBACService/RevokeRoleFromUser": "manage_roles",
	"/rbac.v1alpha.RBACService/GetUserRoles":       "view_user",
	"/rbac.v1alpha.RBACService/GetUserRoleDetails": "view_user",
	"/rbac.v1alpha.RBACService/GetUsersWithRole":   "view_roles",
	
	// Booking service methods
	"/booking.v1alpha.BookingService/CreateBooking":     "book_class",
	"/booking.v1alpha.BookingService/CancelBooking":     "cancel_booking",
	"/booking.v1alpha.BookingService/GetBooking":        "view_booking",
	"/booking.v1alpha.BookingService/ListUserBookings":  "view_booking",
	"/booking.v1alpha.BookingService/ListClassBookings": "view_booking",

	// Trainer profile methods
	"/user.v1alpha.UserService/GetTrainerProfile":    "view_user",
	"/user.v1alpha.UserService/UpdateTrainerProfile": "update_user",
	"/user.v1alpha.UserService/ListTrainers":         "view_user",

	// Gym service methods
	"/gym.v1alpha.GymService/CreateGym":    "manage_gym",
	"/gym.v1alpha.GymService/GetGym":       "view_gym",
	"/gym.v1alpha.GymService/ListGyms":     "view_gym",
	"/gym.v1alpha.GymService/UpdateGym":    "manage_gym",
	"/gym.v1alpha.GymService/AddMember":    "manage_gym",
	"/gym.v1alpha.GymService/ListMembers":  "view_gym",

	// Schedule service methods
	"/schedule.v1alpha.ScheduleService/CreateSchedule": "manage_schedule",
	"/schedule.v1alpha.ScheduleService/GetSchedule":    "view_schedule",
	"/schedule.v1alpha.ScheduleService/ListSchedules":  "view_schedule",
	"/schedule.v1alpha.ScheduleService/UpdateSchedule": "manage_schedule",
	"/schedule.v1alpha.ScheduleService/DeleteSchedule": "manage_schedule",

	// Payment service methods
	"/payment.v1alpha.PaymentService/CreatePayment": "create_payment",
	"/payment.v1alpha.PaymentService/GetPayment":    "view_payment",
	"/payment.v1alpha.PaymentService/ListPayments":  "view_payment",
	"/payment.v1alpha.PaymentService/RefundPayment": "manage_payment",

	// Notification service methods
	"/notification.v1alpha.NotificationService/SendNotification":  "send_notification",
	"/notification.v1alpha.NotificationService/ListNotifications": "",
	"/notification.v1alpha.NotificationService/MarkRead":          "",
	"/notification.v1alpha.NotificationService/GetPreferences":    "",
	"/notification.v1alpha.NotificationService/UpdatePreference":  "",

	// Activity service methods
	"/activity.v1alpha.ActivityService/LogActivity":    "",
	"/activity.v1alpha.ActivityService/GetActivity":    "",
	"/activity.v1alpha.ActivityService/ListActivities": "",
	"/activity.v1alpha.ActivityService/GetStats":       "",

	// Report service methods
	"/report.v1alpha.ReportService/GetDashboard":        "view_reports",
	"/report.v1alpha.ReportService/GetRevenueReport":    "view_reports",
	"/report.v1alpha.ReportService/GetAttendanceReport": "view_reports",

	// RBAC service methods - authorization (these should be accessible to check permissions)
	"/rbac.v1alpha.RBACService/CheckPermission":                  "", // Authenticated users can check their own permissions
	"/rbac.v1alpha.RBACService/CheckPermissionByResourceAction":  "", // Authenticated users can check their own permissions
	"/rbac.v1alpha.RBACService/GetUserPermissions":               "", // Authenticated users can get their own permissions
	
	// Internal methods that can be called by other services without auth
}

// PublicMethods lists methods that don't require authentication
var PublicMethods = map[string]bool{
	"/user.v1alpha.UserService/LoginUser":         true,
	"/user.v1alpha.UserService/CreateUser":         true,
	"/user.v1alpha.UserService/RenewAccessToken":   true,
	"/rbac.v1alpha.RBACService/GetRoleByName":      true,
	"/rbac.v1alpha.RBACService/AssignRoleToUser":   true,
}

// IsPublicMethod checks if a method is public
func IsPublicMethod(method string) bool {
	return PublicMethods[method]
}

// GetRequiredPermission returns the permission required for a method
// Returns empty string if no specific permission is required (but authentication may still be needed)
func GetRequiredPermission(method string) string {
	perm, exists := PermissionMapping[method]
	if !exists {
		// Default: require authentication but no specific permission
		return ""
	}
	return perm
}

// RequiresPermission checks if a method requires a specific permission
func RequiresPermission(method string) bool {
	perm := GetRequiredPermission(method)
	return perm != ""
}

