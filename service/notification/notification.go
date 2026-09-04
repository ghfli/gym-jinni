package notification

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	. "github.com/ghfli/gym-jinni/service/gen/go/notification/v1alpha"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImNotificationServiceServer struct {
	UnimplementedNotificationServiceServer
	db *sql.DB
	q  *Queries
}

func NewImNotificationServiceServer() (*ImNotificationServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImNotificationServiceServer{}, err
	}
	return &ImNotificationServiceServer{
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

func notificationToProto(n NotificationNotification) *Notification {
	notif := &Notification{
		Id:        n.ID,
		UserId:    n.UserID,
		Type:      n.Type,
		Channel:   n.Channel,
		Title:     n.Title,
		IsRead:    n.IsRead,
		CreatedAt: TimeToDateTime(n.CreatedAt),
	}
	if n.Body.Valid {
		notif.Body = n.Body.String
	}
	return notif
}

func preferenceToProto(p NotificationPreference) *Preference {
	return &Preference{
		Id:      p.ID,
		UserId:  p.UserID,
		Channel: p.Channel,
		Enabled: p.Enabled,
	}
}

func (s *ImNotificationServiceServer) SendNotification(ctx context.Context, req *SendNotificationRequest) (*SendNotificationResponse, error) {
	log.Printf("SendNotification: user=%d type=%s channel=%s", req.GetUserId(), req.GetType(), req.GetChannel())

	channel := req.GetChannel()
	if channel == "" {
		channel = "in_app"
	}

	row, err := s.q.CreateNotification(ctx, CreateNotificationParams{
		UserID:  req.GetUserId(),
		Type:    req.GetType(),
		Channel: channel,
		Title:   req.GetTitle(),
		Body:    sql.NullString{String: req.GetBody(), Valid: req.GetBody() != ""},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create notification: %v", err)
	}

	return &SendNotificationResponse{Notification: notificationToProto(row)}, nil
}

func (s *ImNotificationServiceServer) ListNotifications(ctx context.Context, req *ListNotificationsRequest) (*ListNotificationsResponse, error) {
	rows, err := s.q.ListUserNotifications(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list notifications: %v", err)
	}
	notifications := make([]*Notification, len(rows))
	for i, row := range rows {
		notifications[i] = notificationToProto(row)
	}
	return &ListNotificationsResponse{Notifications: notifications}, nil
}

func (s *ImNotificationServiceServer) MarkRead(ctx context.Context, req *MarkReadRequest) (*MarkReadResponse, error) {
	log.Printf("MarkRead: id=%d", req.GetId())
	if err := s.q.MarkNotificationRead(ctx, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "mark read: %v", err)
	}
	return &MarkReadResponse{}, nil
}

func (s *ImNotificationServiceServer) GetPreferences(ctx context.Context, req *GetPreferencesRequest) (*GetPreferencesResponse, error) {
	rows, err := s.q.GetUserPreferences(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get preferences: %v", err)
	}
	prefs := make([]*Preference, len(rows))
	for i, row := range rows {
		prefs[i] = preferenceToProto(row)
	}
	return &GetPreferencesResponse{Preferences: prefs}, nil
}

func (s *ImNotificationServiceServer) UpdatePreference(ctx context.Context, req *UpdatePreferenceRequest) (*UpdatePreferenceResponse, error) {
	log.Printf("UpdatePreference: user=%d channel=%s enabled=%v", req.GetUserId(), req.GetChannel(), req.GetEnabled())

	row, err := s.q.UpsertPreference(ctx, UpsertPreferenceParams{
		UserID:  req.GetUserId(),
		Channel: req.GetChannel(),
		Enabled: req.GetEnabled(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert preference: %v", err)
	}

	return &UpdatePreferenceResponse{Preference: preferenceToProto(row)}, nil
}
