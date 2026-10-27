package httpapi

import (
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes adds the employee endpoints beneath the supplied API group.
func (h *EmployeeHandler) RegisterRoutes(api *gin.RouterGroup) {
	employees := api.Group("/employees")
	employees.GET("", RateLimit(10, 5*time.Second), h.list)
	employees.GET("/:id", h.get)
	employees.POST("", h.create)
	employees.PUT("/:id", h.update)
	employees.DELETE("/:id", h.delete)
}
