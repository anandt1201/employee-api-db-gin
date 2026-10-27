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

func TestEmployeeAPI(t *testing.T) {
	r := httpapi.NewRouter(employee.NewService(testutil.NewEmployeeRepository()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	cases := []struct {
		name, method, path, body string
		status                   int
		contains                 string
	}{
		{"empty list", "GET", "/api/v1/employees", "", 200, `"data":[]`},
		{"create", "POST", "/api/v1/employees", `{"name":" Anand ","email":"ANAND@example.com","department":"Engineering"}`, 201, `"email":"anand@example.com"`},
		{"get", "GET", "/api/v1/employees/1", "", 200, `"name":"Anand"`},
		{"duplicate email", "POST", "/api/v1/employees", `{"name":"Other","email":"anand@example.com","department":"HR"}`, 409, "conflict"},
		{"invalid email", "POST", "/api/v1/employees", `{"name":"A","email":"bad","department":"HR"}`, 400, "validation_error"},
		{"unknown field", "POST", "/api/v1/employees", `{"id":99,"name":"A","email":"a@example.com","department":"HR"}`, 400, "invalid_json"},
		{"trailing JSON", "POST", "/api/v1/employees", `{} {}`, 400, "invalid_json"},
		{"invalid id", "GET", "/api/v1/employees/nope", "", 400, "invalid_id"},
		{"missing employee", "GET", "/api/v1/employees/99", "", 404, "not_found"},
		{"update", "PUT", "/api/v1/employees/1", `{"name":"Anand T","email":"anand@example.com","department":"Platform"}`, 200, `"department":"Platform"`},
		{"delete", "DELETE", "/api/v1/employees/1", "", 204, ""},
		{"get deleted", "GET", "/api/v1/employees/1", "", 404, "not_found"},
		{"delete missing", "DELETE", "/api/v1/employees/1", "", 404, "not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("body = %s, want %s", w.Body.String(), tc.contains)
			}
			if tc.status == http.StatusCreated && w.Header().Get("Location") != "/api/v1/employees/1" {
				t.Fatal("missing resource Location")
			}
		})
	}
}
