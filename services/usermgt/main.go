package usermgt

import (
	"context"
	userv1 "github.com/ghfli/gym-jinni/services/gen/proto/go/user/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpc "google.golang.org/grpc"
	"log"
	"math/rand"
)

type userMgtServiceServer struct {
	userv1.UnimplementedUserMgtServiceServer
}

func (s *userMgtServiceServer) CreateUser(ctx context.Context,
	req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user := req.GetUser()
	log.Println("Got a request to create user with", user)
	user.Id = rand.Uint32()

	log.Println("Responding with", user)
	return &userv1.CreateUserResponse{User: user}, nil
}

func RegisterUserMgtServiceServer(s grpc.ServiceRegistrar) {
	userv1.RegisterUserMgtServiceServer(s, &userMgtServiceServer{})
}

func RegisterUserMgtServiceHandlerFromEndpoint(ctx context.Context,
	mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return userv1.RegisterUserMgtServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}
