package httpapi

import (
	"log/slog"

	"example.com/employee-api-db-gin/internal/employee"

	"github.com/gin-gonic/gin"
)

// NewRouter configures shared middleware and registers each resource's routes.
func NewRouter(service *employee.Service, log *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	// Direct local requests only; do not trust arbitrary forwarded client IPs.
	_ = router.SetTrustedProxies(nil)

	registerSwagger(router)

	router.GET("/healthz", health)
	api := router.Group("/api/v1")
	NewEmployeeHandler(service, log).RegisterRoutes(api)

	router.NoRoute(func(c *gin.Context) {
		fail(c, 404, "not_found", "route not found")
	})
	return router
}
