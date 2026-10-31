package booking

import (
	"context"
	"database/sql"
	"errors"
	"github.com/ghfli/gym-jinni/service/middleware"
	"github.com/jackc/pgconn"
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
	db     *sql.DB
	q      *Queries
	classQ *classdb.Queries
}

func NewImBookingServiceServer() (*ImBookingServiceServer, error) {
	dburl := os.Getenv("DBURL")
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open database: %v", err)
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
	if req.GetClassId() <= 0 || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user and class IDs are required")
	}
	if err := middleware.RequireSelf(ctx, req.GetUserId()); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin booking")
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	cls, err := s.classQ.WithTx(tx).GetClassForUpdate(ctx, req.GetClassId())
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "class not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "lock class")
	}
	dup, err := q.CheckDuplicateBooking(ctx, CheckDuplicateBookingParams{UserID: req.GetUserId(), ClassID: req.GetClassId()})
	if err != nil {
		return nil, status.Error(codes.Internal, "check duplicate")
	}
	if dup {
		return nil, status.Error(codes.AlreadyExists, "user already booked this class")
	}
	count, err := q.CountClassBookings(ctx, req.GetClassId())
	if err != nil {
		return nil, status.Error(codes.Internal, "count bookings")
	}
	if cls.MaxHdcnt.Valid && cls.MaxHdcnt.Int32 > 0 && count >= int64(cls.MaxHdcnt.Int32) {
		return nil, status.Error(codes.ResourceExhausted, "class is full")
	}
	row, err := q.CreateBooking(ctx, CreateBookingParams{UserID: req.GetUserId(), ClassID: req.GetClassId()})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "user already booked this class")
		}
		return nil, status.Error(codes.Internal, "create booking")
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit booking")
	}
	return &CreateBookingResponse{Booking: bookingToProto(row)}, nil
}

func (s *ImBookingServiceServer) CancelBooking(ctx context.Context, req *CancelBookingRequest) (*CancelBookingResponse, error) {
	existing, err := s.q.GetBooking(ctx, req.GetId())
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "booking not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "get booking")
	}
	if err := s.authorizeBooking(ctx, existing); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "begin cancellation")
	}
	defer tx.Rollback()
	if _, err := s.classQ.WithTx(tx).GetClassForUpdate(ctx, existing.ClassID); err != nil {
		return nil, status.Error(codes.Internal, "lock class")
	}
	q := s.q.WithTx(tx)
	existing, err = q.GetBookingForUpdate(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, "lock booking")
	}
	if existing.Status != "confirmed" {
		return nil, status.Error(codes.FailedPrecondition, "only confirmed bookings can be cancelled")
	}
	if err := q.CancelBooking(ctx, req.GetId()); err != nil {
		return nil, status.Error(codes.Internal, "cancel booking")
	}
	row, err := q.GetBooking(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, "get cancelled booking")
	}
	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "commit cancellation")
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
	if err := s.authorizeBooking(ctx, row); err != nil {
		return nil, err
	}
	return &GetBookingResponse{Booking: bookingToProto(row)}, nil
}

func (s *ImBookingServiceServer) ListUserBookings(ctx context.Context, req *ListUserBookingsRequest) (*ListUserBookingsResponse, error) {
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, req.GetUserId()); err != nil {
		return nil, err
	}
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
	if err := s.authorizeClass(ctx, req.GetClassId()); err != nil {
		return nil, err
	}
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

func (s *ImBookingServiceServer) authorizeClass(ctx context.Context, classID int32) error {
	cls, err := s.classQ.GetClass(ctx, classID)
	if err == sql.ErrNoRows {
		return status.Error(codes.NotFound, "class not found")
	}
	if err != nil {
		return status.Error(codes.Internal, "get class scope")
	}
	return middleware.RequireOwnerOrAdmin(ctx, s.db, cls.CreatedBy.Int32)
}
func (s *ImBookingServiceServer) authorizeBooking(ctx context.Context, b BookingBooking) error {
	actor, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}
	if actor == b.UserID {
		return nil
	}
	return s.authorizeClass(ctx, b.ClassID)
}
