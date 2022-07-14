package user

import (
	"context"
	"database/sql"
	"fmt"
	. "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"log"
	"os"
	"regexp"
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
	if email == "" {
		return false
	}
	re := regexp.MustCompile(
		"^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return re.MatchString(email)
}

func ValidatePhone(phone string) bool {
	if phone == "" {
		return false
	}
	re := regexp.MustCompile(`^(?:(?:\(?(?:00|\+)([1-4]\d\d|[1-9]\d?)\)?)?[\-\.\ \\\/]?)?((?:\(?\d{1,}\)?[\-\.\ \\\/]?){0,})(?:[\-\.\ \\\/]?(?:#|ext\.?|extension|x)[\-\.\ \\\/]?(\d+))?$`)
	return re.MatchString(phone)
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

	if phone := user.GetPhone(); ValidatePhone(phone) {
		arg.Phone.String = phone
		arg.Phone.Valid = true
	} else {
		return &res, fmt.Errorf("Invalid phone: %s", phone)
	}

	if name := user.GetName(); name != "" {
		arg.Name = name
	} else {
		return &res, fmt.Errorf("Invalid name: %s", name)
	}

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
