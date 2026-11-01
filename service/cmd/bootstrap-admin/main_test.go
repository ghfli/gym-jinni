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

func TestParseKV(t *testing.T) {
	m, err := parseKV("# c\nADMIN_EMAIL=a@b.co\n\nADMIN_PASSWORD = \"p w=1\"\n")
	if err != nil || m["ADMIN_EMAIL"] != "a@b.co" || m["ADMIN_PASSWORD"] != "p w=1" {
		t.Fatalf("got %v %v", m, err)
	}
	if _, err := parseKV("junk"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfigFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string, mode os.FileMode) string {
		p := dir + "/" + name
		if err := os.WriteFile(p, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, k := range []string{"ADMIN_EMAIL", "ADMIN_PHONE", "ADMIN_NAME", "ADMIN_PASSWORD", "ADMIN_PASSWORD_FILE", "ADMIN_CONFIG_FILE"} {
		t.Setenv(k, "")
	}
	conf := write("admin.conf", "ADMIN_EMAIL=root@example.com\nADMIN_PHONE=5551234\nADMIN_PASSWORD=from-file-pass\n", 0o600)
	t.Setenv("ADMIN_CONFIG_FILE", conf)
	cfg, err := loadConfig()
	if err != nil || cfg.Email != "root@example.com" || cfg.Password != "from-file-pass" || cfg.Name != "Super Admin" {
		t.Fatalf("config file: %+v %v", cfg, err)
	}
	t.Setenv("ADMIN_PASSWORD_FILE", write("pw", "secret-from-pwfile\n", 0o600))
	if cfg, err = loadConfig(); err != nil || cfg.Password != "secret-from-pwfile" {
		t.Fatalf("password file: %+v %v", cfg, err)
	}
	t.Setenv("ADMIN_PASSWORD", "env-wins-password")
	if cfg, _ = loadConfig(); cfg.Password != "env-wins-password" {
		t.Fatalf("env precedence: %+v", cfg)
	}
	t.Setenv("ADMIN_CONFIG_FILE", write("open.conf", "ADMIN_EMAIL=x@y.zz\n", 0o644))
	if _, err := loadConfig(); err == nil {
		t.Fatal("group/world-readable secret file accepted")
	}
	t.Setenv("ADMIN_CONFIG_FILE", "")
	t.Setenv("ADMIN_PASSWORD_FILE", "")
	t.Setenv("ADMIN_PASSWORD", "short")
	if _, err := loadConfig(); err == nil {
		t.Fatal("short password accepted")
	}
}

func TestProvisionIdempotent(t *testing.T) {
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
	cfg := adminConfig{Email: "provision-test@example.com", Phone: "5550001", Name: "Prov", Password: "a-long-test-password"}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM rbac.user_role WHERE user_id IN (SELECT id FROM "user"."user" WHERE email=$1)`, cfg.Email)
		db.Exec(`DELETE FROM "user"."user" WHERE email=$1`, cfg.Email)
	})
	if _, err := provision(cfg); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id JOIN "user"."user" u ON u.id=ur.user_id WHERE u.email=$1 AND r.name='super_admin'`, cfg.Email).Scan(&n); err != nil || n != 1 {
		t.Fatalf("grant missing: %d %v", n, err)
	}
	if _, err := provision(cfg); err != nil {
		t.Fatalf("second run must be a no-op: %v", err)
	}
}
