package user

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/mail"
	"os"

	. "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
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

	hash, err := bcrypt.GenerateFromPassword([]byte(user.GetPasswd()), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Failed to hash password:", err)
		return &res, fmt.Errorf("hash password: %w", err)
	}
	arg.HashedPasswd = string(hash)

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

func (s *ImUserServiceServer) LoginUser(ctx context.Context,
	req *LoginUserRequest) (*LoginUserResponse, error) {
	u := req.GetUser()
	if !ValidateEmail(u.GetEmail()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email")
	}

	const q = `SELECT id, email, phone, name, hashed_passwd FROM "user"."user" WHERE email = $1 LIMIT 1`
	row := s.db.QueryRowContext(ctx, q, u.GetEmail())
	var id int32
	var email, phone sql.NullString
	var name, hashed string
	if err := row.Scan(&id, &email, &phone, &name, &hashed); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "login: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(u.GetPasswd())); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	return &LoginUserResponse{
		User: &User{
			Id:    id,
			Email: email.String,
			Phone: phone.String,
			Name:  name,
		},
		SessionId:  uuid.New().String(),
		AccessTkn:  uuid.New().String(),
		RefreshTkn: uuid.New().String(),
	}, nil
}
