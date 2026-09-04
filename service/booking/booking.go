package booking

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	. "github.com/ghfli/gym-jinni/service/gen/go/booking/v1alpha"
	classdb "github.com/ghfli/gym-jinni/service/gen/go/class/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImBookingServiceServer struct {
	UnimplementedBookingServiceServer
	db      *sql.DB
	q       *Queries
	classQ  *classdb.Queries
}

func NewImBookingServiceServer() (*ImBookingServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImBookingServiceServer{}, err
	}
	return &ImBookingServiceServer{
		db:     db,
		q:      New(db),
		classQ: classdb.New(db),
	}, nil
}

func StatusFromString(s string) BookingStatus {
	switch s {
	case "confirmed":
		return BookingStatus_BOOKING_STATUS_CONFIRMED
	case "cancelled":
		return BookingStatus_BOOKING_STATUS_CANCELLED
	case "completed":
		return BookingStatus_BOOKING_STATUS_COMPLETED
	case "no_show":
		return BookingStatus_BOOKING_STATUS_NO_SHOW
	default:
		return BookingStatus_BOOKING_STATUS_UNSPECIFIED
	}
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

func bookingToProto(b BookingBooking) *Booking {
	return &Booking{
		Id:          b.ID,
		UserId:      b.UserID,
		ClassId:     b.ClassID,
		Status:      StatusFromString(b.Status),
		BookedAt:    TimeToDateTime(b.BookedAt),
		CancelledAt: NullTimeToDateTime(b.CancelledAt),
	}
}

func (s *ImBookingServiceServer) CreateBooking(ctx context.Context, req *CreateBookingRequest) (*CreateBookingResponse, error) {
	log.Printf("CreateBooking: user=%d class=%d", req.GetUserId(), req.GetClassId())

	dup, err := s.q.CheckDuplicateBooking(ctx, CheckDuplicateBookingParams{
		UserID:  req.GetUserId(),
		ClassID: req.GetClassId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check duplicate: %v", err)
	}
	if dup {
		return nil, status.Errorf(codes.AlreadyExists, "user already booked this class")
	}

	cls, err := s.classQ.GetClass(ctx, req.GetClassId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "class %d not found", req.GetClassId())
		}
		return nil, status.Errorf(codes.Internal, "get class: %v", err)
	}

	if cls.MaxHdcnt.Valid && cls.MaxHdcnt.Int32 > 0 {
		count, err := s.q.CountClassBookings(ctx, req.GetClassId())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "count bookings: %v", err)
		}
		if count >= int64(cls.MaxHdcnt.Int32) {
			return nil, status.Errorf(codes.ResourceExhausted, "class is full (%d/%d)", count, cls.MaxHdcnt.Int32)
		}
	}

	row, err := s.q.CreateBooking(ctx, CreateBookingParams{
		UserID:  req.GetUserId(),
		ClassID: req.GetClassId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create booking: %v", err)
	}

	return &CreateBookingResponse{Booking: bookingToProto(row)}, nil
}

func (s *ImBookingServiceServer) CancelBooking(ctx context.Context, req *CancelBookingRequest) (*CancelBookingResponse, error) {
	log.Printf("CancelBooking: id=%d", req.GetId())

	existing, err := s.q.GetBooking(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "booking %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get booking: %v", err)
	}
	if existing.Status == "cancelled" {
		return nil, status.Errorf(codes.FailedPrecondition, "booking already cancelled")
	}

	if err := s.q.CancelBooking(ctx, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "cancel booking: %v", err)
	}

	row, err := s.q.GetBooking(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get updated booking: %v", err)
	}

	return &CancelBookingResponse{Booking: bookingToProto(row)}, nil
}

func (s *ImBookingServiceServer) GetBooking(ctx context.Context, req *GetBookingRequest) (*GetBookingResponse, error) {
	row, err := s.q.GetBooking(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "booking %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get booking: %v", err)
	}
	return &GetBookingResponse{Booking: bookingToProto(row)}, nil
}

func (s *ImBookingServiceServer) ListUserBookings(ctx context.Context, req *ListUserBookingsRequest) (*ListUserBookingsResponse, error) {
	rows, err := s.q.ListUserBookings(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list user bookings: %v", err)
	}
	bookings := make([]*Booking, len(rows))
	for i, row := range rows {
		bookings[i] = bookingToProto(row)
	}
	return &ListUserBookingsResponse{Bookings: bookings}, nil
}

func (s *ImBookingServiceServer) ListClassBookings(ctx context.Context, req *ListClassBookingsRequest) (*ListClassBookingsResponse, error) {
	rows, err := s.q.ListClassBookings(ctx, req.GetClassId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list class bookings: %v", err)
	}
	bookings := make([]*Booking, len(rows))
	for i, row := range rows {
		bookings[i] = bookingToProto(row)
	}
	return &ListClassBookingsResponse{Bookings: bookings}, nil
}
