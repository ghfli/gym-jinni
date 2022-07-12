package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"github.com/ghfli/gym-jinni/service/user"
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
		"127.0.0.1:8080", "gRPC server endpoint")
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

	server := grpc.NewServer()
	userv1alpha.RegisterUserServiceServer(server, usersvc)
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
	err := userv1alpha.RegisterUserServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)

	if err != nil {
		return err
	}

	log.Println("gRPC gateway server listening on :8081")
	return http.ListenAndServe(":8081", mux)
}
