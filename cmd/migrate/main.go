package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"example.com/employee-api-db-gin/internal/config"
	"example.com/employee-api-db-gin/internal/database"
	"example.com/employee-api-db-gin/migrations"
	"github.com/golang-migrate/migrate/v4"
	migratedb "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}
	if len(os.Args) > 2 || (action != "up" && action != "version") {
		return errors.New("usage: go run ./cmd/migrate [up|version]")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	orm, err := database.Open(context.Background(), config.Config{
		DatabaseURL:    url,
		MaxConnections: 2,
		ConnectTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	db, err := orm.DB()
	if err != nil {
		return err
	}
	defer db.Close()
	driver, err := migratedb.WithInstance(db, &migratedb.Config{StatementTimeout: 30 * time.Second})
	if err != nil {
		return fmt.Errorf("initialize migration driver: %w", err)
	}
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return err
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return err
	}
	defer migrator.Close()
	migrator.LockTimeout = 10 * time.Second
	if action == "up" {
		if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	}
	version, dirty, err := migrator.Version()
	if err != nil {
		return err
	}
	slog.Info("database schema", "version", version, "dirty", dirty)
	return nil
}
