package report

import (
	"context"
	"database/sql"
	"log"
	"os"

	. "github.com/ghfli/gym-jinni/service/gen/go/report/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImReportServiceServer struct {
	UnimplementedReportServiceServer
	db *sql.DB
}

func NewImReportServiceServer() (*ImReportServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImReportServiceServer{}, err
	}
	return &ImReportServiceServer{db: db}, nil
}

func (s *ImReportServiceServer) GetDashboard(ctx context.Context, req *GetDashboardRequest) (*GetDashboardResponse, error) {
	var totalUsers, totalClasses, totalBookings int32
	var totalRevenue int64

	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM "user"."user"`).Scan(&totalUsers); err != nil {
		return nil, status.Errorf(codes.Internal, "count users: %v", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM "class"."class"`).Scan(&totalClasses); err != nil {
		return nil, status.Errorf(codes.Internal, "count classes: %v", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM booking.booking`).Scan(&totalBookings); err != nil {
		return nil, status.Errorf(codes.Internal, "count bookings: %v", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT coalesce(sum(amount),0) FROM payment.payment WHERE status != 'refunded'`).Scan(&totalRevenue); err != nil {
		return nil, status.Errorf(codes.Internal, "sum revenue: %v", err)
	}

	return &GetDashboardResponse{
		TotalUsers:    totalUsers,
		TotalClasses:  totalClasses,
		TotalBookings: totalBookings,
		TotalRevenue:  totalRevenue,
	}, nil
}

func (s *ImReportServiceServer) GetRevenueReport(ctx context.Context, req *GetRevenueReportRequest) (*GetRevenueReportResponse, error) {
	var totalRevenue int64
	var paymentCount int32

	if err := s.db.QueryRowContext(ctx,
		`SELECT coalesce(sum(amount),0), count(*) FROM payment.payment WHERE status != 'refunded' AND created_at >= $1::timestamptz AND created_at <= $2::timestamptz`,
		req.GetStartDate(), req.GetEndDate(),
	).Scan(&totalRevenue, &paymentCount); err != nil {
		return nil, status.Errorf(codes.Internal, "revenue report: %v", err)
	}

	return &GetRevenueReportResponse{
		TotalRevenue: totalRevenue,
		PaymentCount: paymentCount,
	}, nil
}

func (s *ImReportServiceServer) GetAttendanceReport(ctx context.Context, req *GetAttendanceReportRequest) (*GetAttendanceReportResponse, error) {
	var totalBookings, confirmed, cancelled, completed int32

	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM booking.booking`).Scan(&totalBookings); err != nil {
		return nil, status.Errorf(codes.Internal, "count all bookings: %v", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM booking.booking WHERE status = 'confirmed'`).Scan(&confirmed); err != nil {
		return nil, status.Errorf(codes.Internal, "count confirmed: %v", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM booking.booking WHERE status = 'cancelled'`).Scan(&cancelled); err != nil {
		return nil, status.Errorf(codes.Internal, "count cancelled: %v", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM booking.booking WHERE status = 'completed'`).Scan(&completed); err != nil {
		return nil, status.Errorf(codes.Internal, "count completed: %v", err)
	}

	return &GetAttendanceReportResponse{
		TotalBookings: totalBookings,
		Confirmed:     confirmed,
		Cancelled:     cancelled,
		Completed:     completed,
	}, nil
}
