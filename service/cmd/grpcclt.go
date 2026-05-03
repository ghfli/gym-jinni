package main

import (
	"context"
	"fmt"
	"log"
	"time"

	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
	userv1alpha "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	connectTo := "127.0.0.1:39080"
	conn, err := grpc.Dial(connectTo, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", connectTo, err)
	}
	defer conn.Close()
	log.Println("Connected to", connectTo)

	rbc := rbacv1alpha.NewRBACServiceClient(conn)
	roles, err := rbc.ListRoles(context.Background(), &rbacv1alpha.ListRolesRequest{})
	if err != nil {
		return fmt.Errorf("failed to ListRoles: %w", err)
	}
	log.Printf("ListRoles: %d roles", len(roles.GetRoles()))
	for _, r := range roles.GetRoles() {
		log.Printf("  role: %s", r.GetName())
	}

	usc := userv1alpha.NewUserServiceClient(conn)
	email := fmt.Sprintf("grpcclt%d@example.com", time.Now().UnixNano())
	user, err := usc.CreateUser(context.Background(),
		&userv1alpha.CreateUserRequest{
			User: &userv1alpha.User{Email: email,
				Phone: "6041234567", Name: "grpcclt", Passwd: "abc",
			},
		})
	if err != nil {
		return fmt.Errorf("failed to CreateUser: %w", err)
	}
	log.Println("Successfully created user", user)

	return nil
}
