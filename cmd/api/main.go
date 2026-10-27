package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/employee-api-db-gin/internal/config"
	"example.com/employee-api-db-gin/internal/database"
	"example.com/employee-api-db-gin/internal/employee"
	"example.com/employee-api-db-gin/internal/httpapi"
	"example.com/employee-api-db-gin/internal/repository"
)

// @title Employee API
// @version 1.0
// @description Employee CRUD API. The list endpoint allows 10 requests per client IP per rolling 5-second window. Employees are stored in PostgreSQL.
// @BasePath /
func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// Composition root: explicit constructor injection, no DI container required.
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(context.Background(), cfg)
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	// Fail before accepting requests if migrations have not been applied.
	check, cancel := context.WithTimeout(context.Background(), cfg.QueryTimeout)
	err = db.WithContext(check).Table("employees").Select("id", "name", "email", "department").Limit(1).Find(&[]employee.Employee{}).Error
	cancel()
	if err != nil {
		return errors.New("employees schema is unavailable; apply migrations before starting the API")
	}
	repo := repository.NewEmployeeRepository(db, cfg.QueryTimeout)
	service := employee.NewService(repo)
	router := httpapi.NewRouter(service, log)
	addr := cfg.HTTPAddr
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	failures := make(chan error, 1)
	go func() {
		log.Info("server starting", "address", addr)
		failures <- server.ListenAndServe()
	}()
	select {
	case err := <-failures:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}
