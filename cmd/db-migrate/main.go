package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	jaxdb "jax-trading-assistant/libs/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	migrationsDir := strings.TrimSpace(os.Getenv("MIGRATIONS_DIR"))
	if migrationsDir == "" {
		migrationsDir = "/migrations"
	}

	sourceURL := migrationsDir
	if !strings.Contains(sourceURL, "://") {
		absPath, err := filepath.Abs(sourceURL)
		if err != nil {
			log.Fatalf("resolve migrations path: %v", err)
		}
		sourceURL = "file://" + filepath.ToSlash(absPath)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	log.Printf("applying migrations from %s", sourceURL)
	if err := jaxdb.RunMigrations(db, sourceURL); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	version, dirty, err := jaxdb.MigrationVersion(db, sourceURL)
	if err != nil {
		log.Fatalf("migration version: %v", err)
	}
	if dirty {
		log.Fatalf("migration state is dirty at version %d", version)
	}

	log.Printf("migrations ready at version %d", version)
	fmt.Println("db-migrate completed successfully")
}
