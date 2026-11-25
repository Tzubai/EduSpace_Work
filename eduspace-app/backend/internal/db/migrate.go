package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var migrationName = regexp.MustCompile(`^\d+_.*\.sql$`)

// Migrate applies numbered .sql files once, under an advisory lock.
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(hashtext('eduspace_migrations'))`); err != nil {
		return err
	}
	defer pool.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtext('eduspace_migrations'))`)

	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT`); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && migrationName.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		sqlText, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		hash := sha256.Sum256(sqlText)
		checksum := hex.EncodeToString(hash[:])

		var appliedChecksum string
		err = pool.QueryRow(ctx, `SELECT COALESCE(checksum,'') FROM schema_migrations WHERE name=$1`, name).Scan(&appliedChecksum)
		if err == nil {
			if appliedChecksum == "" {
				if _, err := pool.Exec(ctx, `UPDATE schema_migrations SET checksum=$1 WHERE name=$2`, checksum, name); err != nil {
					return err
				}
				continue
			}
			if appliedChecksum != checksum {
				return fmt.Errorf("migration %s was modified after it was applied", name)
			}
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sqlText)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (name, checksum) VALUES ($1,$2)`, name, checksum); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		fmt.Println("applied migration", name)
	}
	return nil
}
