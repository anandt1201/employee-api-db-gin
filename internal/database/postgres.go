package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/employee-api-db-gin/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open creates GORM with a bounded database/sql pool and verifies connectivity.
// The caller closes the underlying pool after HTTP shutdown.
func Open(ctx context.Context, cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
		// The application logs failures; avoid logging SQL parameters and employee data.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// Driver initialization errors can include connection credentials.
		return nil, errors.New("cannot initialize database; check DATABASE_URL")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, errors.New("cannot access database connection pool")
	}
	pool.SetMaxOpenConns(int(cfg.MaxConnections))
	pool.SetMaxIdleConns(int(cfg.MaxConnections))
	pool.SetConnMaxLifetime(time.Hour)
	pool.SetConnMaxIdleTime(5 * time.Minute)
	startup, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.PingContext(startup); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database connection check failed: %w", err)
	}
	return db, nil
}
