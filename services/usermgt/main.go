package usermgt

import (
	"context"
	. "github.com/ghfli/gym-jinni/services/gen/go/usermgt/v1"
	"log"
	"math/rand"
)

type ImUserMgtServiceServer struct {
	UnimplementedUserMgtServiceServer
}

func (s *ImUserMgtServiceServer) CreateUser(ctx context.Context,
	req *CreateUserRequest) (*CreateUserResponse, error) {
	user := req.GetUser()
	log.Println("Got a request to create user with", user)
	user.Id = rand.Uint32()

	log.Println("Responding with", user)
	return &CreateUserResponse{User: user}, nil
}
