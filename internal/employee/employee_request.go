package employee

// EmployeeRequest is deliberately separate from Employee: clients cannot choose an ID.
type EmployeeRequest struct {
	Name       string `json:"name" binding:"required" example:"Anand"`
	Email      string `json:"email" binding:"required" example:"anand@example.com"`
	Department string `json:"department" binding:"required" example:"Engineering"`
}
