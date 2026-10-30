package middleware

import (
	"context"
	"github.com/ghfli/gym-jinni/service/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestUnknownMethodAndRevokedSessionFailClosed(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret-for-session-validation")
	token, _, err := auth.GenerateAccessToken(7, nil, "session")
	if err != nil {
		t.Fatal(err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	for _, revoked := range []bool{false, true} {
		called := false
		interceptor := AuthInterceptor(nil, func(context.Context, *auth.Claims) error {
			if revoked {
				return status.Error(codes.Unauthenticated, "revoked")
			}
			return nil
		})
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/unknown.Service/Method"}, func(context.Context, interface{}) (interface{}, error) { called = true; return nil, nil })
		want := codes.PermissionDenied
		if revoked {
			want = codes.Unauthenticated
		}
		if status.Code(err) != want || called {
			t.Fatalf("revoked=%t code=%v handler=%t", revoked, status.Code(err), called)
		}
	}
}
