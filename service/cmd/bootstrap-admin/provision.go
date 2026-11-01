package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var phonePattern = regexp.MustCompile(`^[0-9]{3,15}$`)

const minPasswordLen = 12

// adminConfig describes the administrator account created by -from-env.
type adminConfig struct {
	Email, Phone, Name, Password string
}

// loadConfig reads ADMIN_EMAIL, ADMIN_PHONE, ADMIN_NAME and ADMIN_PASSWORD.
// Values may also come from a KEY=VALUE file named by ADMIN_CONFIG_FILE, and the
// password from a file named by ADMIN_PASSWORD_FILE (e.g. a mounted Kubernetes
// Secret). Precedence: environment variable > ADMIN_PASSWORD_FILE > config file.
// Secret files must not be readable by group or others.
func loadConfig() (adminConfig, error) {
	vals := map[string]string{}
	if path := os.Getenv("ADMIN_CONFIG_FILE"); path != "" {
		b, err := readSecretFile(path)
		if err != nil {
			return adminConfig{}, err
		}
		if vals, err = parseKV(string(b)); err != nil {
			return adminConfig{}, fmt.Errorf("ADMIN_CONFIG_FILE: %w", err)
		}
	}
	if path := os.Getenv("ADMIN_PASSWORD_FILE"); path != "" {
		b, err := readSecretFile(path)
		if err != nil {
			return adminConfig{}, err
		}
		vals["ADMIN_PASSWORD"] = strings.TrimRight(string(b), "\r\n")
	}
	get := func(k string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return vals[k]
	}
	cfg := adminConfig{Email: get("ADMIN_EMAIL"), Phone: get("ADMIN_PHONE"), Name: get("ADMIN_NAME"), Password: get("ADMIN_PASSWORD")}
	if cfg.Name == "" {
		cfg.Name = "Super Admin"
	}
	if a, err := mail.ParseAddress(cfg.Email); err != nil || a.Address != cfg.Email {
		return cfg, errors.New("ADMIN_EMAIL missing or invalid")
	}
	if !phonePattern.MatchString(cfg.Phone) {
		return cfg, errors.New("ADMIN_PHONE missing or invalid (3-15 digits)")
	}
	if len(cfg.Password) < minPasswordLen || len(cfg.Password) > 72 {
		return cfg, fmt.Errorf("ADMIN_PASSWORD must be %d-72 bytes", minPasswordLen)
	}
	return cfg, nil
}

func readSecretFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open secret file %s", path)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%s must not be accessible by group or others (chmod 600)", path)
	}
	var sb strings.Builder
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), 64*1024)
	for sc.Scan() {
		sb.WriteString(sc.Text())
		sb.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("cannot read secret file %s", path)
	}
	return []byte(sb.String()), nil
}

// parseKV parses KEY=VALUE lines; blank lines and # comments are ignored.
func parseKV(s string) (map[string]string, error) {
	out := map[string]string{}
	for i, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", i+1)
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
		out[strings.TrimSpace(k)] = v
	}
	return out, nil
}

// provision creates the administrator account if needed and grants super_admin.
// It is idempotent: once any active super_admin exists it changes nothing and
// succeeds, so it is safe to run on every deploy. An existing account with the
// same email is reused and its password is left untouched.
func provision(cfg adminConfig) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", os.Getenv("DBURL"))
	if err != nil {
		return "", errors.New("open database")
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", errors.New("begin bootstrap")
	}
	defer tx.Rollback()
	exists, err := lockAndCheck(ctx, tx)
	if err != nil {
		return "", err
	}
	if exists {
		return "Administrator already exists; nothing to do", nil
	}
	var id int
	err = tx.QueryRowContext(ctx, `SELECT id FROM "user"."user" WHERE email=$1 ORDER BY id LIMIT 1`, cfg.Email).Scan(&id)
	created := false
	switch {
	case err == sql.ErrNoRows:
		hash, herr := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.DefaultCost)
		if herr != nil {
			return "", errors.New("hash password")
		}
		if err := tx.QueryRowContext(ctx, `INSERT INTO "user"."user"(email,phone,name,hashed_passwd) VALUES($1,$2,$3,$4) RETURNING id`, cfg.Email, cfg.Phone, cfg.Name, string(hash)).Scan(&id); err != nil {
			return "", errors.New("create administrator account")
		}
		created = true
	case err != nil:
		return "", errors.New("look up administrator account")
	}
	if err := grantAdmin(ctx, tx, id); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", errors.New("commit bootstrap")
	}
	if created {
		return "Administrator account created and granted super_admin", nil
	}
	return "Existing account granted super_admin", nil
}
