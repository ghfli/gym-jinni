package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ghfli/gym-jinni/csr_service/gen/go/rbac/v1alpha"
	"github.com/ghfli/gym-jinni/csr_service/rbac"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/http"
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
		"127.0.0.1:8082", "gRPC server endpoint")
	gatewayPort = flag.String("gateway-port",
		":8083", "HTTP gateway port")
)

func runGRPCServer() error {
	listener, err := net.Listen("tcp", *grpcServerEndpoint)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w",
			*grpcServerEndpoint, err)
	}

	rbacsvc, err := rbac.NewImRBACServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create RBAC service server: %w", err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_validator.UnaryServerInterceptor(),
			// Note: Authorization middleware would be added here in production
			// For now, we run without it to allow testing the RBAC service itself
			// middleware.AuthInterceptor(rbacClient),
		)))
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
	err := rbacv1alpha.RegisterRBACServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)

	if err != nil {
		return err
	}

	log.Println("gRPC gateway server listening on", *gatewayPort)
	return http.ListenAndServe(*gatewayPort, mux)
}

