package gym

import (
	"context"
	"database/sql"
	"log"
	"os"

	. "github.com/ghfli/gym-jinni/service/gen/go/gym/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImGymServiceServer struct {
	UnimplementedGymServiceServer
	db *sql.DB
	q  *Queries
}

func NewImGymServiceServer() (*ImGymServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImGymServiceServer{}, err
	}
	return &ImGymServiceServer{
		db: db,
		q:  New(db),
	}, nil
}

func gymToProto(g GymGym) *Gym {
	return &Gym{
		Id:        g.ID,
		Name:      g.Name,
		Address:   NullStringToString(g.Address),
		Phone:     NullStringToString(g.Phone),
		Email:     NullStringToString(g.Email),
		OwnerId:   NullInt32ToInt32(g.OwnerID),
		CreatedAt: TimeToDateTime(g.CreatedAt),
	}
}

func membershipToProto(m GymMembership) *Membership {
	return &Membership{
		Id:       m.ID,
		UserId:   m.UserID,
		GymId:    m.GymID,
		Role:     m.Role,
		JoinedAt: TimeToDateTime(m.JoinedAt),
	}
}

func (s *ImGymServiceServer) CreateGym(ctx context.Context, req *CreateGymRequest) (*CreateGymResponse, error) {
	log.Printf("CreateGym: name=%s owner_id=%d", req.GetName(), req.GetOwnerId())

	row, err := s.q.CreateGym(ctx, CreateGymParams{
		Name:    req.GetName(),
		Address: sql.NullString{String: req.GetAddress(), Valid: req.GetAddress() != ""},
		Phone:   sql.NullString{String: req.GetPhone(), Valid: req.GetPhone() != ""},
		Email:   sql.NullString{String: req.GetEmail(), Valid: req.GetEmail() != ""},
		OwnerID: sql.NullInt32{Int32: req.GetOwnerId(), Valid: req.GetOwnerId() != 0},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create gym: %v", err)
	}

	return &CreateGymResponse{Gym: gymToProto(row)}, nil
}

func (s *ImGymServiceServer) GetGym(ctx context.Context, req *GetGymRequest) (*GetGymResponse, error) {
	row, err := s.q.GetGym(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "gym %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get gym: %v", err)
	}
	return &GetGymResponse{Gym: gymToProto(row)}, nil
}

func (s *ImGymServiceServer) ListGyms(ctx context.Context, req *ListGymsRequest) (*ListGymsResponse, error) {
	rows, err := s.q.ListGyms(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list gyms: %v", err)
	}
	gyms := make([]*Gym, len(rows))
	for i, row := range rows {
		gyms[i] = gymToProto(row)
	}
	return &ListGymsResponse{Gyms: gyms}, nil
}

func (s *ImGymServiceServer) UpdateGym(ctx context.Context, req *UpdateGymRequest) (*UpdateGymResponse, error) {
	log.Printf("UpdateGym: id=%d name=%s", req.GetId(), req.GetName())

	_, err := s.q.GetGym(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "gym %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get gym: %v", err)
	}

	row, err := s.q.UpdateGym(ctx, UpdateGymParams{
		ID:      req.GetId(),
		Name:    req.GetName(),
		Address: sql.NullString{String: req.GetAddress(), Valid: req.GetAddress() != ""},
		Phone:   sql.NullString{String: req.GetPhone(), Valid: req.GetPhone() != ""},
		Email:   sql.NullString{String: req.GetEmail(), Valid: req.GetEmail() != ""},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update gym: %v", err)
	}

	return &UpdateGymResponse{Gym: gymToProto(row)}, nil
}

func (s *ImGymServiceServer) AddMember(ctx context.Context, req *AddMemberRequest) (*AddMemberResponse, error) {
	log.Printf("AddMember: gym_id=%d user_id=%d role=%s", req.GetGymId(), req.GetUserId(), req.GetRole())

	_, err := s.q.GetGym(ctx, req.GetGymId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "gym %d not found", req.GetGymId())
		}
		return nil, status.Errorf(codes.Internal, "get gym: %v", err)
	}

	role := req.GetRole()
	if role == "" {
		role = "member"
	}

	row, err := s.q.CreateMembership(ctx, CreateMembershipParams{
		UserID: req.GetUserId(),
		GymID:  req.GetGymId(),
		Role:   role,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add member: %v", err)
	}

	return &AddMemberResponse{Membership: membershipToProto(row)}, nil
}

func (s *ImGymServiceServer) ListMembers(ctx context.Context, req *ListMembersRequest) (*ListMembersResponse, error) {
	rows, err := s.q.ListGymMembers(ctx, req.GetGymId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list members: %v", err)
	}
	members := make([]*Membership, len(rows))
	for i, row := range rows {
		members[i] = membershipToProto(row)
	}
	return &ListMembersResponse{Members: members}, nil
}
