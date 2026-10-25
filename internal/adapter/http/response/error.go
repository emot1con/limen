package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorDetails struct {
	Fields []FieldError `json:"fields"`
}

type Error struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details ErrorDetails `json:"details"`
}

func TraceError(c *gin.Context, operation string, err error) {
	if err == nil {
		return
	}
	c.Set("internal_error_operation", operation)
	c.Set("internal_error_type", fmt.Sprintf("%T", err))
}

func AbortError(c *gin.Context, status int, code, message string, fields ...FieldError) {
	if fields == nil {
		fields = []FieldError{}
	}
	c.AbortWithStatusJSON(status, Error{
		Code:    code,
		Message: message,
		Details: ErrorDetails{Fields: fields},
	})
}

func WriteError(c *gin.Context, status int, code, message string, fields ...FieldError) {
	if status >= http.StatusInternalServerError {
		c.Set("internal_error_code", code)
	}
	if fields == nil {
		fields = []FieldError{}
	}
	c.JSON(status, Error{
		Code:    code,
		Message: message,
		Details: ErrorDetails{Fields: fields},
	})
}
