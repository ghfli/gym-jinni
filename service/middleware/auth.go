package middleware

import (
	"context"
	"log"
	"strings"

	"github.com/ghfli/gym-jinni/service/auth"
	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserIDContextKey contextKey = "user_id"

func AuthInterceptor(rbacClient rbacv1alpha.RBACServiceClient) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if IsPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		userID, err := extractUserIDFromJWT(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "authentication required: %v", err)
		}

		ctx = context.WithValue(ctx, UserIDContextKey, userID)

		requiredPermission := GetRequiredPermission(info.FullMethod)
		if requiredPermission == "" {
			return handler(ctx, req)
		}

		authResp, err := rbacClient.CheckPermission(ctx, &rbacv1alpha.CheckPermissionRequest{
			UserId:         userID,
			PermissionName: requiredPermission,
		})
		if err != nil {
			log.Printf("AuthInterceptor: error checking permission: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to check permissions: %v", err)
		}

		if !authResp.Authorized {
			return nil, status.Errorf(codes.PermissionDenied, "insufficient permissions: %s", authResp.Reason)
		}

		return handler(ctx, req)
	}
}

func extractUserIDFromJWT(ctx context.Context) (int32, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Errorf(codes.Unauthenticated, "no metadata")
	}

	// grpc-gateway maps HTTP "Authorization" header to "authorization" metadata key
	// Also accept "grpcgateway-authorization" for direct gRPC calls
	var tokenStr string
	for _, key := range []string{"authorization", "grpcgateway-authorization"} {
		vals := md.Get(key)
		if len(vals) > 0 {
			tokenStr = vals[0]
			break
		}
	}
	if tokenStr == "" {
		return 0, status.Errorf(codes.Unauthenticated, "missing authorization header")
	}

	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	tokenStr = strings.TrimPrefix(tokenStr, "bearer ")

	claims, err := auth.ValidateToken(tokenStr)
	if err != nil {
		return 0, err
	}

	return claims.UserID, nil
}

func GetUserIDFromContext(ctx context.Context) (int32, error) {
	userID, ok := ctx.Value(UserIDContextKey).(int32)
	if !ok {
		return 0, status.Errorf(codes.Internal, "user ID not found in context")
	}
	return userID, nil
}

func StreamAuthInterceptor(rbacClient rbacv1alpha.RBACServiceClient) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if IsPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		_, err := extractUserIDFromJWT(ctx)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "authentication required: %v", err)
		}

		return handler(srv, ss)
	}
}
