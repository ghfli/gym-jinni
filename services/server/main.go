package main

import (
	"context"
	"flag"
	"fmt"
	userv1 "github.com/ghfli/gym-jinni/services/gen/proto/go/user/v1"
	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"net"
	"net/http"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := run(); err != nil {
		glog.Fatal(err)
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

	server := grpc.NewServer()
	userv1.RegisterUserMgtServiceServer(server, &userMgtServiceServer{})
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
	err := userv1.RegisterUserMgtServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)

	if err != nil {
		return err
	}

	log.Println("gRPC gateway server listening on :8081")
	return http.ListenAndServe(":8081", mux)
}

type userMgtServiceServer struct {
	userv1.UnimplementedUserMgtServiceServer
}

func (s *userMgtServiceServer) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user := req.GetUser()
	log.Println("Got a request to create user with", user)
	user.Id = rand.Uint32()

	log.Println("Responding with", user)
	return &userv1.CreateUserResponse{User: user}, nil
}
