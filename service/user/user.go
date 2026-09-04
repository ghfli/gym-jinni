package user

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/mail"
	"os"
	"time"

	"github.com/ghfli/gym-jinni/service/auth"
	. "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/sqlc-dev/pqtype"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
)

type ImUserServiceServer struct {
	UnimplementedUserServiceServer
	db         *sql.DB
	q          *Queries
	rbacClient rbacv1alpha.RBACServiceClient
}

func (s *ImUserServiceServer) SetRBACClient(client rbacv1alpha.RBACServiceClient) {
	s.rbacClient = client
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

func timeToDateTime(t time.Time) *datetime.DateTime {
	return &datetime.DateTime{
		Year:    int32(t.Year()),
		Month:   int32(t.Month()),
		Day:     int32(t.Day()),
		Hours:   int32(t.Hour()),
		Minutes: int32(t.Minute()),
		Seconds: int32(t.Second()),
		Nanos:   int32(t.Nanosecond()),
	}
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
		return &res, fmt.Errorf("invalid email: %s", email)
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

	// Also assign the default 'customer' role so they can book classes
	if s.rbacClient != nil {
		// First get the customer role ID
		roleResp, err := s.rbacClient.GetRoleByName(ctx, &rbacv1alpha.GetRoleByNameRequest{Name: "customer"})
		if err == nil && roleResp.Role != nil {
			_, err = s.rbacClient.AssignRoleToUser(ctx, &rbacv1alpha.AssignRoleToUserRequest{
				UserId: userUser.ID,
				RoleId: roleResp.Role.Id,
			})
			if err != nil {
				log.Printf("Warning: failed to assign customer role to new user %d: %v", userUser.ID, err)
			}
		} else {
			log.Printf("Warning: failed to find customer role: %v", err)
		}
	} else {
		log.Println("Warning: rbacClient not set, cannot assign customer role")
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

	accessTkn, accessExp, err := auth.GenerateAccessToken(id, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	refreshTkn, refreshExp, err := auth.GenerateRefreshToken(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate refresh token: %v", err)
	}

	sessionID := uuid.New()
	now := time.Now()
	_, err = s.q.CreateSession(ctx, CreateSessionParams{
		ID:         sessionID,
		UserID:     sql.NullInt32{Int32: id, Valid: true},
		RefreshTkn: refreshTkn,
		UserAgent:  "unknown",
		ClientIp:   pqtype.Inet{IPNet: net.IPNet{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(32, 32)}, Valid: true},
		ExpiresAt:  refreshExp,
		CreatedAt:  now,
	})
	if err != nil {
		log.Printf("Failed to create session: %v", err)
	}

	return &LoginUserResponse{
		User: &User{
			Id:    id,
			Email: email.String,
			Phone: phone.String,
			Name:  name,
		},
		SessionId:  sessionID.String(),
		AccessTkn:  accessTkn,
		RefreshTkn: refreshTkn,
		AccessTknExpAt:  timeToDateTime(accessExp),
		RefreshTknExpAt: timeToDateTime(refreshExp),
	}, nil
}

func (s *ImUserServiceServer) RenewAccessToken(ctx context.Context,
	req *RenewAccessTokenRequest) (*RenewAccessTokenResponse, error) {
	claims, err := auth.ValidateToken(req.GetRefreshTkn())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token: %v", err)
	}

	accessTkn, accessExp, err := auth.GenerateAccessToken(claims.UserID, claims.Roles)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	return &RenewAccessTokenResponse{
		AccessTkn:      accessTkn,
		AccessTknExpAt: timeToDateTime(accessExp),
	}, nil
}

func (s *ImUserServiceServer) GetTrainerProfile(ctx context.Context,
	req *GetTrainerProfileRequest) (*GetTrainerProfileResponse, error) {
	tp, err := s.q.GetTrainerProfile(ctx, req.GetUserId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "trainer profile not found for user %d", req.GetUserId())
		}
		return nil, status.Errorf(codes.Internal, "get trainer profile: %v", err)
	}
	return &GetTrainerProfileResponse{Profile: trainerToProto(tp)}, nil
}

