package common

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"limen/internal/adapter/http/response"
	"limen/internal/domain"

	"github.com/gin-gonic/gin"
)

func PageQuery(c *gin.Context) (int, int, bool) {
	page := 1
	pageSize := 20
	if raw := c.Query("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "page", Message: "must be 1 or greater"})
			return 0, 0, false
		}
		page = value
	}
	if raw := c.Query("pageSize"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "pageSize", Message: "must be between 1 and 100"})
			return 0, 0, false
		}
		pageSize = value
	}
	return page, pageSize, true
}

func WriteMappedError(c *gin.Context, err error, duplicateField ...string) {
	if !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, domain.ErrAlreadyExists) && !errors.Is(err, domain.ErrInvalid) && !errors.Is(err, domain.ErrConflict) && !IsUniqueViolation(err) {
		response.TraceError(c, "handler.use_case", err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		response.WriteError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, domain.ErrAlreadyExists) || IsUniqueViolation(err) {
		field := "request"
		if len(duplicateField) > 0 {
			field = duplicateField[0]
		}
		response.WriteError(c, http.StatusConflict, "CONFLICT", "resource already exists", response.FieldError{Field: field, Message: "already exists"})
		return
	}
	if errors.Is(err, domain.ErrConflict) {
		response.WriteError(c, http.StatusConflict, "CONFLICT", "resource is in use")
		return
	}
	if errors.Is(err, domain.ErrInvalid) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed")
		return
	}
	response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func IsUniqueViolation(err error) bool {
	message := err.Error()
	return strings.Contains(message, "23505") || strings.Contains(message, "duplicate key")
}

func ValidUUID(value string) bool {
	matched, _ := regexp.MatchString(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, value)
	return matched
}

func ValidationBody(c *gin.Context) {
	response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
}

func UUIDError(c *gin.Context, field string) {
	response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: field, Message: "must be a UUID"})
}

func BindJSON(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		ValidationBody(c)
		return false
	}
	return true
}
