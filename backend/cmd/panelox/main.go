package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/jakub/panelox/internal/config"
	"github.com/jakub/panelox/internal/pkg/db"
	"github.com/jakub/panelox/internal/server"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "Run migrations and exit")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Find migrations directory
	migrationsDir := findMigrationsDir()
	if err := db.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if *migrateOnly {
		log.Println("Migrations completed successfully")
		os.Exit(0)
	}

	srv := server.New(cfg, pool)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func findMigrationsDir() string {
	// Try relative to executable
	exe, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exe), "migrations")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}

	// Try relative to working directory
	dir = "migrations"
	if _, err := os.Stat(dir); err == nil {
		return dir
	}

	// Try parent directory (when running from cmd/panelox)
	dir = filepath.Join("..", "..", "migrations")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}

	log.Fatal("Could not find migrations directory")
	return ""
}
