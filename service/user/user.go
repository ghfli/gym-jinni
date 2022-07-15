package user

import (
	"context"
	"database/sql"
	"fmt"
	. "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"log"
	"net/mail"
	"os"
)

type ImUserServiceServer struct {
	UnimplementedUserServiceServer
	db *sql.DB
	q  *Queries
}

func NewImUserServiceServer() (*ImUserServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Println("Failed to open DBURL %s: %w", dburl, err)
		return &ImUserServiceServer{}, err
	}
	return &ImUserServiceServer{
		db: db,
		q:  New(db),
	}, nil
}

func ValidateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (s *ImUserServiceServer) CreateUser(ctx context.Context,
	req *CreateUserRequest) (*CreateUserResponse, error) {
	user := req.GetUser()
	log.Println("Got a request to create user with:", user)

	var res CreateUserResponse
	var arg CreateUserParams

	if email := user.GetEmail(); ValidateEmail(email) {
		arg.Email.String = email
		arg.Email.Valid = true
	} else {
		return &res, fmt.Errorf("Invalid email: %s", email)
	}

	arg.Phone.String = user.GetPhone()
	arg.Phone.Valid = true
	arg.Name = user.GetName()

	userUser, err := s.q.CreateUser(ctx, arg)
	if err != nil {
		log.Println("Failed to CreateUser:", err)
		return &res, err
	}

	log.Println("Responding with", userUser)
	res.User = &User{
		Id:    userUser.ID,
		Email: userUser.Email.String,
		Phone: userUser.Phone.String,
		Name:  userUser.Name,
	}
	return &res, nil
}
