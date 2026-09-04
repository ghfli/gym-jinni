package schedule

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	. "github.com/ghfli/gym-jinni/service/gen/go/schedule/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImScheduleServiceServer struct {
	UnimplementedScheduleServiceServer
	db *sql.DB
	q  *Queries
}

func NewImScheduleServiceServer() (*ImScheduleServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImScheduleServiceServer{}, err
	}
	return &ImScheduleServiceServer{
		db: db,
		q:  New(db),
	}, nil
}

func dateTimeToTime(dt *datetime.DateTime) time.Time {
	if dt == nil {
		return time.Time{}
	}
	return time.Date(int(dt.Year), time.Month(dt.Month), int(dt.Day),
		int(dt.Hours), int(dt.Minutes), int(dt.Seconds), 0, time.UTC)
}

func dateTimeToNullTime(dt *datetime.DateTime) sql.NullTime {
	if dt == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: dateTimeToTime(dt), Valid: true}
}

func strToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
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

func NullTimeToDateTime(nt sql.NullTime) *datetime.DateTime {
	if !nt.Valid {
		return nil
	}
	return TimeToDateTime(nt.Time)
}

func scheduleToProto(s ScheduleSchedule) *Schedule {
	return &Schedule{
		Id:             s.ID,
		GymId:          s.GymID.Int32,
		TrainerId:      s.TrainerID.Int32,
		ClassId:        s.ClassID.Int32,
		RecurrenceRule: s.RecurrenceRule.String,
		StartDate:      TimeToDateTime(s.StartDate),
		EndDate:        NullTimeToDateTime(s.EndDate),
		CreatedAt:      TimeToDateTime(s.CreatedAt),
	}
}

func (s *ImScheduleServiceServer) CreateSchedule(ctx context.Context, req *CreateScheduleRequest) (*CreateScheduleResponse, error) {
	log.Printf("CreateSchedule: gym=%v trainer=%v class=%v", req.GetGymId(), req.GetTrainerId(), req.GetClassId())

	if req.GetStartDate() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "start_date is required")
	}

	row, err := s.q.CreateSchedule(ctx, CreateScheduleParams{
		GymID:          sql.NullInt32{Int32: req.GetGymId(), Valid: req.GetGymId() != 0},
		TrainerID:      sql.NullInt32{Int32: req.GetTrainerId(), Valid: req.GetTrainerId() != 0},
		ClassID:        sql.NullInt32{Int32: req.GetClassId(), Valid: req.GetClassId() != 0},
		RecurrenceRule: strToNullString(req.GetRecurrenceRule()),
		StartDate:      dateTimeToTime(req.GetStartDate()),
		EndDate:        dateTimeToNullTime(req.GetEndDate()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create schedule: %v", err)
	}

	return &CreateScheduleResponse{Schedule: scheduleToProto(row)}, nil
}

func (s *ImScheduleServiceServer) GetSchedule(ctx context.Context, req *GetScheduleRequest) (*GetScheduleResponse, error) {
	row, err := s.q.GetSchedule(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "schedule %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get schedule: %v", err)
	}
	return &GetScheduleResponse{Schedule: scheduleToProto(row)}, nil
}

func (s *ImScheduleServiceServer) ListSchedules(ctx context.Context, req *ListSchedulesRequest) (*ListSchedulesResponse, error) {
	var rows []ScheduleSchedule
	var err error

	if req.GetGymId() != 0 {
		rows, err = s.q.ListSchedulesByGym(ctx, sql.NullInt32{Int32: req.GetGymId(), Valid: true})
	} else {
		rows, err = s.q.ListSchedules(ctx)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list schedules: %v", err)
	}

	schedules := make([]*Schedule, len(rows))
	for i, row := range rows {
		schedules[i] = scheduleToProto(row)
	}
	return &ListSchedulesResponse{Schedules: schedules}, nil
}

func (s *ImScheduleServiceServer) UpdateSchedule(ctx context.Context, req *UpdateScheduleRequest) (*UpdateScheduleResponse, error) {
	log.Printf("UpdateSchedule: id=%d", req.GetId())

	_, err := s.q.GetSchedule(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "schedule %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get schedule: %v", err)
	}

	if req.GetStartDate() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "start_date is required")
	}

	row, err := s.q.UpdateSchedule(ctx, UpdateScheduleParams{
		ID:             req.GetId(),
		RecurrenceRule: strToNullString(req.GetRecurrenceRule()),
		StartDate:      dateTimeToTime(req.GetStartDate()),
		EndDate:        dateTimeToNullTime(req.GetEndDate()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update schedule: %v", err)
	}

	return &UpdateScheduleResponse{Schedule: scheduleToProto(row)}, nil
}

func (s *ImScheduleServiceServer) DeleteSchedule(ctx context.Context, req *DeleteScheduleRequest) (*DeleteScheduleResponse, error) {
	log.Printf("DeleteSchedule: id=%d", req.GetId())

	_, err := s.q.GetSchedule(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "schedule %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get schedule: %v", err)
	}

	if err := s.q.DeleteSchedule(ctx, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete schedule: %v", err)
	}

	return &DeleteScheduleResponse{}, nil
}
