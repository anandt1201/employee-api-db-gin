package httpapi_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/employee-api-db-gin/internal/employee"
	"example.com/employee-api-db-gin/internal/httpapi"
	"example.com/employee-api-db-gin/internal/testutil"
)

func TestSwaggerDocumentation(t *testing.T) {
	router := httpapi.NewRouter(employee.NewService(testutil.NewEmployeeRepository()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, path := range []string{"/swagger/index.html", "/swagger/swagger-ui.css", "/swagger/doc.json"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 200 {
			t.Fatalf("%s: status %d", path, response.Code)
		}
		if path == "/swagger/index.html" && !strings.Contains(response.Body.String(), "Swagger UI") {
			t.Fatal("Swagger UI was not served")
		}
		if path != "/swagger/doc.json" {
			continue
		}
		var spec struct {
			Swagger string                                `json:"swagger"`
			Paths   map[string]map[string]json.RawMessage `json:"paths"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
			t.Fatal(err)
		}
		if spec.Swagger != "2.0" {
			t.Fatal("unexpected specification version")
		}
		expected := map[string][]string{
			"/api/v1/employees":      {"get", "post"},
			"/api/v1/employees/{id}": {"get", "put", "delete"},
			"/healthz":               {"get"},
		}
		for route, methods := range expected {
			for _, method := range methods {
				if len(spec.Paths[route][method]) == 0 {
					t.Fatalf("undocumented endpoint: %s %s", method, route)
				}
			}
		}
		var operation struct {
			Responses map[string]struct {
				Headers map[string]json.RawMessage `json:"headers"`
			} `json:"responses"`
		}
		if err := json.Unmarshal(spec.Paths["/api/v1/employees"]["get"], &operation); err != nil {
			t.Fatal(err)
		}
		if len(operation.Responses["429"].Headers["Retry-After"]) == 0 {
			t.Fatal("list rate-limit response header is undocumented")
		}
	}
}
