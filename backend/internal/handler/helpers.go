package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// PaginatedResponse wraps list data with pagination metadata.
type PaginatedResponse struct {
	Data       interface{}        `json:"data"`
	Pagination PaginationMetadata `json:"pagination"`
}

// PaginationMetadata holds page-level info for list endpoints.
type PaginationMetadata struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// Success responds with 200 and a data wrapper.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Created responds with 201 and a data wrapper.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// Error responds with the given status and an error message.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// Paginated responds with 200 and wraps data + pagination metadata.
func Paginated(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	c.JSON(http.StatusOK, PaginatedResponse{
		Data: data,
		Pagination: PaginationMetadata{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// Pagination holds parsed page/pageSize/offset values.
type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

// GetPagination extracts pagination parameters from query string with defaults.
func GetPagination(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Pagination{Page: page, PageSize: pageSize, Offset: (page - 1) * pageSize}
}

// parseUUID converts a string UUID into a pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID: %s", s)
	}
	return u, nil
}
