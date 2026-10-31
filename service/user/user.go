package user

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgconn"
	"log"
	"net"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ghfli/gym-jinni/service/auth"
	. "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"github.com/ghfli/gym-jinni/service/middleware"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/sqlc-dev/pqtype"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/genproto/googleapis/type/datetime"
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
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open database: %v", err)
		return &ImUserServiceServer{}, err
	}
	return &ImUserServiceServer{
		db: db,
		q:  New(db),
	}, nil
}

var phonePattern = regexp.MustCompile(`^[0-9]{3,15}$`)

func ValidateEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
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
	if user == nil {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}

	if strings.TrimSpace(user.GetName()) == "" || !phonePattern.MatchString(user.GetPhone()) || len(user.GetPasswd()) == 0 || len(user.GetPasswd()) > 72 {
		return nil, status.Error(codes.InvalidArgument, "name, phone (3-15 digits), and password (1-72 bytes) are required")
	}
	var res CreateUserResponse
	var arg CreateUserParams

	if email := user.GetEmail(); ValidateEmail(email) {
		arg.Email.String = email
		arg.Email.Valid = true
	} else {
		return &res, status.Error(codes.InvalidArgument, "invalid email")
	}

	arg.Phone.String = user.GetPhone()
	arg.Phone.Valid = true
	arg.Name = user.GetName()

	hash, err := bcrypt.GenerateFromPassword([]byte(user.GetPasswd()), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Failed to hash password:", err)
		return &res, status.Error(codes.Internal, "hash password")
	}
	arg.HashedPasswd = string(hash)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin registration")
	}
	defer tx.Rollback()
	userUser, err := s.q.WithTx(tx).CreateUser(ctx, arg)
	if err != nil {
		return nil, status.Error(codes.Internal, "create user")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO rbac.user_role(user_id,role_id) SELECT $1,id FROM rbac.role WHERE name='customer'`, userUser.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "assign customer role")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return nil, status.Error(codes.Internal, "customer role unavailable")
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit registration")
	}
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
	if u == nil || !ValidateEmail(u.GetEmail()) || u.GetPasswd() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid email")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin login")
	}
	defer tx.Rollback()
	const q = `SELECT id, email, phone, name, hashed_passwd FROM "user"."user" WHERE email = $1 LIMIT 1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, q, u.GetEmail())
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

	sessionID := uuid.New()
	accessTkn, accessExp, err := auth.GenerateAccessToken(id, nil, sessionID.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	refreshTkn, refreshExp, err := auth.GenerateRefreshToken(id, sessionID.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate refresh token: %v", err)
	}

	now := time.Now()
	_, err = s.q.WithTx(tx).CreateSession(ctx, CreateSessionParams{
		ID:             sessionID,
		UserID:         sql.NullInt32{Int32: id, Valid: true},
		RefreshTknHash: auth.RefreshTokenHash(refreshTkn),
		UserAgent:      "unknown",
		ClientIp:       pqtype.Inet{IPNet: net.IPNet{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(32, 32)}, Valid: true},
		ExpiresAt:      refreshExp,
		CreatedAt:      now,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "persist session")
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit login")
	}
	return &LoginUserResponse{
		User: &User{
			Id:    id,
			Email: email.String,
			Phone: phone.String,
			Name:  name,
		},
		SessionId:       sessionID.String(),
		AccessTkn:       accessTkn,
		RefreshTkn:      refreshTkn,
		AccessTknExpAt:  timeToDateTime(accessExp),
		RefreshTknExpAt: timeToDateTime(refreshExp),
	}, nil
}

