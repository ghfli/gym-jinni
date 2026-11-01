package integration

import (
	"fmt"
	"testing"
)

func TestSettledRevenueAndAttendanceBoundaries(t *testing.T) {
	s := setup(t)
	admin := s.user()
	s.role(admin, "super_admin")
	for _, row := range []struct {
		amount                 int
		currency, status, date string
	}{
		{100, "usd", "settled", "2041-01-02"}, {200, "cad", "settled", "2041-01-02"},
		{777, "usd", "pending", "2041-01-02"}, {999, "usd", "refunded", "2041-01-02"},
		{300, "usd", "settled", "2041-01-03"},
	} {
		if _, err := s.db.Exec(`INSERT INTO payment.payment(user_id,amount,currency,status,payment_type,created_at) VALUES($1,$2,$3,$4,'fixture',$5::date)`, admin.id, row.amount, row.currency, row.status, row.date); err != nil {
			t.Fatal(err)
		}
	}
	path := "/v1/reports/revenue?start_date=2041-01-01T00:00:00Z&end_date=2041-01-03T00:00:00Z"
	s.expect("GET", path, admin.token, nil, 400)
	totals := s.expect("GET", path+"&currency=USD", admin.token, nil, 200)
	if field(totals, "total_revenue", "totalRevenue") != "100" || field(totals, "payment_count", "paymentCount") != "1" || totals["currency"] != "usd" {
		t.Fatalf("incorrect settled revenue: %v", totals)
	}
	s.expect("GET", "/v1/reports/revenue?start_date=invalid&end_date=invalid", admin.token, nil, 400)
	s.expect("GET", path+"&currency=invalid", admin.token, nil, 400)
	var classID, excludedID int
	for i, id := range []*int{&classID, &excludedID} {
		date := fmt.Sprintf("2041-01-0%d", i+2)
		if err := s.db.QueryRow(`INSERT INTO "class"."class"(created_by,description,start_time,end_time) VALUES($1,'fixture',$2::date,$2::date+interval '1 hour') RETURNING id`, admin.id, date).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	for _, state := range []string{"confirmed", "cancelled", "completed", "no_show"} {
		if _, err := s.db.Exec(`INSERT INTO booking.booking(user_id,class_id,status) VALUES($1,$2,$3)`, admin.id, classID, state); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(`INSERT INTO booking.booking(user_id,class_id,status) VALUES($1,$2,'confirmed')`, admin.id, excludedID); err != nil {
		t.Fatal(err)
	}
	attendance := s.expect("GET", "/v1/reports/attendance?start_date=2041-01-01T00:00:00Z&end_date=2041-01-03T00:00:00Z", admin.token, nil, 200)
	if field(attendance, "total_bookings", "totalBookings") != "4" || field(attendance, "no_show", "noShow") != "1" {
		t.Fatalf("incorrect attendance boundary: %v", attendance)
	}
	s.expect("GET", "/v1/reports/attendance?start_date=2041-01-03T00:00:00Z&end_date=2041-01-01T00:00:00Z", admin.token, nil, 400)
}
