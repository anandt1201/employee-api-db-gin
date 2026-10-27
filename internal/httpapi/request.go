package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseID validates the common positive integer resource ID parameter.
func parseID(c *gin.Context) (int64, bool) {
	n, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || n <= 0 {
		fail(c, 400, "invalid_id", "id must be a positive integer")
		return 0, false
	}
	return n, true
}

// decodeJSON applies consistent JSON decoding rules to request types.
func decodeJSON[T any](c *gin.Context) (T, bool) {
	var in *T
	if c.ContentType() != "application/json" {
		fail(c, 415, "unsupported_media_type", "Content-Type must be application/json")
		return *new(T), false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil {
		writeDecodeError(c, err)
		return *new(T), false
	}
	if in == nil {
		fail(c, http.StatusBadRequest, "invalid_json", "body must be a JSON object")
		return *new(T), false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeDecodeError(c, err)
		return *new(T), false
	}
	return *in, true
}

func writeDecodeError(c *gin.Context, err error) {
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		fail(c, http.StatusRequestEntityTooLarge, "payload_too_large", "body must not exceed 1 MiB")
		return
	}
	fail(c, http.StatusBadRequest, "invalid_json", "body must contain exactly one valid JSON object with supported fields")
}