func (s *ImUserServiceServer) RenewAccessToken(ctx context.Context,
	req *RenewAccessTokenRequest) (*RenewAccessTokenResponse, error) {
	claims, err := auth.ValidateToken(req.GetRefreshTkn(), "refresh")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid refresh token: %v", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin renewal")
	}
	defer tx.Rollback()
	// All authentication mutations lock the user before sessions, including password changes.
	var userID int32
	if err := tx.QueryRowContext(ctx, `SELECT id FROM "user"."user" WHERE id=$1 FOR UPDATE`, claims.UserID).Scan(&userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "invalid session")
		}
		return nil, status.Error(codes.Internal, "check account")
	}
	var hash string
	var blocked bool
	var expires time.Time
	err = tx.QueryRowContext(ctx, `SELECT refresh_tkn_hash, blocked, expires_at FROM "user".session WHERE id=$1 AND user_id=$2 FOR UPDATE`, claims.SessionID, claims.UserID).Scan(&hash, &blocked, &expires)
	if err == sql.ErrNoRows || (err == nil && (blocked || !expires.After(time.Now()))) {
		return nil, status.Error(codes.Unauthenticated, "invalid session")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "check session")
	}
	if hash != auth.RefreshTokenHash(req.GetRefreshTkn()) {
		// A validly signed older credential is a replay; revoke the entire session.
		if _, err := tx.ExecContext(ctx, `UPDATE "user".session SET blocked=true WHERE id=$1`, claims.SessionID); err != nil {
			return nil, status.Error(codes.Internal, "revoke replayed session")
		}
		if err := tx.Commit(); err != nil {
			return nil, status.Error(codes.Internal, "commit revocation")
		}
		return nil, status.Error(codes.Unauthenticated, "refresh credential reused; sign in again")
	}
	access, accessExp, err := auth.GenerateAccessToken(claims.UserID, nil, claims.SessionID)
	if err != nil {
		return nil, status.Error(codes.Internal, "generate access token")
	}
	refresh, refreshExp, err := auth.GenerateRefreshTokenUntil(claims.UserID, claims.SessionID, expires)
	if err != nil {
		return nil, status.Error(codes.Internal, "generate refresh token")
	}
	// Keep the original session expiry: renewal does not create an unlimited lifetime.
	if _, err := tx.ExecContext(ctx, `UPDATE "user".session SET refresh_tkn_hash=$2 WHERE id=$1`, claims.SessionID, auth.RefreshTokenHash(refresh)); err != nil {
		return nil, status.Error(codes.Internal, "rotate refresh credential")
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit renewal")
	}
	return &RenewAccessTokenResponse{AccessTkn: access, AccessTknExpAt: timeToDateTime(accessExp), RefreshTkn: refresh, RefreshTknExpAt: timeToDateTime(refreshExp)}, nil
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

	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, p.UserId); err != nil {
		return nil, err
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
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, userID); err != nil {
		return nil, err
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

	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, u.GetId()); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin profile update")
	}
	defer tx.Rollback()
	var lockedID int32
	if err := tx.QueryRowContext(ctx, `SELECT id FROM "user"."user" WHERE id=$1 FOR UPDATE`, u.GetId()).Scan(&lockedID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "lock account")
	}
	existing, err := s.q.WithTx(tx).GetUser(ctx, u.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %d not found", u.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}

	if (u.GetEmail() != "" && !ValidateEmail(u.GetEmail())) || (u.GetPhone() != "" && !phonePattern.MatchString(u.GetPhone())) || len(req.GetNewPasswd()) > 72 {
		return nil, status.Error(codes.InvalidArgument, "invalid contact or password")
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

	if err := s.q.WithTx(tx).UpdateUser(ctx, UpdateUserParams{
		ID:           u.GetId(),
		Email:        sql.NullString{String: email, Valid: true},
		Phone:        sql.NullString{String: phone, Valid: true},
		Name:         name,
		HashedPasswd: hashed,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "update user: %v", err)
	}
	if req.GetNewPasswd() != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE "user"."user" SET passwd_changed_at=now() WHERE id=$1`, u.GetId()); err != nil {
			return nil, status.Error(codes.Internal, "record password change")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE "user".session SET blocked=true WHERE user_id=$1`, u.GetId()); err != nil {
			return nil, status.Error(codes.Internal, "revoke sessions")
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit profile update")
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
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, u.GetId()); err != nil {
		return nil, err
	}
	existing, err := s.q.GetUser(ctx, u.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "user %d not found", u.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin account deletion")
	}
	defer tx.Rollback()
	for _, query := range []string{`DELETE FROM "user".session WHERE user_id=$1`, `DELETE FROM rbac.user_role WHERE user_id=$1`, `DELETE FROM "user".trainer_profile WHERE user_id=$1`} {
		if _, err := tx.ExecContext(ctx, query, u.GetId()); err != nil {
			return nil, status.Error(codes.Internal, "prepare account deletion")
		}
	}
	if err := s.q.WithTx(tx).DeleteUser(ctx, u.GetId()); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, status.Error(codes.FailedPrecondition, "account has retained business records")
		}
		return nil, status.Error(codes.Internal, "delete account")
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit account deletion")
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
	if err := middleware.RequireSelf(ctx, session.UserID.Int32); err != nil {
		return nil, err
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

// ValidateSession rejects revoked sessions and accounts that no longer exist.
func (s *ImUserServiceServer) ValidateSession(ctx context.Context, claims *auth.Claims) error {
	id, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid session")
	}
	var valid bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM "user".session s JOIN "user"."user" u ON u.id=s.user_id WHERE s.id=$1 AND s.user_id=$2 AND NOT s.blocked AND s.expires_at>now())`, id, claims.UserID).Scan(&valid)
	if err != nil {
		return status.Error(codes.Internal, "check session")
	}
	if !valid {
		return status.Error(codes.Unauthenticated, "session expired or revoked")
	}
	return nil
}
