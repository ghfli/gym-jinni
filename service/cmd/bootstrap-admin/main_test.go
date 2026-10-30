package main

import (
	"database/sql"
	"os"
	"testing"
)

func TestBootstrapFirstAdministratorOnly(t *testing.T) {
	dsn := os.Getenv("VALIDATION_DBURL")
	if dsn == "" {
		t.Skip("isolated database required")
	}
	t.Setenv("DBURL", dsn)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	var id int
	if err := db.QueryRow(`INSERT INTO "user"."user"(name,hashed_passwd) VALUES('bootstrap-test','unused') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM rbac.user_role WHERE user_id=$1`, id)
		db.Exec(`DELETE FROM "user"."user" WHERE id=$1`, id)
	})
	if err := bootstrap(2147483647); err == nil {
		t.Fatal("bootstrap accepted missing account")
	}
	if err := bootstrap(id); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id WHERE ur.user_id=$1 AND ur.granted_by=$1 AND r.name='super_admin'`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("administrator grant missing")
	}
	if err := bootstrap(id); err == nil {
		t.Fatal("bootstrap must stop once an administrator exists")
	}
}
