package main

import (
	"context"
	"fmt"
	"github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"google.golang.org/grpc"
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	connectTo := "127.0.0.1:8080"
	conn, err := grpc.Dial(connectTo, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", connectTo, err)
	}
	log.Println("Conntected to", connectTo)

	usc := userv1alpha.NewUserServiceClient(conn)
	user, err := usc.CreateUser(context.Background(),
		&userv1alpha.CreateUserRequest{
			User: &userv1alpha.User{Email: "a@b.com", Name: "abc"},
		})

	if err != nil {
		return fmt.Errorf("failed to CreateUser: %w", err)
	}

	log.Println("Successfully created", user)
	return nil
}
