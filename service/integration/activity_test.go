package integration

import "testing"

func TestInvalidActivityHasNoSideEffects(t *testing.T) {
	s := setup(t)
	a := s.user()
	for _, field := range []string{"duration_minutes", "calories", "distance_meters"} {
		s.expect("POST", "/v1/activities", a.token, map[string]any{"user_id": a.id, "activity_type": "workout", "title": "Invalid", field: -1}, 400)
	}
	s.expect("POST", "/v1/activities", a.token, map[string]any{"user_id": a.id, "activity_type": " ", "title": "Invalid"}, 400)
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM activity.activity WHERE user_id=$1`, a.id).Scan(&count); err != nil || count != 0 {
		t.Fatal("invalid activity persisted", err, count)
	}
}
