// Package repository implements storage adapters for domain repository contracts.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/employee-api-db-gin/internal/employee"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmployeeRepository persists employees through GORM.
type EmployeeRepository struct {
	db           *gorm.DB
	queryTimeout time.Duration
}

var _ employee.Repository = (*EmployeeRepository)(nil)

// NewEmployeeRepository creates a repository using the supplied GORM connection.
// The caller owns and closes the underlying connection pool.
func NewEmployeeRepository(db *gorm.DB, queryTimeout time.Duration) *EmployeeRepository {
	return &EmployeeRepository{db: db, queryTimeout: queryTimeout}
}

func (r *EmployeeRepository) List(ctx context.Context, request employee.EmployeeListRequest) (employee.EmployeePage, error) {
	if err := request.Validate(); err != nil {
		return employee.EmployeePage{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	var total int64
	if err := r.db.WithContext(ctx).Model(&employeeRecord{}).Count(&total).Error; err != nil {
		return employee.EmployeePage{}, fmt.Errorf("count employees: %w", err)
	}
	// Never pass user text directly to Order: the field is validated above and
	// emitted as a quoted column, while direction is represented by a boolean.
	query := r.db.WithContext(ctx).Order(clause.OrderByColumn{
		Column: clause.Column{Name: request.SortBy},
		Desc:   request.SortOrder == "desc",
	})
	if request.SortBy != "id" {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}})
	}
	records := make([]employeeRecord, 0)
	offset := (request.Page - 1) * request.PageSize
	if err := query.Limit(request.PageSize).Offset(offset).Find(&records).Error; err != nil {
		return employee.EmployeePage{}, fmt.Errorf("list employees: %w", err)
	}
	items := make([]employee.Employee, len(records))
	for i, record := range records {
		items[i] = record.toEmployee()
	}
	return employee.EmployeePage{Items: items, TotalItems: total}, nil
}
func (r *EmployeeRepository) Get(ctx context.Context, id int64) (employee.Employee, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	var record employeeRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return employee.Employee{}, mapRepositoryError(err)
	}
	return record.toEmployee(), nil
}

func (r *EmployeeRepository) Create(ctx context.Context, request employee.EmployeeRequest) (employee.Employee, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	record := employeeRecord{Name: request.Name, Email: request.Email, Department: request.Department}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return employee.Employee{}, mapRepositoryError(err)
	}
	return record.toEmployee(), nil
}

func (r *EmployeeRepository) Update(ctx context.Context, id int64, request employee.EmployeeRequest) (employee.Employee, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	record := employeeRecord{ID: id, Name: request.Name, Email: request.Email, Department: request.Department}
	// Explicit fields preserve full-replacement semantics, including zero values.
	// Updates never falls back to INSERT when the requested employee is missing.
	result := r.db.WithContext(ctx).Model(&record).
		Select("name", "email", "department").
		Clauses(clause.Returning{}).
		Updates(&record)
	if result.Error != nil {
		return employee.Employee{}, mapRepositoryError(result.Error)
	}
	if result.RowsAffected == 0 {
		return employee.Employee{}, employee.ErrNotFound
	}
	return record.toEmployee(), nil
}

func (r *EmployeeRepository) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&employeeRecord{})
	if result.Error != nil {
		return mapRepositoryError(result.Error)
	}
	if result.RowsAffected == 0 {
		return employee.ErrNotFound
	}
	return nil
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return employee.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		// Email is the only client-supplied unique key in the current schema.
		return employee.ErrConflict
	default:
		return fmt.Errorf("employee storage operation: %w", err)
	}
}
