package httpapi

import (
	_ "example.com/employee-api-db-gin/internal/apidocs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Regenerate committed documentation from annotations with go generate ./internal/httpapi.
//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d ../../cmd/api,.,../employee --parseInternal -o ../apidocs

func registerSwagger(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("doc.json"),
		ginSwagger.PersistAuthorization(false),
	))
}
