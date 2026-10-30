// Package integration validates requirements against an explicitly isolated stack.
package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ghfli/gym-jinni/service/auth"
	_ "github.com/jackc/pgx/v4/stdlib"
	"golang.org/x/crypto/bcrypt"
)

type suite struct {
	t      *testing.T
	db     *sql.DB
	base   string
	client *http.Client
}
type account struct {
	id                      int
	token, refresh, session string
}

func setup(t *testing.T) *suite {
	t.Helper()
	base, dsn := os.Getenv("VALIDATION_HTTP_BASE"), os.Getenv("VALIDATION_DBURL")
	if base == "" || dsn == "" {
		t.Skip("use scripts/validate-business.sh (requires disposable database)")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = db.Ping(); err != nil {
		t.Fatal(err)
	}
	return &suite{t, db, base, &http.Client{Timeout: 15 * time.Second}}
}
func (s *suite) request(method, path, token string, payload any) (int, map[string]any) {
	s.t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		s.t.Fatal(err)
	}
	req, err := http.NewRequest(method, s.base+path, bytes.NewReader(b))
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := s.client.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.t.Fatal(err)
	}
	var out map[string]any
	if err = json.Unmarshal(body, &out); err != nil {
		s.t.Fatalf("%s %s: non-JSON status %d", method, path, res.StatusCode)
	}
	return res.StatusCode, out
}
func (s *suite) expect(method, path, token string, payload any, want int) map[string]any {
	s.t.Helper()
	code, body := s.request(method, path, token, payload)
	if code != want {
		s.t.Fatalf("%s %s: status %d, want %d (grpc code %v)", method, path, code, want, body["code"])
	}
	return body
}
func field(m map[string]any, snake, camel string) string {
	if v, ok := m[snake]; ok {
		return fmt.Sprint(v)
	}
	return fmt.Sprint(m[camel])
}
func (s *suite) user() account {
	s.t.Helper()
	email := fmt.Sprintf("validation-%d@example.com", time.Now().UnixNano())
	u := map[string]any{"email": email, "phone": "1234567890", "name": "Validation", "passwd": "validation-password"}
	body := s.expect("POST", "/v1/create_user", "", map[string]any{"user": u}, 200)
	public := body["user"].(map[string]any)
	if public["passwd"] != nil && public["passwd"] != "" {
		s.t.Fatal("USER-02: password returned")
	}
	id := int(public["id"].(float64))
	var hash string
	if err := s.db.QueryRow(`SELECT hashed_passwd FROM "user"."user" WHERE id=$1`, id).Scan(&hash); err != nil {
		s.t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("validation-password")) != nil {
		s.t.Fatal("USER-01: invalid password hash")
	}
	body = s.expect("POST", "/v1/login_user", "", map[string]any{"user": u}, 200)
	return account{id, field(body, "access_tkn", "accessTkn"), field(body, "refresh_tkn", "refreshTkn"), field(body, "session_id", "sessionId")}
}
func (s *suite) role(a account, name string) {
	s.t.Helper()
	_, err := s.db.Exec(`INSERT INTO rbac.user_role(user_id,role_id) SELECT $1,id FROM rbac.role WHERE name=$2 ON CONFLICT DO NOTHING`, a.id, name)
	if err != nil {
		s.t.Fatal(err)
	}
}
func (s *suite) class(a account, capacity int) int {
	s.t.Helper()
	body := s.expect("POST", "/v1/classes", a.token, map[string]any{"class": map[string]any{"created_by": a.id, "description": "Validation class", "start_time": map[string]int{"year": 2030, "month": 1, "day": 1, "hours": 10}, "end_time": map[string]int{"year": 2030, "month": 1, "day": 1, "hours": 11}, "max_hdcnt": capacity}}, 200)
	return int(body["class"].(map[string]any)["id"].(float64))
}
func TestSeedPermissions(t *testing.T) {
	s := setup(t)
	for _, name := range []string{"manage_gym", "view_gym", "manage_schedule", "view_schedule", "create_payment", "view_payment", "manage_payment", "send_notification"} {
		t.Run(name, func(t *testing.T) {
			var n int
			if err := s.db.QueryRow(`SELECT count(*) FROM rbac.permission WHERE name=$1`, name).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != 1 {
				t.Errorf("RBAC-07: permission %s missing from migrations", name)
			}
		})
	}
}
func TestCustomerJourney(t *testing.T) {
	s := setup(t)
	trainer := s.user()
	s.role(trainer, "trainer")
	customer := s.user()
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='customer'`, customer.id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("registration did not assign customer role")
	}
	c := s.class(trainer, 1)
	s.expect("GET", "/v1/classes", customer.token, nil, 200)
	payload := map[string]int{"user_id": customer.id, "class_id": c}
	b := s.expect("POST", "/v1/bookings", customer.token, payload, 200)["booking"].(map[string]any)
	id := int(b["id"].(float64))
	s.expect("POST", "/v1/bookings", customer.token, payload, 409)
	s.expect("POST", fmt.Sprintf("/v1/bookings/%d/cancel", id), customer.token, nil, 200)
	var status string
	if err := s.db.QueryRow(`SELECT status FROM booking.booking WHERE id=$1`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" {
		t.Fatal("cancellation did not persist")
	}
	s.expect("POST", "/v1/bookings", customer.token, payload, 200)
	other := s.user()
	s.expect("POST", "/v1/bookings", other.token, map[string]int{"user_id": other.id, "class_id": c}, 429)
	s.expect("POST", "/v1/classes", customer.token, map[string]any{"class": map[string]string{"description": "Forbidden"}}, 403)
	s.expect("GET", "/v1/bookings/2147483647", customer.token, nil, 404)
}
func TestActivityTotals(t *testing.T) {
	s := setup(t)
	a := s.user()
	for i := 0; i < 2; i++ {
		s.expect("POST", "/v1/activities", a.token, map[string]any{"user_id": a.id, "activity_type": "workout", "title": "Validation", "duration_minutes": 20, "calories": 100}, 200)
	}
	body := s.expect("GET", fmt.Sprintf("/v1/users/%d/activity_stats", a.id), a.token, nil, 200)
	stats := body["stats"].(map[string]any)
	for _, f := range []struct{ snake, camel, want string }{{"total_count", "totalCount", "2"}, {"total_duration", "totalDuration", "40"}, {"total_calories", "totalCalories", "200"}} {
		if field(stats, f.snake, f.camel) != f.want {
			t.Errorf("activity %s incorrect", f.snake)
		}
	}
}
func TestAuthorizationRequirements(t *testing.T) {
	s := setup(t)
	t.Run("unauthenticated", func(t *testing.T) {
		s := setup(t)
		s.expect("GET", "/v1/classes", "", nil, 401)
		s.expect("GET", "/v1/classes", "forged", nil, 401)
	})
	t.Run("public_role_assignment", func(t *testing.T) {
		s := setup(t)
		a := s.user()
		var role int
		if err := s.db.QueryRow(`SELECT id FROM rbac.role WHERE name='super_admin'`).Scan(&role); err != nil {
			t.Fatal(err)
		}
		s.expect("POST", fmt.Sprintf("/v1/users/%d/roles/%d", a.id, role), "", map[string]any{}, 401)
	})
	t.Run("cross_user_activity", func(t *testing.T) {
		s := setup(t)
		a, b := s.user(), s.user()
		s.expect("GET", fmt.Sprintf("/v1/users/%d/activities", b.id), a.token, nil, 403)
	})
	t.Run("cross_user_booking", func(t *testing.T) {
		s := setup(t)
		a, b := s.user(), s.user()
		s.expect("GET", fmt.Sprintf("/v1/users/%d/bookings", b.id), a.token, nil, 403)
	})
	t.Run("forged_creator", func(t *testing.T) {
		s := setup(t)
		a, b := s.user(), s.user()
		s.role(a, "trainer")
		s.expect("POST", "/v1/classes", a.token, map[string]any{"class": map[string]any{"created_by": b.id, "description": "Spoofed", "start_time": map[string]int{"year": 2030, "month": 1, "day": 1}, "end_time": map[string]int{"year": 2030, "month": 1, "day": 2}}}, 403)
	})
	t.Run("forged_booking_user", func(t *testing.T) {
		s := setup(t)
		a, b := s.user(), s.user()
		s.role(a, "trainer")
		c := s.class(a, 10)
		s.expect("POST", "/v1/bookings", a.token, map[string]int{"user_id": b.id, "class_id": c}, 403)
	})
	t.Run("unmapped_class_list", func(t *testing.T) {
		s := setup(t)
		a := s.user()
		if _, err := s.db.Exec(`DELETE FROM rbac.user_role WHERE user_id=$1`, a.id); err != nil {
			t.Fatal(err)
		}
		s.expect("GET", "/v1/classes", a.token, nil, 403)
	})
	// Verify expiry/revocation independently of the misnamed list policy.
	a := s.user()
	s.role(a, "trainer")
	s.expect("GET", "/v1/classes/2147483647", a.token, nil, 404)
	if _, err := s.db.Exec(`UPDATE rbac.user_role SET expires_at=now()-interval '1 second' WHERE user_id=$1`, a.id); err != nil {
		t.Fatal(err)
	}
	s.expect("GET", "/v1/classes/2147483647", a.token, nil, 403)
	if _, err := s.db.Exec(`DELETE FROM rbac.user_role WHERE user_id=$1`, a.id); err != nil {
		t.Fatal(err)
	}
	s.expect("GET", "/v1/classes/2147483647", a.token, nil, 403)
}
func TestSessionRequirements(t *testing.T) {
	for _, test := range []string{"logout_renewal", "access_as_refresh", "refresh_as_access", "expired_access"} {
		t.Run(test, func(t *testing.T) {
			s := setup(t)
			a := s.user()
			switch test {
			case "logout_renewal":
				s.expect("POST", "/v1/logout_user", a.token, map[string]string{"session_id": a.session}, 200)
				s.expect("POST", "/v1/renew_token", "", map[string]string{"refresh_tkn": a.refresh}, 401)
			case "access_as_refresh":
				s.expect("POST", "/v1/renew_token", "", map[string]string{"refresh_tkn": a.token}, 401)
			case "refresh_as_access":
				s.expect("GET", "/v1/classes", a.refresh, nil, 401)
			case "expired_access":
				secret := os.Getenv("VALIDATION_JWT_SECRET")
				if secret == "" {
					t.Skip("test signing secret unavailable")
				}
				t.Setenv("JWT_SECRET", secret)
				previous := auth.AccessTokenDuration
				auth.AccessTokenDuration = -time.Minute
				token, _, err := auth.GenerateAccessToken(int32(a.id), nil)
				auth.AccessTokenDuration = previous
				if err != nil {
					t.Fatal(err)
				}
				s.expect("GET", "/v1/classes", token, nil, 401)
			}
		})
	}
}
func TestInvalidClassRange(t *testing.T) {
	s := setup(t)
	a := s.user()
	s.role(a, "trainer")
	s.expect("POST", "/v1/classes", a.token, map[string]any{"class": map[string]any{"created_by": a.id, "description": "Invalid range", "start_time": map[string]int{"year": 2030, "month": 1, "day": 2}, "end_time": map[string]int{"year": 2030, "month": 1, "day": 1}, "min_hdcnt": 5, "max_hdcnt": 1}}, 400)
}
func TestConcurrentBookings(t *testing.T) {
	for _, same := range []bool{false, true} {
		t.Run(fmt.Sprintf("same_user_%t", same), func(t *testing.T) {
			s := setup(t)
			a := s.user()
			s.role(a, "trainer")
			capacity := 1
			if same {
				capacity = 100
			}
			c := s.class(a, capacity)
			const contenders = 16
			users := make([]account, contenders)
			for i := range users {
				if same {
					users[i] = a
				} else {
					users[i] = s.user()
				}
			}
			start := make(chan struct{})
			results := make(chan int, contenders)
			var wg sync.WaitGroup
			for _, u := range users {
				wg.Add(1)
				go func(u account) {
					defer wg.Done()
					<-start
					body := strings.NewReader(fmt.Sprintf(`{"user_id":%d,"class_id":%d}`, u.id, c))
					req, err := http.NewRequest("POST", s.base+"/v1/bookings", body)
					if err != nil {
						results <- 0
						return
					}
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", "Bearer "+u.token)
					res, err := s.client.Do(req)
					if err != nil {
						results <- 0
						return
					}
					io.Copy(io.Discard, res.Body)
					res.Body.Close()
					results <- res.StatusCode
				}(u)
			}
			close(start)
			wg.Wait()
			close(results)
			successes := 0
			for code := range results {
				if code == 200 {
					successes++
				} else if code != 409 && code != 429 {
					t.Errorf("unexpected contender status %d", code)
				}
			}
			var count int
			if err := s.db.QueryRow(`SELECT count(*) FROM booking.booking WHERE class_id=$1 AND status='confirmed'`, c).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if successes != 1 || count != 1 {
				t.Errorf("BOOK-01/02: successes=%d confirmed=%d, want exactly one", successes, count)
			}
		})
	}
}
