package employee

import "fmt"

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxPage         = 1000000
)

// EmployeeListRequest controls a bounded page of employees and its ordering.
type EmployeeListRequest struct {
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
}

func DefaultEmployeeListRequest() EmployeeListRequest {
	return EmployeeListRequest{Page: 1, PageSize: DefaultPageSize, SortBy: "id", SortOrder: "asc"}
}

// Validate also protects repository callers outside the HTTP layer.
func (request EmployeeListRequest) Validate() error {
	if request.Page < 1 || request.Page > MaxPage {
		return fmt.Errorf("%w: page must be between 1 and %d", ErrInvalid, MaxPage)
	}
	if request.PageSize < 1 || request.PageSize > MaxPageSize {
		return fmt.Errorf("%w: page_size must be between 1 and %d", ErrInvalid, MaxPageSize)
	}
	switch request.SortBy {
	case "id", "name", "email", "department":
	default:
		return fmt.Errorf("%w: sort_by must be id, name, email or department", ErrInvalid)
	}
	if request.SortOrder != "asc" && request.SortOrder != "desc" {
		return fmt.Errorf("%w: sort_order must be asc or desc", ErrInvalid)
	}
	return nil
}

// EmployeePage contains the selected records and the total across all pages.
type EmployeePage struct {
	Items      []Employee
	TotalItems int64
}
