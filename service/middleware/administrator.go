package middleware

import (
	"context"
	"database/sql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LockAdministrators serializes bootstrap and all changes to global administrator grants.
func LockAdministrators(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(187421,1)`); err != nil {
		return status.Error(codes.Internal, "lock administrator grants")
	}
	return nil
}

// ProtectAdministrator preserves a permanent administrator when a grant is removed or expires.
// Call after LockAdministrators and before mutating any target's roles or account.
func ProtectAdministrator(ctx context.Context, tx *sql.Tx, userID int32) error {
	var protected bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='super_admin' AND (ur.expires_at IS NULL OR ur.expires_at>now())) AND NOT EXISTS(SELECT 1 FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id WHERE ur.user_id<>$1 AND r.name='super_admin' AND ur.expires_at IS NULL)`, userID).Scan(&protected)
	if err != nil {
		return status.Error(codes.Internal, "check administrator continuity")
	}
	if protected {
		return status.Error(codes.FailedPrecondition, "keep another permanent super administrator before removing or expiring this grant")
	}
	return nil
}
