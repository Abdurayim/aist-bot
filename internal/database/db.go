package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"aist-bot/internal/config"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// DB is the process-wide database handle.
var DB *sql.DB

// Connect opens the SQLite database and applies every embedded migration.
// Migrations are embedded in the binary so the bot runs correctly no matter
// which directory it is started from.
func Connect(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000", cfg.Path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite handles a single writer at a time; a small pool avoids
	// "database is locked" under concurrent updates.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to reach database: %w", err)
	}

	DB = db

	if err := Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Migrate applies every embedded .sql migration that has not run yet, in
// filename order, recording each one so it is never applied twice.
//
// Tracking matters because not every statement can be written to be safe to
// re-run: ALTER TABLE ADD COLUMN fails the second time.
func Migrate() error {
	if _, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	applied, err := appliedMigrations()
	if err != nil {
		return err
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if applied[name] {
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}

		if _, err := DB.Exec(string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}

		if _, err := DB.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", name, err)
		}
	}

	return nil
}

// appliedMigrations returns the set of migrations already run against this
// database.
func appliedMigrations() (map[string]bool, error) {
	rows, err := DB.Query(`SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("failed to read applied migrations: %w", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

// Close closes the database handle.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// HealthCheck verifies the connection is still usable.
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database is not connected")
	}
	return DB.Ping()
}
