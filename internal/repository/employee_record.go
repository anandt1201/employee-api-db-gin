package repository

import "example.com/employee-api-db-gin/internal/employee"

// employeeRecord keeps ORM mapping separate from the domain and API models.
// No gorm.Model: the existing schema has no timestamps or soft-delete column.
type employeeRecord struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name       string `gorm:"column:name"`
	Email      string `gorm:"column:email"`
	Department string `gorm:"column:department"`
}

func (employeeRecord) TableName() string {
	return "employees"
}

func (r employeeRecord) toEmployee() employee.Employee {
	return employee.Employee{ID: r.ID, Name: r.Name, Email: r.Email, Department: r.Department}
}