func (s *ImUserServiceServer) UpdateTrainerProfile(ctx context.Context,
	req *UpdateTrainerProfileRequest) (*UpdateTrainerProfileResponse, error) {
	p := req.GetProfile()
	if p == nil {
		return nil, status.Errorf(codes.InvalidArgument, "profile is required")
	}

	_, err := s.q.GetTrainerProfile(ctx, p.UserId)
	if err == sql.ErrNoRows {
		tp, err := s.q.CreateTrainerProfile(ctx, CreateTrainerProfileParams{
			UserID:          p.UserId,
			Bio:             sql.NullString{String: p.Bio, Valid: p.Bio != ""},
			Specializations: p.Specializations,
			Certifications:  p.Certifications,
			HourlyRate:      sql.NullInt32{Int32: p.HourlyRate, Valid: p.HourlyRate > 0},
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create trainer profile: %v", err)
		}
		return &UpdateTrainerProfileResponse{Profile: trainerToProto(tp)}, nil
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "get trainer profile: %v", err)
	}

	tp, err := s.q.UpdateTrainerProfile(ctx, UpdateTrainerProfileParams{
		UserID:          p.UserId,
		Bio:             sql.NullString{String: p.Bio, Valid: p.Bio != ""},
		Specializations: p.Specializations,
		Certifications:  p.Certifications,
		HourlyRate:      sql.NullInt32{Int32: p.HourlyRate, Valid: p.HourlyRate > 0},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update trainer profile: %v", err)
	}
	return &UpdateTrainerProfileResponse{Profile: trainerToProto(tp)}, nil
}

func (s *ImUserServiceServer) ListTrainers(ctx context.Context,
	req *ListTrainersRequest) (*ListTrainersResponse, error) {
	rows, err := s.q.ListTrainerProfiles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list trainers: %v", err)
	}
	profiles := make([]*TrainerProfile, len(rows))
	for i, row := range rows {
		profiles[i] = trainerToProto(row)
	}
	return &ListTrainersResponse{Profiles: profiles}, nil
}

func (s *ImUserServiceServer) GetUser(ctx context.Context,
	req *GetUserRequest) (*GetUserResponse, error) {
	userID := int32(0)
	if u := req.GetUser(); u != nil {
		userID = u.GetId()
	}
	if userID == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user id is required")
	}
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %d not found", userID)
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	return &GetUserResponse{
		User: &User{
			Id:    u.ID,
			Email: u.Email.String,
			Phone: u.Phone.String,
			Name:  u.Name,
		},
		CreatedAt:       timeToDateTime(u.CreatedAt),
		EmailVerified:   u.EmailVerified.Bool,
		PhoneVerified:   u.PhoneVerified.Bool,
		PasswdChangedAt: timeToDateTime(u.PasswdChangedAt),
	}, nil
}

func (s *ImUserServiceServer) UpdateUser(ctx context.Context,
	req *UpdateUserRequest) (*UpdateUserResponse, error) {
	u := req.GetUser()
	if u == nil {
		return nil, status.Errorf(codes.InvalidArgument, "user is required")
	}
	if u.GetId() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user id is required")
	}

	existing, err := s.q.GetUser(ctx, u.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %d not found", u.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}

	email := existing.Email.String
	if u.GetEmail() != "" {
		email = u.GetEmail()
	}
	phone := existing.Phone.String
	if u.GetPhone() != "" {
		phone = u.GetPhone()
	}
	name := existing.Name
	if u.GetName() != "" {
		name = u.GetName()
	}
	hashed := existing.HashedPasswd
	if req.GetNewPasswd() != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.GetNewPasswd()), bcrypt.DefaultCost)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "hash password: %v", err)
		}
		hashed = string(h)
	}

	if err := s.q.UpdateUser(ctx, UpdateUserParams{
		ID:           u.GetId(),
		Email:        sql.NullString{String: email, Valid: true},
		Phone:        sql.NullString{String: phone, Valid: true},
		Name:         name,
		HashedPasswd: hashed,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "update user: %v", err)
	}
	return &UpdateUserResponse{
		User: &User{Id: u.GetId(), Email: email, Phone: phone, Name: name},
	}, nil
}

func (s *ImUserServiceServer) DeleteUser(ctx context.Context,
	req *DeleteUserRequest) (*DeleteUserResponse, error) {
	u := req.GetUser()
	if u == nil || u.GetId() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user id is required")
	}
	existing, err := s.q.GetUser(ctx, u.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %d not found", u.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	if err := s.q.DeleteUser(ctx, u.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete user: %v", err)
	}
	return &DeleteUserResponse{
		User: &User{Id: existing.ID, Email: existing.Email.String, Phone: existing.Phone.String, Name: existing.Name},
	}, nil
}

func (s *ImUserServiceServer) LogoutUser(ctx context.Context,
	req *LogoutUserRequest) (*LogoutUserResponse, error) {
	sessionID := req.GetSessionId()
	if sessionID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session_id is required")
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session_id: %v", err)
	}
	session, err := s.q.GetSession(ctx, sid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "get session: %v", err)
	}
	if err := s.q.BlockSession(ctx, sid); err != nil {
		return nil, status.Errorf(codes.Internal, "block session: %v", err)
	}
	return &LogoutUserResponse{
		SessionId: session.ID.String(),
	}, nil
}

func trainerToProto(tp UserTrainerProfile) *TrainerProfile {
	return &TrainerProfile{
		UserId:          tp.UserID,
		Bio:             tp.Bio.String,
		Specializations: tp.Specializations,
		Certifications:  tp.Certifications,
		HourlyRate:      tp.HourlyRate.Int32,
	}
}
