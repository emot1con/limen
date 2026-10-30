package handler

import (
	"math"
	"net/http"
	"strings"

	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	"limen/internal/domain/assessment"
	assessmentuc "limen/internal/usecase/assessment"

	"github.com/gin-gonic/gin"
)

type AssessmentHandler struct{ assessments *assessmentuc.UseCase }

func NewAssessmentHandler(assessments *assessmentuc.UseCase) *AssessmentHandler {
	return &AssessmentHandler{assessments: assessments}
}

func (h *AssessmentHandler) List(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	filter := assessment.AssessmentFilter{ClassID: c.Query("classId"), SubjectID: c.Query("subjectId"), Kind: strings.TrimSpace(c.Query("kind")), Period: strings.TrimSpace(c.Query("period"))}
	for key, value := range map[string]string{"classId": filter.ClassID, "subjectId": filter.SubjectID} {
		if value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	items, total, err := h.assessments.List(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = assessmentJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AssessmentHandler) Create(c *gin.Context) {
	var req struct {
		ClassID   string   `json:"classId"`
		SubjectID string   `json:"subjectId"`
		Kind      string   `json:"kind"`
		Period    string   `json:"period"`
		KKM       *float64 `json:"kkm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	var fields []response.FieldError
	if !ValidUUID(req.ClassID) {
		fields = append(fields, response.FieldError{Field: "classId", Message: "must be a UUID"})
	}
	if !ValidUUID(req.SubjectID) {
		fields = append(fields, response.FieldError{Field: "subjectId", Message: "must be a UUID"})
	}
	if strings.TrimSpace(req.Kind) == "" {
		fields = append(fields, response.FieldError{Field: "kind", Message: "required"})
	}
	if strings.TrimSpace(req.Period) == "" {
		fields = append(fields, response.FieldError{Field: "period", Message: "required"})
	}
	if req.KKM != nil && (math.IsNaN(*req.KKM) || math.IsInf(*req.KKM, 0) || *req.KKM < 0) {
		fields = append(fields, response.FieldError{Field: "kkm", Message: "must be zero or greater"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	teacherID, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	item, err := h.assessments.Create(c.Request.Context(), assessmentuc.CreateInput{ClassID: req.ClassID, SubjectID: req.SubjectID, Kind: req.Kind, Period: req.Period, KKM: req.KKM, TeacherUserID: teacherID})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assessmentJSON(item))
}

func (h *AssessmentHandler) ListScores(c *gin.Context) {
	assessmentID := c.Param("assessmentId")
	if !ValidUUID(assessmentID) {
		uuidError(c, "assessmentId")
		return
	}
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	items, total, err := h.assessments.ListScores(c.Request.Context(), assessmentID, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = scoreJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AssessmentHandler) UpsertScore(c *gin.Context) {
	assessmentID, studentID := c.Param("assessmentId"), c.Param("studentUserId")
	if !ValidUUID(assessmentID) {
		uuidError(c, "assessmentId")
		return
	}
	if !ValidUUID(studentID) {
		uuidError(c, "studentUserId")
		return
	}
	var req struct {
		Score *float64 `json:"score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Score == nil || math.IsNaN(*req.Score) || math.IsInf(*req.Score, 0) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "score", Message: "required finite number"})
		return
	}
	actor, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}

	item, err := h.assessments.UpsertScore(c.Request.Context(), assessmentID, studentID, actor, *req.Score)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, scoreJSON(item))
}

func assessmentJSON(item assessment.Assessment) gin.H {
	return gin.H{"id": item.ID, "classId": item.ClassID, "subjectId": item.SubjectID, "teacherUserId": item.TeacherUserID, "kind": item.Kind, "kkm": item.KKM, "period": item.Period}
}

func scoreJSON(item assessment.ScoreDetail) gin.H {
	return gin.H{"assessmentId": item.AssessmentID, "studentUserId": item.StudentUserID, "studentName": item.StudentName, "score": item.Score, "updatedBy": item.UpdatedBy, "updatedAt": item.UpdatedAt}
}
