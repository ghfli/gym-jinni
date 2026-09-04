package payment

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	. "github.com/ghfli/gym-jinni/service/gen/go/payment/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImPaymentServiceServer struct {
	UnimplementedPaymentServiceServer
	db *sql.DB
	q  *Queries
}

func NewImPaymentServiceServer() (*ImPaymentServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImPaymentServiceServer{}, err
	}
	return &ImPaymentServiceServer{
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

func paymentToProto(p PaymentPayment) *Payment {
	pay := &Payment{
		Id:          p.ID,
		UserId:      p.UserID,
		Amount:      p.Amount,
		Currency:    p.Currency,
		Status:      p.Status,
		PaymentType: p.PaymentType,
		CreatedAt:   TimeToDateTime(p.CreatedAt),
	}
	if p.GymID.Valid {
		pay.GymId = p.GymID.Int32
	}
	if p.Provider.Valid {
		pay.Provider = p.Provider.String
	}
	if p.ProviderRef.Valid {
		pay.ProviderRef = p.ProviderRef.String
	}
	return pay
}

func (s *ImPaymentServiceServer) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	log.Printf("CreatePayment: user=%d amount=%d currency=%s type=%s", req.GetUserId(), req.GetAmount(), req.GetCurrency(), req.GetPaymentType())

	currency := req.GetCurrency()
	if currency == "" {
		currency = "usd"
	}

	row, err := s.q.CreatePayment(ctx, CreatePaymentParams{
		UserID:      req.GetUserId(),
		GymID:       sql.NullInt32{Int32: req.GetGymId(), Valid: req.GetGymId() != 0},
		Amount:      req.GetAmount(),
		Currency:    currency,
		PaymentType: req.GetPaymentType(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create payment: %v", err)
	}

	return &CreatePaymentResponse{Payment: paymentToProto(row)}, nil
}

func (s *ImPaymentServiceServer) GetPayment(ctx context.Context, req *GetPaymentRequest) (*GetPaymentResponse, error) {
	row, err := s.q.GetPayment(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "payment %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get payment: %v", err)
	}
	return &GetPaymentResponse{Payment: paymentToProto(row)}, nil
}

func (s *ImPaymentServiceServer) ListPayments(ctx context.Context, req *ListPaymentsRequest) (*ListPaymentsResponse, error) {
	rows, err := s.q.ListUserPayments(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list payments: %v", err)
	}
	payments := make([]*Payment, len(rows))
	for i, row := range rows {
		payments[i] = paymentToProto(row)
	}
	return &ListPaymentsResponse{Payments: payments}, nil
}

func (s *ImPaymentServiceServer) RefundPayment(ctx context.Context, req *RefundPaymentRequest) (*RefundPaymentResponse, error) {
	log.Printf("RefundPayment: id=%d", req.GetId())

	existing, err := s.q.GetPayment(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "payment %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "get payment: %v", err)
	}
	if existing.Status == "refunded" {
		return nil, status.Errorf(codes.FailedPrecondition, "payment already refunded")
	}

	row, err := s.q.UpdatePaymentStatus(ctx, UpdatePaymentStatusParams{
		ID:     req.GetId(),
		Status: "refunded",
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "refund payment: %v", err)
	}

	return &RefundPaymentResponse{Payment: paymentToProto(row)}, nil
}
