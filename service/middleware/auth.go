package middleware

import (
	"context"
	"fmt"
	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"strconv"
)

// UserContextKey is the key for storing user ID in context
type contextKey string

const UserIDContextKey contextKey = "user_id"

// AuthInterceptor creates a gRPC unary interceptor for authentication and authorization
func AuthInterceptor(rbacClient rbacv1alpha.RBACServiceClient) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		log.Printf("AuthInterceptor: Checking method %s", info.FullMethod)
		
		// Check if method is public
		if IsPublicMethod(info.FullMethod) {
			log.Printf("AuthInterceptor: Method %s is public, skipping auth", info.FullMethod)
			return handler(ctx, req)
		}
		
		// Extract user ID from metadata (in real implementation, this would be from JWT token)
		userID, err := extractUserID(ctx)
		if err != nil {
			log.Printf("AuthInterceptor: Failed to extract user ID: %v", err)
			return nil, status.Errorf(codes.Unauthenticated, "authentication required: %v", err)
		}
		
		// Add user ID to context for downstream handlers
		ctx = context.WithValue(ctx, UserIDContextKey, userID)
		
		// Check if method requires specific permission
		requiredPermission := GetRequiredPermission(info.FullMethod)
		if requiredPermission == "" {
			// Method requires authentication but no specific permission
			log.Printf("AuthInterceptor: Method %s requires auth only, user %d authenticated", info.FullMethod, userID)
			return handler(ctx, req)
		}
		
		// Check if user has required permission
		authResp, err := rbacClient.CheckPermission(ctx, &rbacv1alpha.CheckPermissionRequest{
			UserId:         userID,
			PermissionName: requiredPermission,
		})
		
		if err != nil {
			log.Printf("AuthInterceptor: Error checking permission: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to check permissions: %v", err)
		}
		
		if !authResp.Authorized {
			log.Printf("AuthInterceptor: User %d denied access to %s: %s", userID, info.FullMethod, authResp.Reason)
			return nil, status.Errorf(codes.PermissionDenied, "insufficient permissions: %s", authResp.Reason)
		}
		
		log.Printf("AuthInterceptor: User %d authorized for %s", userID, info.FullMethod)
		return handler(ctx, req)
	}
}

// extractUserID extracts the user ID from gRPC metadata
// In a real implementation, this would validate a JWT token and extract the user ID
// For now, we expect the client to send a "user-id" metadata header
func extractUserID(ctx context.Context) (int32, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, fmt.Errorf("no metadata found in context")
	}
	
	userIDStrs := md.Get("user-id")
	if len(userIDStrs) == 0 {
		return 0, fmt.Errorf("user-id not found in metadata")
	}
	
	userID, err := strconv.ParseInt(userIDStrs[0], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user-id format: %v", err)
	}
	
	if userID <= 0 {
		return 0, fmt.Errorf("invalid user-id value: %d", userID)
	}
	
	return int32(userID), nil
}

// GetUserIDFromContext extracts the user ID from context
// This should be used by service handlers to get the authenticated user ID
func GetUserIDFromContext(ctx context.Context) (int32, error) {
	userID, ok := ctx.Value(UserIDContextKey).(int32)
	if !ok {
		return 0, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// StreamAuthInterceptor creates a gRPC stream interceptor for authentication and authorization
// This is a placeholder for streaming RPC support
func StreamAuthInterceptor(rbacClient rbacv1alpha.RBACServiceClient) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Printf("StreamAuthInterceptor: Checking method %s", info.FullMethod)
		
		// Check if method is public
		if IsPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}
		
		// Extract user ID from metadata
		ctx := ss.Context()
		userID, err := extractUserID(ctx)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "authentication required: %v", err)
		}
		
		// Check if method requires specific permission
		requiredPermission := GetRequiredPermission(info.FullMethod)
		if requiredPermission == "" {
			// Method requires authentication but no specific permission
			return handler(srv, ss)
		}
		
		// Check if user has required permission
		authResp, err := rbacClient.CheckPermission(ctx, &rbacv1alpha.CheckPermissionRequest{
			UserId:         userID,
			PermissionName: requiredPermission,
		})
		
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permissions: %v", err)
		}
		
		if !authResp.Authorized {
			return status.Errorf(codes.PermissionDenied, "insufficient permissions: %s", authResp.Reason)
		}
		
		return handler(srv, ss)
	}
}

