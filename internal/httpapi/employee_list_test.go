package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"reflect"
	"testing"

	"example.com/employee-api-db-gin/internal/employee"
	"example.com/employee-api-db-gin/internal/httpapi"
	"example.com/employee-api-db-gin/internal/testutil"
)

func TestEmployeePaginationAndSorting(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		status     int
		ids        []int64
		page       int
		pageSize   int
		totalPages int64
	}{
		{"defaults", "", 200, []int64{1, 2, 3, 4, 5}, 1, 20, 1},
		{"page and name order", "?page=2&page_size=2&sort_by=name&sort_order=asc", 200, []int64{5, 3}, 2, 2, 3},
		{"descending with stable ties", "?page_size=5&sort_by=name&sort_order=desc", 200, []int64{1, 3, 5, 2, 4}, 1, 5, 1},
		{"last partial page", "?page=3&page_size=2&sort_by=name", 200, []int64{1}, 3, 2, 3},
		{"beyond last page", "?page=9&page_size=2", 200, []int64{}, 9, 2, 3},
		{"department sort", "?sort_by=department&page_size=2", 200, []int64{2, 4}, 1, 2, 3},
		{"email sort", "?sort_by=email&sort_order=desc&page_size=2", 200, []int64{5, 4}, 1, 2, 3},
		{"descending ID", "?sort_order=desc&page_size=2", 200, []int64{5, 4}, 1, 2, 3},
		{name: "zero page", query: "?page=0", status: 400},
		{name: "negative page", query: "?page=-1", status: 400},
		{name: "too large page", query: "?page=1000001", status: 400},
		{name: "overflow page", query: "?page=999999999999999999999", status: 400},
		{name: "invalid number", query: "?page=abc", status: 400},
		{name: "empty page", query: "?page=", status: 400},
		{name: "zero size", query: "?page_size=0", status: 400},
		{name: "large size", query: "?page_size=101", status: 400},
		{name: "duplicate parameter", query: "?page=1&page=2", status: 400},
		{name: "invalid sort field", query: "?sort_by=salary", status: 400},
		{name: "unsafe sort field", query: "?sort_by=name%3BDROP%20TABLE%20employees", status: 400},
		{name: "unsafe direction", query: "?sort_order=desc%3BDROP", status: 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := testutil.NewEmployeeRepository()
			names := []string{"Zoe", "Alice", "Charlie", "Alice", "Bob"}
			for i, name := range names {
				department := "IT"
				if i == 1 || i == 3 {
					department = "HR"
				}
				if _, err := repo.Create(context.Background(), employee.EmployeeRequest{Name: name, Email: fmt.Sprintf("e%d@example.com", i+1), Department: department}); err != nil {
					t.Fatal(err)
				}
			}
			router := httpapi.NewRouter(employee.NewService(repo), slog.New(slog.NewTextHandler(io.Discard, nil)))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/employees"+tc.query, nil))
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d: %s", response.Code, tc.status, response.Body.String())
			}
			if tc.status != 200 {
				return
			}
			var result httpapi.EmployeeListResponse
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			ids := make([]int64, len(result.Data))
			for i, item := range result.Data {
				ids[i] = item.ID
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("ids=%v want=%v", ids, tc.ids)
			}
			if result.Data == nil || result.Pagination.Page != tc.page || result.Pagination.PageSize != tc.pageSize || result.Pagination.TotalItems != 5 || result.Pagination.TotalPages != tc.totalPages {
				t.Fatalf("unexpected pagination: %+v", result)
			}
		})
	}
}

func TestPaginationQueriesShareRateLimit(t *testing.T) {
	router := httpapi.NewRouter(employee.NewService(testutil.NewEmployeeRepository()), slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 1; i <= 11; i++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/employees?page=%d", i), nil))
		expected := 200
		if i == 11 {
			expected = 429
		}
		if response.Code != expected {
			t.Fatalf("request %d: status=%d want=%d", i, response.Code, expected)
		}
	}
}
