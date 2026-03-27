package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/jakub/panelox/internal/agent"
	"github.com/jakub/panelox/internal/config"
	"github.com/jakub/panelox/internal/pkg/db"
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

	migrationsDir := findMigrationsDir()
	if err := db.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if *migrateOnly {
		log.Println("Migrations completed")
		os.Exit(0)
	}

	log.Println("Starting Panelox Agent...")
	a := agent.New(cfg, pool)
	if err := a.Start(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}
}

func findMigrationsDir() string {
	for _, dir := range []string{"migrations", filepath.Join("..", "..", "migrations"), "/opt/panelox/migrations"} {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	exe, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exe), "migrations")
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	log.Fatal("Could not find migrations directory")
	return ""
}
