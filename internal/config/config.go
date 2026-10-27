package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is loaded once in the composition root, then passed explicitly.
type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	MaxConnections int32
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
}

func Load() (Config, error) {
	return load(os.Getenv)
}

func load(env func(string) string) (Config, error) {
	cfg := Config{HTTPAddr: env("HTTP_ADDR"), DatabaseURL: env("DATABASE_URL"), MaxConnections: 10, ConnectTimeout: 5 * time.Second, QueryTimeout: 3 * time.Second}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = "127.0.0.1:8080"
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if value := env("DB_MAX_CONNECTIONS"); value != "" {
		n, err := strconv.ParseInt(value, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return Config{}, errors.New("DB_MAX_CONNECTIONS must be between 1 and 100")
		}
		cfg.MaxConnections = int32(n)
	}
	for _, setting := range []struct {
		name   string
		target *time.Duration
	}{
		{"DB_CONNECT_TIMEOUT", &cfg.ConnectTimeout},
		{"DB_QUERY_TIMEOUT", &cfg.QueryTimeout},
	} {
		if value := env(setting.name); value != "" {
			duration, err := time.ParseDuration(value)
			if err != nil || duration <= 0 {
				return Config{}, fmt.Errorf("%s must be a positive duration", setting.name)
			}
			*setting.target = duration
		}
	}
	return cfg, nil
}
