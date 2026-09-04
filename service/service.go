package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	activityv1alpha "github.com/ghfli/gym-jinni/service/gen/go/activity/v1alpha"
	bookingv1alpha "github.com/ghfli/gym-jinni/service/gen/go/booking/v1alpha"
	classv1alpha "github.com/ghfli/gym-jinni/service/gen/go/class/v1alpha"
	gymv1alpha "github.com/ghfli/gym-jinni/service/gen/go/gym/v1alpha"
	notificationv1alpha "github.com/ghfli/gym-jinni/service/gen/go/notification/v1alpha"
	paymentv1alpha "github.com/ghfli/gym-jinni/service/gen/go/payment/v1alpha"
	rbacv1alpha "github.com/ghfli/gym-jinni/service/gen/go/rbac/v1alpha"
	reportv1alpha "github.com/ghfli/gym-jinni/service/gen/go/report/v1alpha"
	schedulev1alpha "github.com/ghfli/gym-jinni/service/gen/go/schedule/v1alpha"
	userv1alpha "github.com/ghfli/gym-jinni/service/gen/go/user/v1alpha"
	"github.com/ghfli/gym-jinni/service/activity"
	"github.com/ghfli/gym-jinni/service/booking"
	"github.com/ghfli/gym-jinni/service/class"
	"github.com/ghfli/gym-jinni/service/gym"
	"github.com/ghfli/gym-jinni/service/middleware"
	"github.com/ghfli/gym-jinni/service/notification"
	"github.com/ghfli/gym-jinni/service/payment"
	"github.com/ghfli/gym-jinni/service/rbac"
	"github.com/ghfli/gym-jinni/service/report"
	"github.com/ghfli/gym-jinni/service/schedule"
	"github.com/ghfli/gym-jinni/service/user"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	go runGRPCServer()
	return runGatewayServer()
}

var (
	grpcServerEndpoint = flag.String("grpc-server-endpoint",
		"0.0.0.0:8080", "gRPC server endpoint")
	gatewayAddr = flag.String("gateway-addr",
		":8081", "HTTP gateway listen address")
)

func runGRPCServer() error {
	listener, err := net.Listen("tcp", *grpcServerEndpoint)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w",
			*grpcServerEndpoint, err)
	}

	usersvc, err := user.NewImUserServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create user service server: %w", err)
	}

	rbacsvc, err := rbac.NewImRBACServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create RBAC service server: %w", err)
	}

	// Loopback client for RBAC permission checks in the auth interceptor and user service
	rbacConn, err := grpc.Dial("127.0.0.1:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			// Extract authorization from incoming context
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				// Create outgoing context with the same authorization
				outMd := metadata.MD{}
				for _, key := range []string{"authorization", "grpcgateway-authorization", "grpc-metadata-authorization"} {
					if vals := md.Get(key); len(vals) > 0 {
						outMd.Set(key, vals...)
					}
				}
				ctx = metadata.NewOutgoingContext(ctx, outMd)
			}
			return invoker(ctx, method, req, reply, cc, opts...)
		}))
	if err != nil {
		log.Printf("Warning: could not create RBAC loopback client: %v", err)
	}
	// Create a second loopback client without interceptors for internal service calls
	internalRbacConn, err := grpc.Dial("127.0.0.1:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Warning: could not create internal RBAC loopback client: %v", err)
	}

	rbacClient := rbacv1alpha.NewRBACServiceClient(rbacConn)
	internalRbacClient := rbacv1alpha.NewRBACServiceClient(internalRbacConn)
	usersvc.SetRBACClient(internalRbacClient)

	classsvc, err := class.NewImClassServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create class service server: %w", err)
	}

	bookingsvc, err := booking.NewImBookingServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create booking service server: %w", err)
	}

	gymsvc, err := gym.NewImGymServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create gym service server: %w", err)
	}

	schedulesvc, err := schedule.NewImScheduleServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create schedule service server: %w", err)
	}

	paymentsvc, err := payment.NewImPaymentServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create payment service server: %w", err)
	}

	notificationsvc, err := notification.NewImNotificationServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create notification service server: %w", err)
	}

	activitysvc, err := activity.NewImActivityServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create activity service server: %w", err)
	}

	reportsvc, err := report.NewImReportServiceServer()
	if err != nil {
		return fmt.Errorf("failed to create report service server: %w", err)
	}

	// Loopback client for RBAC permission checks in the auth interceptor and user service

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_validator.UnaryServerInterceptor(),
			middleware.AuthInterceptor(rbacClient),
		)))
	userv1alpha.RegisterUserServiceServer(server, usersvc)
	rbacv1alpha.RegisterRBACServiceServer(server, rbacsvc)
	classv1alpha.RegisterClassServiceServer(server, classsvc)
	bookingv1alpha.RegisterBookingServiceServer(server, bookingsvc)
	gymv1alpha.RegisterGymServiceServer(server, gymsvc)
	schedulev1alpha.RegisterScheduleServiceServer(server, schedulesvc)
	paymentv1alpha.RegisterPaymentServiceServer(server, paymentsvc)
	notificationv1alpha.RegisterNotificationServiceServer(server, notificationsvc)
	activityv1alpha.RegisterActivityServiceServer(server, activitysvc)
	reportv1alpha.RegisterReportServiceServer(server, reportsvc)

	log.Println("gRPC server listening on", *grpcServerEndpoint)
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

func runGatewayServer() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	
	// Use 127.0.0.1 for loopback connections since 0.0.0.0 might not work for Dial in all environments
	loopbackEndpoint := "127.0.0.1:8080"
	
	if err := userv1alpha.RegisterUserServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := rbacv1alpha.RegisterRBACServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := classv1alpha.RegisterClassServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := bookingv1alpha.RegisterBookingServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := gymv1alpha.RegisterGymServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := schedulev1alpha.RegisterScheduleServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := paymentv1alpha.RegisterPaymentServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := notificationv1alpha.RegisterNotificationServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := activityv1alpha.RegisterActivityServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}
	if err := reportv1alpha.RegisterReportServiceHandlerFromEndpoint(ctx, mux, loopbackEndpoint, opts); err != nil {
		return err
	}

	log.Println("gRPC gateway server listening on", *gatewayAddr)
	return http.ListenAndServe(*gatewayAddr, withCORS(mux))
}

// withCORS wraps the gateway so Flutter web (or other browsers) can call the API cross-origin during local dev.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, Grpc-Metadata-authorization, Grpc-Metadata-user-id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
