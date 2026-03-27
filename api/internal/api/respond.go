package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/gin-gonic/gin"
)

// respondError sends a JSON error response.
func respondError(c *gin.Context, status int, message string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	c.JSON(status, models.ErrorResponseMsg(message, errStr))
}

// respondBadRequest sends a 400 JSON error response.
func respondBadRequest(c *gin.Context, message string, err error) {
	respondError(c, http.StatusBadRequest, message, err)
}

// respondInternalError sends a 500 JSON error response.
func respondInternalError(c *gin.Context, message string, err error) {
	respondError(c, http.StatusInternalServerError, message, err)
}

// respondSuccess sends a 200 JSON success response.
func respondSuccess(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, models.SuccessResponse(data, message))
}

// PaginationParams holds parsed query params for paginated endpoints.
type PaginationParams struct {
	Page      int
	Limit     int
	Search    string
	SortBy    string
	SortOrder string
}

// parsePagination reads page/limit/search/sort/order from the query string.
// Caller provides defaults for sort field and order.
func parsePagination(c *gin.Context, defaultSort, defaultOrder string) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 24
	}
	return PaginationParams{
		Page:      page,
		Limit:     limit,
		Search:    c.Query("search"),
		SortBy:    c.DefaultQuery("sort", defaultSort),
		SortOrder: c.DefaultQuery("order", defaultOrder),
	}
}
