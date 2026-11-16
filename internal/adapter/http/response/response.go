package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Meta struct {
	RequestID string `json:"requestId"`
}

type PageMeta struct {
	RequestID  string `json:"requestId"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"totalPages"`
}

type successBody struct {
	Data any `json:"data"`
	Meta any `json:"meta"`
}

func WriteOK(c *gin.Context, data any) {
	writeSuccess(c, http.StatusOK, data, meta(c))
}

func WriteCreated(c *gin.Context, data any) {
	writeSuccess(c, http.StatusCreated, data, meta(c))
}

func WriteList(c *gin.Context, items any, page, pageSize int, total int64) {
	if items == nil {
		items = []any{}
	}
	writeSuccess(c, http.StatusOK, items, PageMeta{
		RequestID:  requestID(c),
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
	})
}

func WriteNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func writeSuccess(c *gin.Context, status int, data, meta any) {
	c.JSON(status, successBody{Data: data, Meta: meta})
}

func meta(c *gin.Context) Meta {
	return Meta{RequestID: requestID(c)}
}

func requestID(c *gin.Context) string {
	return c.GetString("request_id")
}

func totalPages(total int64, pageSize int) int {
	if pageSize <= 0 || total <= 0 {
		return 0
	}
	pages := (total + int64(pageSize) - 1) / int64(pageSize)
	if pages > int64(^uint(0)>>1) {
		return int(^uint(0) >> 1)
	}
	return int(pages)
}
