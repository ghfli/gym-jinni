package integration

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestAdministratorDirectoryAndContinuity(t *testing.T) {
	s := setup(t)
	admin, customer := s.user(), s.user()
	s.expect("GET", "/v1/users?page_size=1", customer.token, nil, 403)
	s.role(admin, "super_admin")
	page := s.expect("GET", "/v1/users?page_size=1", admin.token, nil, 200)
	users, ok := page["users"].([]any)
	if !ok || len(users) != 1 {
		t.Fatal("directory pagination failed")
	}
	if strings.Contains(fmt.Sprint(page), "hashed") || (users[0].(map[string]any)["passwd"] != nil && users[0].(map[string]any)["passwd"] != "") {
		t.Fatal("directory leaked credentials")
	}
	if field(page, "next_offset", "nextOffset") != "1" {
		t.Fatal("missing next offset")
	}
	s.expect("GET", "/v1/users?page_size=201", admin.token, nil, 400)
	s.expect("GET", "/v1/users?offset=-1", admin.token, nil, 400)
	var roleID int
	if err := s.db.QueryRow(`SELECT id FROM rbac.role WHERE name='super_admin'`).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	// Remove the independent fixture administrators while keeping this permanent one.
	rows, err := s.db.Query(`SELECT user_id FROM rbac.user_role WHERE role_id=$1 AND user_id<>$2`, roleID, admin.id)
	if err != nil {
		t.Fatal(err)
	}
	var others []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		others = append(others, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	for _, id := range others {
		s.expect("DELETE", fmt.Sprintf("/v1/users/%d/roles/%d", id, roleID), admin.token, nil, 200)
	}
	s.expect("DELETE", fmt.Sprintf("/v1/users/%d/roles/%d", admin.id, roleID), admin.token, nil, 400)
	s.expect("DELETE", fmt.Sprintf("/v1/users/%d", admin.id), admin.token, nil, 400)
	s.expect("DELETE", fmt.Sprintf("/v1/roles/%d", roleID), admin.token, nil, 400)
	expiry := map[string]int{"year": 2035, "month": 1, "day": 1}
	s.expect("POST", fmt.Sprintf("/v1/users/%d/roles/%d", admin.id, roleID), admin.token, map[string]any{"expires_at": expiry}, 400)
	// A replacement permanent administrator makes revocation legal.
	replacement := s.user()
	s.expect("POST", fmt.Sprintf("/v1/users/%d/roles/%d", replacement.id, roleID), admin.token, map[string]any{}, 200)
	s.expect("DELETE", fmt.Sprintf("/v1/users/%d/roles/%d", admin.id, roleID), replacement.token, nil, 200)
	s.expect("GET", "/v1/users", admin.token, nil, 403)
	s.expect("POST", fmt.Sprintf("/v1/users/%d/roles/%d", customer.id, roleID), replacement.token, map[string]any{"expires_at": map[string]int{"year": 2035, "month": 2, "day": 30}}, 400)
	third := s.user()
	s.expect("POST", fmt.Sprintf("/v1/users/%d/roles/%d", third.id, roleID), replacement.token, map[string]any{}, 200)
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for _, id := range []int{replacement.id, third.id} {
		wg.Add(1)
		go func(target int) {
			defer wg.Done()
			code, _ := s.request("DELETE", fmt.Sprintf("/v1/users/%d/roles/%d", target, roleID), replacement.token, nil)
			codes <- code
		}(id)
	}
	wg.Wait()
	close(codes)
	successes := 0
	for code := range codes {
		if code == 200 {
			successes++
		} else if code != 400 && code != 403 {
			t.Fatalf("unexpected administrator race status %d", code)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one administrator revocation, got %d", successes)
	}
	var permanent int
	if err := s.db.QueryRow(`SELECT count(*) FROM rbac.user_role WHERE role_id=$1 AND expires_at IS NULL`, roleID).Scan(&permanent); err != nil || permanent < 1 {
		t.Fatal("concurrent changes removed permanent administrator", err)
	}

}
