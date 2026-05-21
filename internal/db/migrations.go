package db

import (
	"context"
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations executes all SQL migration files against the database
func RunMigrations(ctx context.Context, conn *pgx.Conn) error {
	log.Println("🔄 Running database migrations...")

	// Read migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", entry.Name(), err)
		}

		sql := string(content)

		// Split by semicolons to handle multiple statements
		statements := strings.Split(sql, ";")

		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			log.Printf("📝 Executing: %s...", entry.Name())
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
			}
		}

		log.Printf("✅ Migration completed: %s", entry.Name())
	}

	log.Println("✅ All migrations completed successfully")
	return nil
}
