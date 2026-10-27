package httpapi

import (
	"log/slog"
	"strconv"

	"example.com/employee-api-db-gin/internal/employee"

	"github.com/gin-gonic/gin"
)

type EmployeeHandler struct {
	service *employee.Service
	log     *slog.Logger
}

func NewEmployeeHandler(service *employee.Service, log *slog.Logger) *EmployeeHandler {
	return &EmployeeHandler{service: service, log: log}
}

// list returns a bounded, sorted page of employees.
// @Summary List employees with pagination and sorting
// @Description Returns a bounded page. Defaults: page=1, page_size=20, sort_by=id, sort_order=asc. Equal sort values use ID ascending as a tiebreaker. Maximum 10 requests per IP per rolling 5 seconds.
// @Tags employees
// @Produce json
// @Param page query int false "Page number" default(1) minimum(1) maximum(1000000)
// @Param page_size query int false "Records per page" default(20) minimum(1) maximum(100)
// @Param sort_by query string false "Sort field" default(id) Enums(id,name,email,department)
// @Param sort_order query string false "Sort direction" default(asc) Enums(asc,desc)
// @Success 200 {object} EmployeeListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Header 429 {integer} Retry-After "Seconds until the next request is allowed, rounded up"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees [get]
func (h *EmployeeHandler) list(c *gin.Context) {
	request, err := parseEmployeeListRequest(c)
	if err != nil {
		h.respondError(c, err)
		return
	}
	page, err := h.service.List(c.Request.Context(), request)
	if err != nil {
		h.respondError(c, err)
		return
	}
	responses := make([]EmployeeResponse, len(page.Items))
	for i, item := range page.Items {
		responses[i] = newEmployeeResponse(item)
	}
	totalPages := page.TotalItems / int64(request.PageSize)
	if page.TotalItems%int64(request.PageSize) != 0 {
		totalPages++
	}
	c.JSON(200, EmployeeListResponse{
		Data:       responses,
		Pagination: PaginationResponse{Page: request.Page, PageSize: request.PageSize, TotalItems: page.TotalItems, TotalPages: totalPages},
	})
}

// get returns an employee by ID.
// @Summary Get employee
// @Tags employees
// @Produce json
// @Param id path int true "Positive employee ID" minimum(1)
// @Success 200 {object} EmployeeDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id} [get]
func (h *EmployeeHandler) get(c *gin.Context) {
	n, ok := parseID(c)
	if !ok {
		return
	}
	e, err := h.service.Get(c.Request.Context(), n)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(200, EmployeeDataResponse{Data: newEmployeeResponse(e)})
}

// create creates an employee.
// @Summary Create employee
// @Description Requires name, email, and department. Email is normalized and unique. Request body limit is 1 MiB.
// @Tags employees
// @Accept json
// @Produce json
// @Param employee body employee.EmployeeRequest true "Employee to create"
// @Success 201 {object} EmployeeDataResponse
// @Header 201 {string} Location "Relative URL of the created employee"
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 415 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees [post]
func (h *EmployeeHandler) create(c *gin.Context) {
	in, ok := decodeJSON[employee.EmployeeRequest](c)
	if !ok {
		return
	}
	e, err := h.service.Create(c.Request.Context(), in)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Header("Location", "/api/v1/employees/"+strconv.FormatInt(e.ID, 10))
	c.JSON(201, EmployeeDataResponse{Data: newEmployeeResponse(e)})
}

// update replaces an employee.
// @Summary Replace employee
// @Description Full replacement: name, email, and department are required. Request body limit is 1 MiB.
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Positive employee ID" minimum(1)
// @Param employee body employee.EmployeeRequest true "Replacement employee"
// @Success 200 {object} EmployeeDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 415 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id} [put]
func (h *EmployeeHandler) update(c *gin.Context) {
	n, ok := parseID(c)
	if !ok {
		return
	}
	in, ok := decodeJSON[employee.EmployeeRequest](c)
	if !ok {
		return
	}
	e, err := h.service.Update(c.Request.Context(), n, in)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(200, EmployeeDataResponse{Data: newEmployeeResponse(e)})
}

// delete removes an employee.
// @Summary Delete employee
// @Tags employees
// @Produce json
// @Param id path int true "Positive employee ID" minimum(1)
// @Success 204 "Employee deleted; no response body"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id} [delete]
func (h *EmployeeHandler) delete(c *gin.Context) {
	n, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), n); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(204)
}
