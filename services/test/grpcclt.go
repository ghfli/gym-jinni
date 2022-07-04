package main

import (
	"context"
	"fmt"
	usermgtv1 "github.com/ghfli/gym-jinni/services/gen/go/usermgt/v1"
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
		return fmt.Errorf("failed to connect to usermgtv1 on %s: %w", connectTo, err)
	}
	log.Println("Conntected to", connectTo)

	userMgt := usermgtv1.NewUserMgtServiceClient(conn)
	user, err := userMgt.CreateUser(context.Background(),
		&usermgtv1.CreateUserRequest{
			User: &usermgtv1.User{Email: "a@b.com", Name: "abc"},
		})

	if err != nil {
		return fmt.Errorf("failed to CreateUser: %w", err)
	}

	log.Println("Successfully created", user)
	return nil
}
