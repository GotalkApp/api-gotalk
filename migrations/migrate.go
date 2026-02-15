package migrations

import (
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

// Run executes all pending up migrations
func Run(dbURL string) error {
	source, err := iofs.New(migrationFiles, "sql")
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("📦 Migrations: no new migrations to apply")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, _ := m.Version()
	log.Printf("✅ Migrations applied successfully (version: %d, dirty: %v)", version, dirty)
	return nil
}

// Rollback reverts the last migration
func Rollback(dbURL string) error {
	source, err := iofs.New(migrationFiles, "sql")
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	log.Println("✅ Last migration rolled back successfully")
	return nil
}

// BuildDBURL constructs a PostgreSQL connection URL from components
func BuildDBURL(host, port, user, password, dbName, sslMode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbName, sslMode)
}
