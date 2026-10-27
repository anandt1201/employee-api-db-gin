package config

import "testing"

func TestLoad(t *testing.T) {
	cases := []struct {
		name      string
		values    map[string]string
		wantError bool
	}{
		{name: "missing database URL", values: map[string]string{}, wantError: true},
		{name: "defaults", values: map[string]string{"DATABASE_URL": "postgres://localhost/employees"}},
		{name: "bad pool size", values: map[string]string{"DATABASE_URL": "x", "DB_MAX_CONNECTIONS": "0"}, wantError: true},
		{name: "bad timeout", values: map[string]string{"DATABASE_URL": "x", "DB_QUERY_TIMEOUT": "-1s"}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := load(func(key string) string {
				return tc.values[key]
			})
			if (err != nil) != tc.wantError {
				t.Fatalf("err=%v wantError=%v", err, tc.wantError)
			}
			if !tc.wantError && (cfg.HTTPAddr != "127.0.0.1:8080" || cfg.MaxConnections != 10) {
				t.Fatalf("unexpected defaults: address=%s maxConnections=%d", cfg.HTTPAddr, cfg.MaxConnections)
			}
		})
	}
}
