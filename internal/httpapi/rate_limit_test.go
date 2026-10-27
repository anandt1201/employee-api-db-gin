package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"example.com/employee-api-db-gin/internal/employee"
	"example.com/employee-api-db-gin/internal/testutil"

	"github.com/gin-gonic/gin"
)

func rateRequest(router http.Handler, ip, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = ip + ":12345"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestRateLimitSlidingWindow(t *testing.T) {
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.GET("/", newRateLimiter(10, 5*time.Second, func() time.Time {
		return current
	}), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 10; i++ {
		if response := rateRequest(router, "192.0.2.1", "/"); response.Code != 204 {
			t.Fatalf("request %d: got %d", i+1, response.Code)
		}
	}
	response := rateRequest(router, "192.0.2.1", "/")
	if response.Code != 429 || response.Header().Get("Retry-After") != "5" {
		t.Fatalf("expected 429 and Retry-After 5, got %d %v", response.Code, response.Header())
	}
	if response := rateRequest(router, "192.0.2.2", "/"); response.Code != 204 {
		t.Fatal("different client must have an independent quota")
	}
	current = current.Add(4999 * time.Millisecond)
	response = rateRequest(router, "192.0.2.1", "/")
	if response.Code != 429 || response.Header().Get("Retry-After") != "1" {
		t.Fatal("limit must remain active until the window expires")
	}
	current = current.Add(time.Millisecond)
	if response := rateRequest(router, "192.0.2.1", "/"); response.Code != 204 {
		t.Fatal("quota must recover at the window boundary")
	}
	current = current.Add(time.Millisecond)
	for i := 0; i < 9; i++ {
		if response := rateRequest(router, "192.0.2.1", "/"); response.Code != 204 {
			t.Fatal("expected remaining quota")
		}
	}
	current = current.Add(4999 * time.Millisecond)
	if response := rateRequest(router, "192.0.2.1", "/"); response.Code != 204 {
		t.Fatal("oldest request should expire independently")
	}
	if response := rateRequest(router, "192.0.2.1", "/"); response.Code != 429 {
		t.Fatal("newer requests must still count in the sliding window")
	}
}

func TestRateLimitConcurrentRequests(t *testing.T) {
	router := gin.New()
	router.GET("/", RateLimit(10, 5*time.Second), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	var wg sync.WaitGroup
	results := make(chan int, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- rateRequest(router, "192.0.2.1", "/").Code
		}()
	}
	wg.Wait()
	close(results)
	allowed, blocked := 0, 0
	for status := range results {
		switch status {
		case 204:
			allowed++
		case 429:
			blocked++
		default:
			t.Fatalf("unexpected status %d", status)
		}
	}
	if allowed != 10 || blocked != 20 {
		t.Fatalf("allowed=%d blocked=%d", allowed, blocked)
	}
}

func TestEmployeeListRateLimitScope(t *testing.T) {
	router := NewRouter(employee.NewService(testutil.NewEmployeeRepository()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 0; i < 10; i++ {
		if response := rateRequest(router, "192.0.2.1", "/api/v1/employees"); response.Code != 200 {
			t.Fatalf("list request %d: got %d", i+1, response.Code)
		}
	}
	response := rateRequest(router, "192.0.2.1", "/api/v1/employees")
	if response.Code != 429 {
		t.Fatalf("expected list to be limited, got %d", response.Code)
	}
	for _, path := range []string{"/healthz", "/api/v1/employees/1"} {
		response := rateRequest(router, "192.0.2.1", path)
		if response.Code == 429 {
			t.Fatalf("rate limit leaked to %s", path)
		}
	}
}
