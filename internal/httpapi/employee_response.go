package httpapi

import "example.com/employee-api-db-gin/internal/employee"

// EmployeeResponse is the public employee representation returned by the API.
// Mapping explicitly keeps the HTTP contract independent of the storage model.
type EmployeeResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
}

func newEmployeeResponse(e employee.Employee) EmployeeResponse {
	return EmployeeResponse{ID: e.ID, Name: e.Name, Email: e.Email, Department: e.Department}
}

// EmployeeDataResponse wraps one employee in the API success envelope.
type EmployeeDataResponse struct {
	Data EmployeeResponse `json:"data"`
}

// EmployeeListResponse wraps an employee list in the API success envelope.
type EmployeeListResponse struct {
	Data       []EmployeeResponse `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse describes the requested page and complete collection size.
type PaginationResponse struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"20"`
	TotalItems int64 `json:"total_items" example:"42"`
	TotalPages int64 `json:"total_pages" example:"3"`
}
