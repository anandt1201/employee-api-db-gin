package httpapi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"example.com/employee-api-db-gin/internal/employee"
	"github.com/gin-gonic/gin"
)

func parseEmployeeListRequest(c *gin.Context) (employee.EmployeeListRequest, error) {
	request := employee.DefaultEmployeeListRequest()
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		return request, fmt.Errorf("%w: malformed query string", employee.ErrInvalid)
	}
	for _, setting := range []struct {
		name   string
		target *int
	}{
		{"page", &request.Page},
		{"page_size", &request.PageSize},
	} {
		if values, present := query[setting.name]; present {
			if len(values) != 1 {
				return request, fmt.Errorf("%w: %s must be supplied once", employee.ErrInvalid, setting.name)
			}
			value, err := strconv.Atoi(values[0])
			if err != nil {
				return request, fmt.Errorf("%w: %s must be an integer", employee.ErrInvalid, setting.name)
			}
			*setting.target = value
		}
	}
	for _, setting := range []struct {
		name   string
		target *string
	}{
		{"sort_by", &request.SortBy},
		{"sort_order", &request.SortOrder},
	} {
		if values, present := query[setting.name]; present {
			if len(values) != 1 {
				return request, fmt.Errorf("%w: %s must be supplied once", employee.ErrInvalid, setting.name)
			}
			*setting.target = strings.ToLower(strings.TrimSpace(values[0]))
		}
	}
	return request, request.Validate()
}
