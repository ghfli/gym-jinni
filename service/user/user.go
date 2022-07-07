package user

import (
	"context"
	. "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"log"
	"math/rand"
)

type ImUserServiceServer struct {
	UnimplementedUserServiceServer
}

func (s *ImUserServiceServer) CreateUser(ctx context.Context,
	req *CreateUserRequest) (*CreateUserResponse, error) {
	user := req.GetUser()
	log.Println("Got a request to create user with", user)
	user.Id = rand.Uint32()

	log.Println("Responding with", user)
	return &CreateUserResponse{User: user}, nil
}
