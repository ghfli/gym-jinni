package report

import (
	"context"
	"database/sql"
	"github.com/ghfli/gym-jinni/service/middleware"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

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
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open database: %v", err)
		return &ImReportServiceServer{}, err
	}
	return &ImReportServiceServer{db: db}, nil
}

// revenue is a current-state net snapshot, not a settlement cash-flow report.
func (s *ImReportServiceServer) revenue(ctx context.Context, currency string, start, end time.Time) (int64, int32, string, error) {
	currency = strings.ToLower(currency)
	if currency != "" && !currencyPattern.MatchString(currency) {
		return 0, 0, "", status.Error(codes.InvalidArgument, "currency must be three ASCII letters")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT lower(currency),sum(amount),count(*) FROM payment.payment WHERE status='settled' AND ($1='' OR lower(currency)=$1) AND created_at >= $2 AND created_at < $3 GROUP BY lower(currency)`, currency, start, end)
	if err != nil {
		return 0, 0, "", status.Error(codes.Internal, "read settled revenue")
	}
	defer rows.Close()
	var total int64
	var count int32
	var selected string
	groups := 0
	for rows.Next() {
		if groups > 0 {
			return 0, 0, "", status.Error(codes.FailedPrecondition, "specify currency; settled payments contain multiple currencies")
		}
		if err := rows.Scan(&selected, &total, &count); err != nil {
			return 0, 0, "", status.Error(codes.Internal, "read revenue totals")
		}
		groups++
		if !currencyPattern.MatchString(selected) || total < 0 {
			return 0, 0, "", status.Error(codes.FailedPrecondition, "settled payment data requires operator review")
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, "", status.Error(codes.Internal, "read revenue totals")
	}
	if selected == "" {
		selected = currency
	}
	return total, count, selected, nil
}

var currencyPattern = regexp.MustCompile(`^[a-z]{3}$`)

func reportRange(start, end string, required bool) (time.Time, time.Time, error) {
	if start == "" && end == "" && !required {
		return time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), nil
	}
	first, err1 := time.Parse(time.RFC3339Nano, start)
	last, err2 := time.Parse(time.RFC3339Nano, end)
	if err1 != nil || err2 != nil || !first.Before(last) {
		return time.Time{}, time.Time{}, status.Error(codes.InvalidArgument, "provide an ordered RFC3339 start_date and exclusive end_date")
	}
	return first.UTC(), last.UTC(), nil
}

func (s *ImReportServiceServer) GetDashboard(ctx context.Context, req *GetDashboardRequest) (*GetDashboardResponse, error) {
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, 0); err != nil {
		return nil, err
	}
	start, end, _ := reportRange("", "", false)
	total, _, currency, err := s.revenue(ctx, req.GetCurrency(), start, end)
	if err != nil {
		return nil, err
	}
	result := &GetDashboardResponse{TotalRevenue: total, Currency: currency}
	if err := s.db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM "user"."user"),(SELECT count(*) FROM "class"."class"),(SELECT count(*) FROM booking.booking)`).Scan(&result.TotalUsers, &result.TotalClasses, &result.TotalBookings); err != nil {
		return nil, status.Error(codes.Internal, "read dashboard counts")
	}
	return result, nil
}

func (s *ImReportServiceServer) GetRevenueReport(ctx context.Context, req *GetRevenueReportRequest) (*GetRevenueReportResponse, error) {
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, 0); err != nil {
		return nil, err
	}
	start, end, err := reportRange(req.GetStartDate(), req.GetEndDate(), true)
	if err != nil {
		return nil, err
	}
	total, count, currency, err := s.revenue(ctx, req.GetCurrency(), start, end)
	if err != nil {
		return nil, err
	}
	return &GetRevenueReportResponse{TotalRevenue: total, PaymentCount: count, Currency: currency}, nil
}

func (s *ImReportServiceServer) GetAttendanceReport(ctx context.Context, req *GetAttendanceReportRequest) (*GetAttendanceReportResponse, error) {
	if err := middleware.RequireOwnerOrAdmin(ctx, s.db, 0); err != nil {
		return nil, err
	}
	start, end, err := reportRange(req.GetStartDate(), req.GetEndDate(), false)
	if err != nil {
		return nil, err
	}
	result := &GetAttendanceReportResponse{}
	err = s.db.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE b.status='confirmed'),count(*) FILTER(WHERE b.status='cancelled'),count(*) FILTER(WHERE b.status='completed'),count(*) FILTER(WHERE b.status='no_show') FROM booking.booking b JOIN "class"."class" c ON c.id=b.class_id WHERE c.start_time >= $1 AND c.start_time < $2`, start, end).Scan(&result.TotalBookings, &result.Confirmed, &result.Cancelled, &result.Completed, &result.NoShow)
	if err != nil {
		return nil, status.Error(codes.Internal, "read attendance totals")
	}
	return result, nil
}
