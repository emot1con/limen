package handler

import (
	"math"
	"net/http"
	"strings"
	"time"

	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	assignmentdomain "limen/internal/domain/assignment"
	assignmentuc "limen/internal/usecase/assignment"

	"github.com/gin-gonic/gin"
)

type AssignmentHandler struct{ assignments *assignmentuc.UseCase }

func NewAssignmentHandler(assignments *assignmentuc.UseCase) *AssignmentHandler {
	return &AssignmentHandler{assignments: assignments}
}

func (h *AssignmentHandler) List(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	for _, key := range []string{"classId", "subjectId", "teacherUserId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	sort, ok := parseAssignmentSort(c.Query("sort"))
	if !ok {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "sort", Message: "must be one of createdAt, -createdAt, dueAt, -dueAt"})
		return
	}
	filter := assignmentdomain.AssignmentFilter{ClassID: c.Query("classId"), SubjectID: c.Query("subjectId"), TeacherUserID: c.Query("teacherUserId"), Sort: sort}
	items, total, err := h.assignments.List(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = assignmentTaskJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AssignmentHandler) Create(c *gin.Context) {
	var req struct {
		ClassID     string  `json:"classId"`
		SubjectID   string  `json:"subjectId"`
		Title       string  `json:"title"`
		Description *string `json:"description"`
		DueAt       string  `json:"dueAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	var fields []response.FieldError
	if !ValidUUID(req.ClassID) {
		fields = append(fields, response.FieldError{Field: "classId", Message: "must be a UUID"})
	}
	if !ValidUUID(req.SubjectID) {
		fields = append(fields, response.FieldError{Field: "subjectId", Message: "must be a UUID"})
	}
	if strings.TrimSpace(req.Title) == "" {
		fields = append(fields, response.FieldError{Field: "title", Message: "required"})
	}
	var dueAt time.Time
	if strings.TrimSpace(req.DueAt) == "" {
		fields = append(fields, response.FieldError{Field: "dueAt", Message: "required"})
	} else if parsed, err := time.Parse(time.RFC3339, req.DueAt); err != nil {
		fields = append(fields, response.FieldError{Field: "dueAt", Message: "must be an RFC3339 date-time"})
	} else {
		dueAt = parsed
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
	item, err := h.assignments.Create(c.Request.Context(), assignmentuc.CreateInput{ClassID: req.ClassID, SubjectID: req.SubjectID, TeacherUserID: teacherID, Title: req.Title, Description: req.Description, DueAt: dueAt})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assignmentTaskJSON(item))
}

func (h *AssignmentHandler) ListSubmissions(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	if !ValidUUID(assignmentID) {
		uuidError(c, "assignmentId")
		return
	}
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	items, total, err := h.assignments.ListSubmissions(c.Request.Context(), assignmentID, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = submissionJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AssignmentHandler) GradeSubmission(c *gin.Context) {
	assignmentID, studentID := c.Param("assignmentId"), c.Param("studentUserId")
	if !ValidUUID(assignmentID) {
		uuidError(c, "assignmentId")
		return
	}
	if !ValidUUID(studentID) {
		uuidError(c, "studentUserId")
		return
	}
	var req struct {
		Score *float64 `json:"score"`
		Note  *string  `json:"note"`
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
	item, err := h.assignments.GradeSubmission(c.Request.Context(), assignmentID, studentID, actor, *req.Score, req.Note)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, submissionJSON(item))
}

func parseAssignmentSort(raw string) (assignmentdomain.AssignmentSort, bool) {
	switch strings.TrimSpace(raw) {
	case "", "-createdAt":
		return assignmentdomain.AssignmentSortNewest, true
	case "createdAt":
		return assignmentdomain.AssignmentSortOldest, true
	case "dueAt":
		return assignmentdomain.AssignmentSortDueAsc, true
	case "-dueAt":
		return assignmentdomain.AssignmentSortDueDesc, true
	default:
		return "", false
	}
}

func assignmentTaskJSON(item assignmentdomain.Assignment) gin.H {
	return gin.H{"id": item.ID, "classId": item.ClassID, "subjectId": item.SubjectID, "teacherUserId": item.TeacherUserID, "title": item.Title, "description": item.Description, "dueAt": item.DueAt}
}

func submissionJSON(item assignmentdomain.SubmissionDetail) gin.H {
	return gin.H{"studentUserId": item.StudentUserID, "studentName": item.StudentName, "submittedAt": item.SubmittedAt, "score": item.Score, "note": item.Note, "gradedBy": item.GradedBy, "gradedAt": item.GradedAt}
}
