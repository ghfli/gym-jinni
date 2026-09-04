package class

import (
	"context"
	"database/sql"
	"log"
	"os"
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
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
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
	log.Println("CreateClass request:", c)

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
	log.Println("UpdateClass request:", c)

	err := s.q.UpdateClass(ctx, UpdateClassParams{
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

	row, err := s.q.GetClass(ctx, int32(c.GetId()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get updated class: %v", err)
	}

	return &UpdateClassResponse{Class: classToProto(row)}, nil
}

func (s *ImClassServiceServer) DeleteClass(ctx context.Context, req *DeleteClassRequest) (*DeleteClassResponse, error) {
	log.Println("DeleteClass request:", req.GetId())

	err := s.q.DeleteClass(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete class: %v", err)
	}

	return &DeleteClassResponse{Success: true}, nil
}
