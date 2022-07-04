package usermgtv1

import (
	"context"
	usermgtv1 "github.com/ghfli/gym-jinni/services/gen/go/usermgt/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpc "google.golang.org/grpc"
	"log"
	"math/rand"
)

type userMgtServiceServer struct {
	usermgtv1.UnimplementedUserMgtServiceServer
}

func (s *userMgtServiceServer) CreateUser(ctx context.Context,
	req *usermgtv1.CreateUserRequest) (*usermgtv1.CreateUserResponse, error) {
	user := req.GetUser()
	log.Println("Got a request to create user with", user)
	user.Id = rand.Uint32()

	log.Println("Responding with", user)
	return &usermgtv1.CreateUserResponse{User: user}, nil
}

func RegisterUserMgtServiceServer(s grpc.ServiceRegistrar) {
	usermgtv1.RegisterUserMgtServiceServer(s, &userMgtServiceServer{})
}

func RegisterUserMgtServiceHandlerFromEndpoint(ctx context.Context,
	mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return usermgtv1.RegisterUserMgtServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}
