//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/employee-api-db-gin/internal/config"
	"example.com/employee-api-db-gin/internal/database"
	"example.com/employee-api-db-gin/internal/employee"
	"example.com/employee-api-db-gin/internal/httpapi"
	"example.com/employee-api-db-gin/internal/repository"
	"example.com/employee-api-db-gin/migrations"
	"github.com/golang-migrate/migrate/v4"
	migratedb "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

func TestEmployeeRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	open := func(connectionURL string) *gorm.DB {
		t.Helper()
		db, err := database.Open(ctx, config.Config{DatabaseURL: connectionURL, MaxConnections: 5, ConnectTimeout: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	closeDB := func(db *gorm.DB) {
		pool, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	}
	admin := open(databaseURL)
	defer closeDB(admin)
	schema := fmt.Sprintf("employee_test_%d", time.Now().UnixNano())
	// The identifier is generated exclusively from a fixed prefix and integer.
	if err := admin.WithContext(ctx).Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := admin.WithContext(cleanup).Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Errorf("cleanup test schema: %v", err)
		}
	}()
	parsedURL, err := url.Parse(databaseURL)
	if err != nil || parsedURL.Host == "" {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	schemaURL := parsedURL.String()
	migrationDB := open(schemaURL)
	sqlDB, err := migrationDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	driver, err := migratedb.WithInstance(sqlDB, &migratedb.Config{SchemaName: schema, StatementTimeout: 10 * time.Second})
	if err != nil {
		closeDB(migrationDB)
		t.Fatal(err)
	}
	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		t.Fatal(err)
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		t.Fatal(err)
	}
	defer migrator.Close()
	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Up(); !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("second migration run: %v", err)
	}
	db := open(schemaURL)
	defer closeDB(db)
	repo := repository.NewEmployeeRepository(db, 3*time.Second)
	service := employee.NewService(repo)
	router := httpapi.NewRouter(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, response.Code, status, response.Body.String())
		}
		return response
	}
	if response := request("GET", "/api/v1/employees", "", 200); !strings.Contains(response.Body.String(), `"data":[]`) {
		t.Fatal("empty list must serialize as []")
	}
	response := request("POST", "/api/v1/employees", `{"name":" Anand ","email":"ANAND@example.com","department":"IT"}`, 201)
	var created struct {
		Data httpapi.EmployeeResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Data.Name != "Anand" || created.Data.Email != "anand@example.com" {
		t.Fatalf("normalization failed: %+v", created.Data)
	}
	path := fmt.Sprintf("/api/v1/employees/%d", created.Data.ID)
	if response.Header().Get("Location") != path {
		t.Fatal("incorrect Location header")
	}
	request("GET", path, "", 200)
	request("POST", "/api/v1/employees", `{"name":"Other","email":"anand@example.com","department":"IT"}`, 409)
	request("PUT", path, `{"name":"Anand T","email":"anand@example.com","department":"Platform"}`, 200)

	secondDB := open(schemaURL)
	persisted, err := repository.NewEmployeeRepository(secondDB, 3*time.Second).Get(ctx, created.Data.ID)
	closeDB(secondDB)
	if err != nil || persisted.Department != "Platform" {
		t.Fatalf("record not visible through another connection pool: %v", err)
	}
	other, err := service.Create(ctx, employee.EmployeeRequest{Name: "Other", Email: "other@example.com", Department: "HR"})
	if err != nil {
		t.Fatal(err)
	}
	paged := request("GET", "/api/v1/employees?page=1&page_size=1&sort_by=id&sort_order=desc", "", 200)
	var pageResponse httpapi.EmployeeListResponse
	if err := json.Unmarshal(paged.Body.Bytes(), &pageResponse); err != nil {
		t.Fatal(err)
	}
	if len(pageResponse.Data) != 1 || pageResponse.Data[0].ID != other.ID || pageResponse.Pagination.TotalItems != 2 || pageResponse.Pagination.TotalPages != 2 {
		t.Fatalf("unexpected database pagination: %+v", pageResponse)
	}
	request("PUT", path, `{"name":"Changed","email":"other@example.com","department":"HR"}`, 409)
	unchanged, err := repo.Get(ctx, created.Data.ID)
	if err != nil || unchanged.Department != "Platform" {
		t.Fatal("failed update changed stored data")
	}
	request("DELETE", path, "", 204)
	request("GET", path, "", 404)
	request("DELETE", path, "", 404)
	request("PUT", path, `{"name":"A","email":"a@example.com","department":"IT"}`, 404)
	if err := repo.Delete(ctx, other.ID); err != nil {
		t.Fatal(err)
	}

	var successes atomic.Int32
	var conflicts atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Create(ctx, employee.EmployeeRequest{Name: "Concurrent", Email: "same@example.com", Department: "IT"})
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, employee.ErrConflict):
				conflicts.Add(1)
			default:
				t.Errorf("concurrent create: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || conflicts.Load() != 11 {
		t.Fatalf("success=%d conflicts=%d", successes.Load(), conflicts.Load())
	}
	canceled, stop := context.WithCancel(ctx)
	stop()
	if _, err := repo.List(canceled, employee.DefaultEmployeeListRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("context cancellation: %v", err)
	}
	request(http.MethodGet, "/swagger/doc.json", "", 200)
}
