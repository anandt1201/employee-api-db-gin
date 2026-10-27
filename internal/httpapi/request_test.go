package httpapi_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/employee-api-db-gin/internal/employee"
	"example.com/employee-api-db-gin/internal/httpapi"
	"example.com/employee-api-db-gin/internal/testutil"
)

func TestRequestBodyValidation(t *testing.T) {
	router := httpapi.NewRouter(employee.NewService(testutil.NewEmployeeRepository()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{name: "null", body: "null", status: http.StatusBadRequest},
		{name: "oversized", body: `{"name":"` + strings.Repeat("a", 1<<20) + `"}`, status: http.StatusRequestEntityTooLarge},
		{name: "trailing oversized whitespace", body: `{"name":"A","email":"a@example.com","department":"IT"}` + strings.Repeat(" ", 1<<20), status: http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/employees", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d: %s", response.Code, tc.status, response.Body.String())
			}
		})
	}
}
