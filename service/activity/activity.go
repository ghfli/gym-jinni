package activity

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	. "github.com/ghfli/gym-jinni/service/gen/go/activity/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImActivityServiceServer struct {
	UnimplementedActivityServiceServer
	db *sql.DB
	q  *Queries
}

func NewImActivityServiceServer() (*ImActivityServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImActivityServiceServer{}, err
	}
	return &ImActivityServiceServer{
		db: db,
		q:  New(db),
	}, nil
}

func TimeToDateTime(t time.Time) *datetime.DateTime {
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

func activityToProto(a ActivityActivity) *Activity {
	act := &Activity{
		Id:           a.ID,
		UserId:       a.UserID,
		ActivityType: a.ActivityType,
		Title:        a.Title,
		CreatedAt:    TimeToDateTime(a.CreatedAt),
	}
	if a.GymID.Valid {
		act.GymId = a.GymID.Int32
	}
	if a.Description.Valid {
		act.Description = a.Description.String
	}
	if a.DurationMinutes.Valid {
		act.DurationMinutes = a.DurationMinutes.Int32
	}
	if a.Calories.Valid {
		act.Calories = a.Calories.Int32
	}
	if a.DistanceMeters.Valid {
		act.DistanceMeters = a.DistanceMeters.Int32
	}
	return act
}

func (s *ImActivityServiceServer) LogActivity(ctx context.Context, req *LogActivityRequest) (*LogActivityResponse, error) {
	log.Printf("LogActivity: user=%d type=%s title=%s", req.GetUserId(), req.GetActivityType(), req.GetTitle())

	row, err := s.q.CreateActivity(ctx, CreateActivityParams{
		UserID:          req.GetUserId(),
		GymID:           sql.NullInt32{Int32: req.GetGymId(), Valid: req.GetGymId() != 0},
		ActivityType:    req.GetActivityType(),
		Title:           req.GetTitle(),
		Description:     sql.NullString{String: req.GetDescription(), Valid: req.GetDescription() != ""},
		DurationMinutes: sql.NullInt32{Int32: req.GetDurationMinutes(), Valid: req.GetDurationMinutes() != 0},
		Calories:        sql.NullInt32{Int32: req.GetCalories(), Valid: req.GetCalories() != 0},
		DistanceMeters:  sql.NullInt32{Int32: req.GetDistanceMeters(), Valid: req.GetDistanceMeters() != 0},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create activity: %v", err)
	}

	return &LogActivityResponse{Activity: activityToProto(row)}, nil
}

func (s *ImActivityServiceServer) GetActivity(ctx context.Context, req *GetActivityRequest) (*GetActivityResponse, error) {
	row, err := s.q.GetActivity(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "activity %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get activity: %v", err)
	}
	return &GetActivityResponse{Activity: activityToProto(row)}, nil
}

func (s *ImActivityServiceServer) ListActivities(ctx context.Context, req *ListActivitiesRequest) (*ListActivitiesResponse, error) {
	rows, err := s.q.ListUserActivities(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list activities: %v", err)
	}
	activities := make([]*Activity, len(rows))
	for i, row := range rows {
		activities[i] = activityToProto(row)
	}
	return &ListActivitiesResponse{Activities: activities}, nil
}

func (s *ImActivityServiceServer) GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsResponse, error) {
	row, err := s.q.GetUserActivityStats(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get activity stats: %v", err)
	}
	return &GetStatsResponse{
		Stats: &ActivityStats{
			TotalCount:    row.TotalCount,
			TotalDuration: row.TotalDuration.(int64),
			TotalCalories: row.TotalCalories.(int64),
			TotalDistance: row.TotalDistance.(int64),
		},
	}, nil
}
