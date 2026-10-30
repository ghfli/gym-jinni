// bootstrap-admin requires direct database operator access and creates no HTTP bypass.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/v4/stdlib"
	"log"
	"os"
	"time"
)

func main() {
	userID := flag.Int("user-id", 0, "existing account to bootstrap as the first administrator")
	flag.Parse()
	if *userID <= 0 || os.Getenv("DBURL") == "" {
		log.Fatal("set DBURL and -user-id to an existing account")
	}
	if err := bootstrap(*userID); err != nil {
		log.Fatal(err)
	}
	log.Println("Administrator bootstrapped")
}
func bootstrap(userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", os.Getenv("DBURL"))
	if err != nil {
		return fmt.Errorf("open database")
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(187421,1)`); err != nil {
		return fmt.Errorf("lock bootstrap")
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rbac.user_role ur JOIN rbac.role r ON r.id=ur.role_id WHERE r.name='super_admin' AND (ur.expires_at IS NULL OR ur.expires_at>now()))`).Scan(&exists); err != nil {
		return fmt.Errorf("check administrator")
	}
	if exists {
		return fmt.Errorf("an administrator already exists; use authenticated role administration")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO rbac.user_role(user_id,role_id,granted_by) SELECT u.id,r.id,u.id FROM "user"."user" u CROSS JOIN rbac.role r WHERE u.id=$1 AND r.name='super_admin' ON CONFLICT(user_id,role_id) DO UPDATE SET expires_at=NULL,granted_by=EXCLUDED.granted_by,granted_at=now()`, userID)
	if err != nil {
		return fmt.Errorf("assign administrator")
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return fmt.Errorf("account or super_admin role missing")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap")
	}
	return nil
}
