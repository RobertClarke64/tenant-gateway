package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Migrate(ctx context.Context) error {
	// Create migrations table if not exists
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Read migration files from embedded filesystem
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	// Sort migrations by version
	versions := make([]string, 0, len(migrations))
	for v := range migrations {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	for _, version := range versions {
		// Check if already applied
		var count int
		err := db.pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM schema_migrations WHERE version = $1",
			version,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("checking migration %s: %w", version, err)
		}

		if count > 0 {
			continue
		}

		// Apply migration
		sql := strings.TrimSpace(migrations[version])
		if sql == "" {
			continue
		}

		_, err = db.pool.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("applying migration %s: %w", version, err)
		}

		_, err = db.pool.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		)
		if err != nil {
			return fmt.Errorf("recording migration %s: %w", version, err)
		}
	}

	return nil
}

func loadMigrations() (map[string]string, error) {
	migrations := make(map[string]string)

	err := fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}

		content, err := migrationsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		// Use filename without extension as version
		filename := filepath.Base(path)
		version := strings.TrimSuffix(filename, ".sql")
		migrations[version] = string(content)

		return nil
	})

	return migrations, err
}
