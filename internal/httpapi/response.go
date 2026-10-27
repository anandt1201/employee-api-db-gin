package httpapi

import "github.com/gin-gonic/gin"

// ErrorDetail describes a rejected or failed request.
type ErrorDetail struct {
	Code    string `json:"code" example:"not_found"`
	Message string `json:"message" example:"employee not found"`
}

// ErrorResponse is the common API error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// fail writes the shared API error envelope.
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}
