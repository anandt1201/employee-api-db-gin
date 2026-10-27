package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse reports process liveness.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// health reports process liveness.
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}
