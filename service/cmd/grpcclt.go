package main

import (
	"context"
	"fmt"
	"log"
	"time"

	activityv1alpha "github.com/ghfli/gym-jinni/service/gen/go/activity/v1alpha"
	bookingv1alpha "github.com/ghfli/gym-jinni/service/gen/go/booking/v1alpha"
	classv1alpha "github.com/ghfli/gym-jinni/service/gen/go/class/v1alpha"
	gymv1alpha "github.com/ghfli/gym-jinni/service/gen/go/gym/v1alpha"
	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
	reportv1alpha "github.com/ghfli/gym-jinni/service/gen/go/report/v1alpha"
	userv1alpha "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func authCtx(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

func run() error {
	connectTo := "127.0.0.1:8080"
	conn, err := grpc.Dial(connectTo, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", connectTo, err)
	}
	defer conn.Close()
	log.Println("Connected to", connectTo)

	ctx := context.Background()
	usc := userv1alpha.NewUserServiceClient(conn)

	// --- Create a user (public) ---
	email := fmt.Sprintf("grpcclt%d@example.com", time.Now().UnixNano())
	createResp, err := usc.CreateUser(ctx, &userv1alpha.CreateUserRequest{
		User: &userv1alpha.User{Email: email, Phone: "6041234567", Name: "grpcclt", Passwd: "abc"},
	})
	if err != nil {
		return fmt.Errorf("CreateUser: %w", err)
	}
	userID := createResp.GetUser().GetId()
	log.Printf("CreateUser OK: id=%d email=%s", userID, email)

	// --- Login (public) ---
	loginResp, err := usc.LoginUser(ctx, &userv1alpha.LoginUserRequest{
		User: &userv1alpha.User{Email: email, Passwd: "abc"},
	})
	if err != nil {
		return fmt.Errorf("LoginUser: %w", err)
	}
	token := loginResp.GetAccessTkn()
	sessionID := loginResp.GetSessionId()
	log.Printf("LoginUser OK: session=%s", sessionID)

	authed := authCtx(ctx, token)

	// --- GetUser (authenticated) ---
	getUserResp, err := usc.GetUser(authed, &userv1alpha.GetUserRequest{
		User: &userv1alpha.User{Id: userID},
	})
	if err != nil {
		return fmt.Errorf("GetUser: %w", err)
	}
	log.Printf("GetUser OK: name=%s email=%s", getUserResp.GetUser().GetName(), getUserResp.GetUser().GetEmail())

	// --- UpdateUser ---
	_, err = usc.UpdateUser(authed, &userv1alpha.UpdateUserRequest{
		User: &userv1alpha.User{Id: userID, Name: "grpcclt-updated", Email: email, Phone: "6041234567"},
	})
	if err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	log.Println("UpdateUser OK")

	// --- RenewAccessToken (public, via package-level helper) ---
	renewResp, err := userv1alpha.RenewAccessTokenClient(ctx, conn, &userv1alpha.RenewAccessTokenRequest{
		RefreshTkn: loginResp.GetRefreshTkn(),
	})
	if err != nil {
		return fmt.Errorf("RenewAccessToken: %w", err)
	}
	token = renewResp.GetAccessTkn()
	authed = authCtx(ctx, token)
	log.Println("RenewAccessToken OK")

	// --- RBAC: list roles ---
	rbc := rbacv1alpha.NewRBACServiceClient(conn)
	rolesResp, err := rbc.ListRoles(authed, &rbacv1alpha.ListRolesRequest{})
	if err != nil {
		return fmt.Errorf("ListRoles: %w", err)
	}
	log.Printf("ListRoles OK: %d roles", len(rolesResp.GetRoles()))
	for _, r := range rolesResp.GetRoles() {
		log.Printf("  role: %s", r.GetName())
	}

	// --- Gym: create and list ---
	gymc := gymv1alpha.NewGymServiceClient(conn)
	gymResp, err := gymc.CreateGym(authed, &gymv1alpha.CreateGymRequest{
		Name: "Test Gym", Address: "123 Main St", Phone: "6041234567",
		Email: "gym@example.com", OwnerId: userID,
	})
	if err != nil {
		return fmt.Errorf("CreateGym: %w", err)
	}
	gymID := gymResp.Gym.Id
	log.Printf("CreateGym OK: id=%d", gymID)

	listGymsResp, err := gymc.ListGyms(authed, &gymv1alpha.ListGymsRequest{})
	if err != nil {
		return fmt.Errorf("ListGyms: %w", err)
	}
	log.Printf("ListGyms OK: %d gyms", len(listGymsResp.Gyms))

	// --- Class: create ---
	clsc := classv1alpha.NewClassServiceClient(conn)
	classResp, err := clsc.CreateClass(authed, &classv1alpha.CreateClassRequest{
		Class: &classv1alpha.Class{
			CreatedBy:   uint32(userID),
			Description: "Yoga 101",
			MinHdcnt:    1,
			MaxHdcnt:    20,
		},
	})
	if err != nil {
		return fmt.Errorf("CreateClass: %w", err)
	}
	classID := classResp.GetClass().GetId()
	log.Printf("CreateClass OK: id=%d", classID)

	// --- Booking: create and list ---
	bkc := bookingv1alpha.NewBookingServiceClient(conn)
	bkResp, err := bkc.CreateBooking(authed, &bookingv1alpha.CreateBookingRequest{
		UserId: userID, ClassId: int32(classID),
	})
	if err != nil {
		return fmt.Errorf("CreateBooking: %w", err)
	}
	bookingID := bkResp.Booking.Id
	log.Printf("CreateBooking OK: id=%d", bookingID)

	listBkResp, err := bkc.ListUserBookings(authed, &bookingv1alpha.ListUserBookingsRequest{UserId: userID})
	if err != nil {
		return fmt.Errorf("ListUserBookings: %w", err)
	}
	log.Printf("ListUserBookings OK: %d bookings", len(listBkResp.Bookings))

	// --- Activity: log and get stats ---
	actc := activityv1alpha.NewActivityServiceClient(conn)
	_, err = actc.LogActivity(authed, &activityv1alpha.LogActivityRequest{
		UserId:          userID,
		GymId:           gymID,
		ActivityType:    "workout",
		Title:           "Morning run",
		DurationMinutes: 30,
		Calories:        200,
	})
	if err != nil {
		return fmt.Errorf("LogActivity: %w", err)
	}
	log.Println("LogActivity OK")

	statsResp, err := actc.GetStats(authed, &activityv1alpha.GetStatsRequest{UserId: userID})
	if err != nil {
		return fmt.Errorf("GetStats: %w", err)
	}
	if statsResp.Stats != nil {
		log.Printf("GetStats OK: total_count=%d total_calories=%d", statsResp.Stats.TotalCount, statsResp.Stats.TotalCalories)
	} else {
		log.Println("GetStats OK")
	}

	// --- Report: dashboard ---
	rptc := reportv1alpha.NewReportServiceClient(conn)
	dashResp, err := rptc.GetDashboard(authed, &reportv1alpha.GetDashboardRequest{})
	if err != nil {
		return fmt.Errorf("GetDashboard: %w", err)
	}
	log.Printf("GetDashboard OK: users=%d classes=%d bookings=%d revenue=%d",
		dashResp.TotalUsers, dashResp.TotalClasses, dashResp.TotalBookings, dashResp.TotalRevenue)

	// --- Logout ---
	_, err = usc.LogoutUser(authed, &userv1alpha.LogoutUserRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return fmt.Errorf("LogoutUser: %w", err)
	}
	log.Println("LogoutUser OK")

	log.Println("All tests passed.")
	return nil
}
