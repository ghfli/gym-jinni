package class

import (
	"context"
	"database/sql"
	"github.com/ghfli/gym-jinni/service/middleware"
	"log"
	"os"
	"strings"
	"time"

	. "github.com/ghfli/gym-jinni/service/gen/go/class/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImClassServiceServer struct {
	UnimplementedClassServiceServer
	db *sql.DB
	q  *Queries
}

func NewImClassServiceServer() (*ImClassServiceServer, error) {
	dburl := os.Getenv("DBURL")
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open database: %v", err)
		return &ImClassServiceServer{}, err
	}
	return &ImClassServiceServer{
		db: db,
		q:  New(db),
	}, nil
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

func dateTimeToTime(dt *datetime.DateTime) time.Time {
	if dt == nil {
		return time.Time{}
	}
	return time.Date(
		int(dt.Year), time.Month(dt.Month), int(dt.Day),
		int(dt.Hours), int(dt.Minutes), int(dt.Seconds),
		int(dt.Nanos), time.UTC,
	)
}

func classToProto(c ClassClass) *Class {
	return &Class{
		Id:          int32(c.ID),
		CreatedBy:   int32(c.CreatedBy.Int32),
		StartTime:   timeToDateTime(c.StartTime),
		EndTime:     timeToDateTime(c.EndTime),
		Description: c.Description,
		MinHdcnt:    int32(c.MinHdcnt.Int32),
		MaxHdcnt:    int32(c.MaxHdcnt.Int32),
	}
}

func (s *ImClassServiceServer) CreateClass(ctx context.Context, req *CreateClassRequest) (*CreateClassResponse, error) {
	c := req.GetClass()
	if err := validateClass(c); err != nil {
		return nil, err
	}
	actor, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if c.GetCreatedBy() != 0 && c.GetCreatedBy() != actor {
		return nil, status.Error(codes.PermissionDenied, "cannot impersonate class creator")
	}
	c.CreatedBy = actor

	row, err := s.q.CreateClass(ctx, CreateClassParams{
		CreatedBy:   sql.NullInt32{Int32: int32(c.GetCreatedBy()), Valid: c.GetCreatedBy() > 0},
		Description: c.GetDescription(),
		StartTime:   dateTimeToTime(c.GetStartTime()),
		EndTime:     dateTimeToTime(c.GetEndTime()),
		MinHdcnt:    sql.NullInt32{Int32: int32(c.GetMinHdcnt()), Valid: true},
		MaxHdcnt:    sql.NullInt32{Int32: int32(c.GetMaxHdcnt()), Valid: true},
	})
	if err != nil {
		log.Println("Failed to CreateClass:", err)
		return nil, status.Errorf(codes.Internal, "create class: %v", err)
	}

	return &CreateClassResponse{Class: classToProto(row)}, nil
}

func (s *ImClassServiceServer) GetClass(ctx context.Context, req *GetClassRequest) (*GetClassResponse, error) {
	log.Println("GetClass request:", req.GetId())

	row, err := s.q.GetClass(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "class %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get class: %v", err)
	}

	return &GetClassResponse{Class: classToProto(row)}, nil
}

func (s *ImClassServiceServer) ListClasses(ctx context.Context, req *ListClassesRequest) (*ListClassesResponse, error) {
	log.Println("ListClasses request")

	rows, err := s.q.ListClass(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list classes: %v", err)
	}

	classes := make([]*Class, len(rows))
	for i, row := range rows {
		classes[i] = classToProto(row)
	}

	return &ListClassesResponse{Classes: classes}, nil
}

func (s *ImClassServiceServer) UpdateClass(ctx context.Context, req *UpdateClassRequest) (*UpdateClassResponse, error) {
	c := req.GetClass()
	if err := validateClass(c); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin class update")
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	existing, err := q.GetClassForUpdate(ctx, c.GetId())
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "class not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "lock class")
	}
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, existing.CreatedBy.Int32); err != nil {
		return nil, err
	}
	if c.GetCreatedBy() != 0 && c.GetCreatedBy() != existing.CreatedBy.Int32 {
		return nil, status.Error(codes.PermissionDenied, "cannot change class creator")
	}
	if c.GetMaxHdcnt() > 0 {
		var count int64
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM booking.booking WHERE class_id=$1 AND status='confirmed'`, c.GetId()).Scan(&count); err != nil {
			return nil, status.Error(codes.Internal, "count bookings")
		}
		if count > int64(c.GetMaxHdcnt()) {
			return nil, status.Error(codes.FailedPrecondition, "capacity below confirmed bookings")
		}
	}

	err = q.UpdateClass(ctx, UpdateClassParams{
		ID:          int32(c.GetId()),
		Description: c.GetDescription(),
		StartTime:   dateTimeToTime(c.GetStartTime()),
		EndTime:     dateTimeToTime(c.GetEndTime()),
		MinHdcnt:    sql.NullInt32{Int32: int32(c.GetMinHdcnt()), Valid: true},
		MaxHdcnt:    sql.NullInt32{Int32: int32(c.GetMaxHdcnt()), Valid: true},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update class: %v", err)
	}

	row, err := q.GetClass(ctx, int32(c.GetId()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get updated class: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit class update")
	}

	return &UpdateClassResponse{Class: classToProto(row)}, nil
}

func (s *ImClassServiceServer) DeleteClass(ctx context.Context, req *DeleteClassRequest) (*DeleteClassResponse, error) {
	log.Println("DeleteClass request:", req.GetId())

	existing, err := s.q.GetClass(ctx, req.GetId())
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "class not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "get class")
	}
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, existing.CreatedBy.Int32); err != nil {
		return nil, err
	}
	err = s.q.DeleteClass(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete class: %v", err)
	}

	return &DeleteClassResponse{Success: true}, nil
}

// A zero maximum means unlimited capacity; finite limits must cover the minimum.
func validateClass(c *Class) error {
	if c == nil || strings.TrimSpace(c.GetDescription()) == "" || c.GetStartTime() == nil || c.GetEndTime() == nil {
		return status.Error(codes.InvalidArgument, "class start and end times are required")
	}
	if !validDateTime(c.GetStartTime()) || !validDateTime(c.GetEndTime()) || !dateTimeToTime(c.GetStartTime()).Before(dateTimeToTime(c.GetEndTime())) {
		return status.Error(codes.InvalidArgument, "invalid class time range (UTC required)")
	}
	if c.GetMinHdcnt() < 0 || c.GetMaxHdcnt() < 0 || (c.GetMaxHdcnt() > 0 && c.GetMinHdcnt() > c.GetMaxHdcnt()) {
		return status.Error(codes.InvalidArgument, "invalid class capacity")
	}
	return nil
}
func validDateTime(dt *datetime.DateTime) bool {
	if dt.GetYear() < 1 || dt.GetYear() > 9999 || dt.GetNanos() < 0 || dt.GetNanos() >= 1e9 {
		return false
	}
	if dt.GetUtcOffset() != nil && dt.GetUtcOffset().AsDuration() != 0 {
		return false
	}
	if dt.GetTimeZone() != nil && dt.GetTimeZone().GetId() != "UTC" {
		return false
	}
	t := dateTimeToTime(dt)
	return int32(t.Year()) == dt.GetYear() && int32(t.Month()) == dt.GetMonth() && int32(t.Day()) == dt.GetDay() && int32(t.Hour()) == dt.GetHours() && int32(t.Minute()) == dt.GetMinutes() && int32(t.Second()) == dt.GetSeconds()
}
