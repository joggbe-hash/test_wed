package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/user/*.sql migrations/system/*.sql
var migrationFiles embed.FS

const (
	userMigrationLockID   int64 = 814_001
	systemMigrationLockID int64 = 814_002
)

type migration struct {
	version int
	name    string
	sql     string
}

func RunMigrations(ctx context.Context) error {
	if err := runMigrationScope(ctx, userPool, "migrations/user", userMigrationLockID); err != nil {
		return fmt.Errorf("run user database migrations: %w", err)
	}
	if err := runMigrationScope(ctx, systemPool, "migrations/system", systemMigrationLockID); err != nil {
		return fmt.Errorf("run system database migrations: %w", err)
	}
	return nil
}

func runMigrationScope(
	ctx context.Context,
	pool *pgxpool.Pool,
	directory string,
	lockID int64,
) error {
	migrations, err := loadMigrations(directory)
	if err != nil {
		return err
	}

	connection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer connection.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockID)

	if _, err := connection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, item := range migrations {
		var applied bool
		if err := connection.QueryRow(
			ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			item.version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", item.name, err)
		}
		if applied {
			continue
		}

		transaction, err := connection.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", item.name, err)
		}
		if _, err := transaction.Exec(ctx, item.sql); err != nil {
			transaction.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", item.name, err)
		}
		if _, err := transaction.Exec(
			ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			item.version,
			item.name,
		); err != nil {
			transaction.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", item.name, err)
		}
		if err := transaction.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", item.name, err)
		}
	}
	return nil
}

func loadMigrations(directory string) ([]migration, error) {
	names, err := fs.Glob(migrationFiles, path.Join(directory, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("list migrations in %s: %w", directory, err)
	}
	sort.Strings(names)

	result := make([]migration, 0, len(names))
	seenVersions := make(map[int]string, len(names))
	for _, name := range names {
		base := path.Base(name)
		versionText, _, found := strings.Cut(base, "_")
		if !found {
			return nil, fmt.Errorf("migration %s must start with a numeric version", base)
		}
		version, err := strconv.Atoi(versionText)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("migration %s has invalid version", base)
		}
		if previous, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("migration version %d is duplicated by %s and %s", version, previous, base)
		}

		content, err := migrationFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", base, err)
		}
		seenVersions[version] = base
		result = append(result, migration{version: version, name: base, sql: string(content)})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no migrations found in %s", directory)
	}
	return result, nil
}
