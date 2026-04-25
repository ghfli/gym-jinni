package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
	userv1alpha "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"github.com/ghfli/gym-jinni/service/rbac"
	"github.com/ghfli/gym-jinni/service/user"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	go runGRPCServer()
	return runGatewayServer()
}

var (
	grpcServerEndpoint = flag.String("grpc-server-endpoint",
		"127.0.0.1:8080", "gRPC server endpoint")
	gatewayAddr = flag.String("gateway-addr",
		":8081", "HTTP gateway listen address")
)

func runGRPCServer() error {
	listener, err := net.Listen("tcp", *grpcServerEndpoint)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w",
			*grpcServerEndpoint, err)
	}

	usersvc, err := user.NewImUserServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create user service server: %w", err)
	}

	rbacsvc, err := rbac.NewImRBACServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create RBAC service server: %w", err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_validator.UnaryServerInterceptor(),
		)))
	userv1alpha.RegisterUserServiceServer(server, usersvc)
	rbacv1alpha.RegisterRBACServiceServer(server, rbacsvc)

	log.Println("gRPC server listening on", *grpcServerEndpoint)
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

func runGatewayServer() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := userv1alpha.RegisterUserServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts); err != nil {
		return err
	}
	if err := rbacv1alpha.RegisterRBACServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts); err != nil {
		return err
	}

	log.Println("gRPC gateway server listening on", *gatewayAddr)
	return http.ListenAndServe(*gatewayAddr, withCORS(mux))
}

// withCORS wraps the gateway so Flutter web (or other browsers) can call the API cross-origin during local dev.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, Grpc-Metadata-user-id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
