package httpapi

import (
	"errors"

	"example.com/employee-api-db-gin/internal/employee"

	"github.com/gin-gonic/gin"
)

func (h *EmployeeHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, employee.ErrNotFound):
		fail(c, 404, "not_found", "employee not found")
	case errors.Is(err, employee.ErrConflict):
		fail(c, 409, "conflict", "email already exists")
	case errors.Is(err, employee.ErrInvalid):
		fail(c, 400, "validation_error", err.Error())
	default:
		h.log.ErrorContext(c.Request.Context(), "request failed", "error", err)
		fail(c, 500, "internal_error", "internal server error")
	}
}
