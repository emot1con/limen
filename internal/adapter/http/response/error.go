package response

import "github.com/gin-gonic/gin"

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
	if fields == nil {
		fields = []FieldError{}
	}
	c.JSON(status, Error{
		Code:    code,
		Message: message,
		Details: ErrorDetails{Fields: fields},
	})
}
