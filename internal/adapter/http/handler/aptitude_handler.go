package handler

import (
	"net/http"
	"strconv"
	"strings"

	"limen/internal/adapter/http/response"
	counselingdomain "limen/internal/domain/counseling"
	aptitudeuc "limen/internal/usecase/aptitude"

	"github.com/gin-gonic/gin"
)

type AptitudeHandler struct{ aptitudes *aptitudeuc.UseCase }

func NewAptitudeHandler(aptitudes *aptitudeuc.UseCase) *AptitudeHandler {
	return &AptitudeHandler{aptitudes: aptitudes}
}

func (h *AptitudeHandler) ListAptitudeResults(c *gin.Context) {
	studentID := c.Param("studentId")
	if !ValidUUID(studentID) {
		uuidError(c, "studentId")
		return
	}
	items, err := h.aptitudes.ListAptitudeResults(c.Request.Context(), studentID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = aptitudeResultJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"studentUserId": studentID, "results": out})
}

func (h *AptitudeHandler) CreateTargetMajor(c *gin.Context) {
	studentID := c.Param("studentId")
	if !ValidUUID(studentID) {
		uuidError(c, "studentId")
		return
	}
	var req struct {
		Major string `json:"major"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	if strings.TrimSpace(req.Major) == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "major", Message: "required"})
		return
	}
	item, err := h.aptitudes.AddTargetMajor(c.Request.Context(), studentID, req.Major)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, targetMajorJSON(item))
}

func (h *AptitudeHandler) ListPassingGrades(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	filter := counselingdomain.PassingGradeFilter{University: c.Query("university"), Major: c.Query("major")}
	if raw := c.Query("year"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "year", Message: "must be an integer"})
			return
		}
		filter.Year, filter.YearSet = value, true
	}
	items, total, err := h.aptitudes.ListPassingGrades(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = passingGradeJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AptitudeHandler) GetPassingGrade(c *gin.Context) {
	id := c.Param("passingGradeId")
	if !ValidUUID(id) {
		uuidError(c, "passingGradeId")
		return
	}
	item, err := h.aptitudes.GetPassingGrade(c.Request.Context(), id)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, passingGradeJSON(item))
}

func aptitudeResultJSON(item counselingdomain.AptitudeResult) gin.H {
	return gin.H{"id": item.ID, "result": item.Result, "assessedAt": item.AssessedAt}
}

func targetMajorJSON(item counselingdomain.TargetMajor) gin.H {
	return gin.H{"id": item.ID, "studentUserId": item.StudentUserID, "major": item.Major, "createdAt": item.CreatedAt}
}

func passingGradeJSON(item counselingdomain.PassingGrade) gin.H {
	return gin.H{"id": item.ID, "university": item.University, "major": item.Major, "score": item.Score, "year": item.Year}
}
