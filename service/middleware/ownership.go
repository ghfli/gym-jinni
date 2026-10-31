package middleware

import (
	"context"
	"database/sql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RequireSelf(ctx context.Context, ownerID int32) error {
	actor, err := GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}
	if actor != ownerID {
		return status.Error(codes.PermissionDenied, "resource belongs to another user")
	}
	return nil
}

// Until tenant-scoped staff policies exist, only the owner or an active system
// administrator may access another user's private resource.
func RequireOwnerOrAdmin(ctx context.Context, db *sql.DB, ownerID int32) error {
	actor, err := GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}
	if actor == ownerID {
		return nil
	}
	var admin bool
	err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='super_admin' AND (ur.expires_at IS NULL OR ur.expires_at>now()))`, actor).Scan(&admin)
	if err != nil {
		return status.Error(codes.Internal, "check resource scope")
	}
	if !admin {
		return status.Error(codes.PermissionDenied, "resource belongs to another user")
	}
	return nil
}
