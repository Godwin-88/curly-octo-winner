// Command migrate applies the SQL migrations in api/migrations to the database
// configured by DATABASE_URL, in filename order, exactly once.
//
// It exists because the Makefile's migrate-up target was previously a stub:
// without a runner there was no reproducible way to bring a fresh development
// database (or a new environment) up to the schema the API expects, and `make
// run` started a server that answered 500 to every query.
//
// Usage:
//
//	go run ./cmd/migrate -status            # list applied / pending / drifted
//	go run ./cmd/migrate                    # apply all pending migrations
//	go run ./cmd/migrate -baseline          # record all as applied, run nothing
//
// Each migration runs inside its own transaction together with the bookkeeping
// insert, so a failing migration leaves the database untouched and the runner
// can simply be re-run after the file is fixed. Applied versions and file
// checksums are tracked in schema_migrations; a checksum mismatch means an
// already-applied file was edited after the fact and is reported as drift.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/shule360/api/internal/config"
	"github.com/shule360/api/pkg/supabase"
)

const migrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    text PRIMARY KEY,
	checksum   text NOT NULL,
	applied_at timestamptz NOT NULL DEFAULT now()
)`

type migration struct {
	version  string
	path     string
	checksum string
	sql      string
}

type applied struct {
	checksum  string
	appliedAt time.Time
}

func main() {
	_ = godotenv.Load(".env")

	dir := flag.String("dir", "migrations", "directory holding NN_name.sql migration files")
	statusOnly := flag.Bool("status", false, "print migration status and exit without changing anything")
	baseline := flag.Bool("baseline", false, "record every migration as applied WITHOUT executing it (for databases created by hand)")
	flag.Parse()

	if err := run(context.Background(), *dir, *statusOnly, *baseline); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dir string, statusOnly, baseline bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	files, err := loadMigrations(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .sql migrations found in %s", dir)
	}

	pool, err := supabase.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}
	if _, err := pool.Exec(ctx, migrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	appliedByVersion, err := loadApplied(ctx, pool)
	if err != nil {
		return err
	}

	// Report every migration with its state and collect the ones to run.
	var pending []migration
	var drifted []string
	for _, m := range files {
		rec, ok := appliedByVersion[m.version]
		switch {
		case !ok:
			fmt.Printf("  pending  %s\n", m.version)
			pending = append(pending, m)
		case rec.checksum != m.checksum:
			fmt.Printf("  DRIFTED  %s (applied %s, file changed since)\n", m.version, rec.appliedAt.Format(time.RFC3339))
			drifted = append(drifted, m.version)
		default:
			fmt.Printf("  applied  %s (%s)\n", m.version, rec.appliedAt.Format(time.RFC3339))
		}
	}

	if len(drifted) > 0 {
		fmt.Printf("\nwarning: %d migration file(s) changed after being applied: %s\n",
			len(drifted), strings.Join(drifted, ", "))
	}

	if statusOnly {
		fmt.Printf("\n%d migration(s) applied, %d pending\n", len(files)-len(pending), len(pending))
		return nil
	}

	if baseline {
		if len(pending) == 0 {
			fmt.Println("\nnothing to baseline: all migrations are already recorded")
			return nil
		}
		for _, m := range pending {
			if _, err := pool.Exec(ctx,
				`INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)
				 ON CONFLICT (version) DO NOTHING`, m.version, m.checksum); err != nil {
				return fmt.Errorf("baseline %s: %w", m.version, err)
			}
		}
		fmt.Printf("\nbaselined %d migration(s) without executing them\n", len(pending))
		return nil
	}

	if len(pending) == 0 {
		fmt.Println("\ndatabase is up to date")
		return nil
	}

	for _, m := range pending {
		if err := apply(ctx, pool, m); err != nil {
			return fmt.Errorf("apply %s: %w", m.version, err)
		}
		fmt.Printf("  applied  %s\n", m.version)
	}
	fmt.Printf("\napplied %d migration(s)\n", len(pending))
	return nil
}

// apply executes one migration file and records it, atomically.
func apply(ctx context.Context, pool *pgxpool.Pool, m migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The simple query protocol is used on purpose: migration files contain
	// many statements each (and DO $$ blocks), which the extended protocol
	// rejects. The connection is already inside the transaction opened above.
	results := tx.Conn().PgConn().Exec(ctx, m.sql)
	if _, err := results.ReadAll(); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)`,
		m.version, m.checksum); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func loadApplied(ctx context.Context, pool *pgxpool.Pool) (map[string]applied, error) {
	rows, err := pool.Query(ctx, `SELECT version, checksum, applied_at FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	out := map[string]applied{}
	for rows.Next() {
		var version string
		var rec applied
		if err := rows.Scan(&version, &rec.checksum, &rec.appliedAt); err != nil {
			return nil, err
		}
		out[version] = rec
	}
	return out, rows.Err()
}

// loadMigrations reads dir/*.sql (the seed/ subdirectory is ignored) in
// lexicographic order, which is why files are numbered 001_, 002_, ...
func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	var files []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		sum := sha256.Sum256(body)
		files = append(files, migration{
			version:  strings.TrimSuffix(e.Name(), ".sql"),
			path:     path,
			checksum: hex.EncodeToString(sum[:]),
			sql:      string(body),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })

	// Guard against duplicate versions.
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f.version] {
			return nil, fmt.Errorf("duplicate migration version %q", f.version)
		}
		seen[f.version] = true
	}
	return files, nil
}
