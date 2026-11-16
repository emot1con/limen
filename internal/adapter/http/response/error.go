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
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details *ErrorDetails `json:"details,omitempty"`
}

type errorBody struct {
	Error *Error `json:"error"`
	Meta  Meta   `json:"meta"`
}

func TraceError(c *gin.Context, operation string, err error) {
	if err == nil {
		return
	}
	c.Set("internal_error_operation", operation)
	c.Set("internal_error_type", fmt.Sprintf("%T", err))
}

func AbortError(c *gin.Context, status int, code, message string, fields ...FieldError) {
	c.AbortWithStatusJSON(status, errorPayload(c, status, code, message, fields))
}

func WriteError(c *gin.Context, status int, code, message string, fields ...FieldError) {
	c.JSON(status, errorPayload(c, status, code, message, fields))
}

func errorPayload(c *gin.Context, status int, code, message string, fields []FieldError) errorBody {
	if status >= http.StatusInternalServerError {
		c.Set("internal_error_code", code)
	}
	body := errorBody{
		Error: &Error{Code: code, Message: message},
		Meta:  meta(c),
	}
	if len(fields) > 0 {
		body.Error.Details = &ErrorDetails{Fields: fields}
	}
	return body
}
