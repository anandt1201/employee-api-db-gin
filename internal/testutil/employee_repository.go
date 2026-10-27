// Package testutil provides test doubles; application wiring never uses these.
package testutil

import (
	"context"
	"sort"
	"strings"
	"sync"

	"example.com/employee-api-db-gin/internal/employee"
)

type EmployeeRepository struct {
	mu        sync.RWMutex
	employees []employee.Employee
	nextID    int64
}

var _ employee.Repository = (*EmployeeRepository)(nil)

func NewEmployeeRepository() *EmployeeRepository {
	return &EmployeeRepository{employees: make([]employee.Employee, 0), nextID: 1}
}

func (r *EmployeeRepository) List(ctx context.Context, request employee.EmployeeListRequest) (employee.EmployeePage, error) {
	if err := request.Validate(); err != nil {
		return employee.EmployeePage{}, err
	}
	if err := ctx.Err(); err != nil {
		return employee.EmployeePage{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]employee.Employee, len(r.employees))
	copy(items, r.employees)
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		comparison := 0
		switch request.SortBy {
		case "id":
			if a.ID < b.ID {
				comparison = -1
			} else if a.ID > b.ID {
				comparison = 1
			}
		case "name":
			comparison = strings.Compare(a.Name, b.Name)
		case "email":
			comparison = strings.Compare(a.Email, b.Email)
		case "department":
			comparison = strings.Compare(a.Department, b.Department)
		}
		if comparison == 0 {
			return a.ID < b.ID
		}
		if request.SortOrder == "desc" {
			return comparison > 0
		}
		return comparison < 0
	})
	total := int64(len(items))
	start := (request.Page - 1) * request.PageSize
	if start >= len(items) {
		items = make([]employee.Employee, 0)
	} else {
		end := start + request.PageSize
		if end > len(items) {
			end = len(items)
		}
		items = items[start:end]
	}
	return employee.EmployeePage{Items: items, TotalItems: total}, nil
}
func (r *EmployeeRepository) Get(ctx context.Context, id int64) (employee.Employee, error) {
	if err := ctx.Err(); err != nil {
		return employee.Employee{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.employees {
		if e.ID == id {
			return e, nil
		}
	}
	return employee.Employee{}, employee.ErrNotFound
}

func (r *EmployeeRepository) Create(ctx context.Context, in employee.EmployeeRequest) (employee.Employee, error) {
	if err := ctx.Err(); err != nil {
		return employee.Employee{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.emailExists(in.Email, 0) {
		return employee.Employee{}, employee.ErrConflict
	}
	e := employee.Employee{ID: r.nextID, Name: in.Name, Email: in.Email, Department: in.Department}
	r.nextID++
	r.employees = append(r.employees, e)
	return e, nil
}

func (r *EmployeeRepository) Update(ctx context.Context, id int64, in employee.EmployeeRequest) (employee.Employee, error) {
	if err := ctx.Err(); err != nil {
		return employee.Employee{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.employees {
		if e.ID == id {
			if r.emailExists(in.Email, id) {
				return employee.Employee{}, employee.ErrConflict
			}
			e = employee.Employee{ID: id, Name: in.Name, Email: in.Email, Department: in.Department}
			r.employees[i] = e
			return e, nil
		}
	}
	return employee.Employee{}, employee.ErrNotFound
}

func (r *EmployeeRepository) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.employees {
		if e.ID == id {
			r.employees = append(r.employees[:i], r.employees[i+1:]...)
			return nil
		}
	}
	return employee.ErrNotFound
}

// Caller holds the lock so checking uniqueness and writing form one operation.
func (r *EmployeeRepository) emailExists(email string, except int64) bool {
	for _, e := range r.employees {
		if e.ID != except && strings.EqualFold(e.Email, email) {
			return true
		}
	}
	return false
}
