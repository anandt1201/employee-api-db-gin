package employee

import "context"

// Repository is owned by its consumer, the employee service. Implementations
// must return ErrNotFound/ErrConflict (possibly wrapped) where applicable.
// List returns a bounded page with a non-nil Items slice and deterministic ordering.
// Email uniqueness is case insensitive.
type Repository interface {
	List(context.Context, EmployeeListRequest) (EmployeePage, error)
	Get(context.Context, int64) (Employee, error)
	Create(context.Context, EmployeeRequest) (Employee, error)
	Update(context.Context, int64, EmployeeRequest) (Employee, error)
	Delete(context.Context, int64) error
}
