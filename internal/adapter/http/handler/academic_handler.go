package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"limen/internal/adapter/http/response"
	academicdomain "limen/internal/domain/academic"
	academicuc "limen/internal/usecase/academic"

	"github.com/gin-gonic/gin"
)

type AcademicHandler struct {
	academic *academicuc.UseCase
}

func NewAcademicHandler(academic *academicuc.UseCase) *AcademicHandler {
	return &AcademicHandler{academic: academic}
}

func (h *AcademicHandler) ListClasses(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	items, total, err := h.academic.ListClasses(c.Request.Context(), c.Query("level"), c.Query("q"), page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = classJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AcademicHandler) CreateClass(c *gin.Context) {
	var req struct {
		Name                  string  `json:"name"`
		Level                 *string `json:"level"`
		HomeroomTeacherUserID *string `json:"homeroomTeacherUserId"`
	}
	if !bindAcademic(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "name", Message: "required"})
		return
	}
	if req.HomeroomTeacherUserID != nil && !ValidUUID(*req.HomeroomTeacherUserID) {
		uuidError(c, "homeroomTeacherUserId")
		return
	}

	fmt.Println("req after: ", req.Name, req.Level, req.HomeroomTeacherUserID)
	out, err := h.academic.CreateClass(c.Request.Context(), req.Name, req.Level, req.HomeroomTeacherUserID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, classJSON(academicdomain.ClassDetail{Class: out}))
}

func (h *AcademicHandler) UpdateClass(c *gin.Context) {
	if !ValidUUID(c.Param("classId")) {
		uuidError(c, "classId")
		return
	}
	var raw map[string]json.RawMessage
	if !bindAcademic(c, &raw) {
		return
	}
	var req struct {
		Name                  *string `json:"name"`
		Level                 *string `json:"level"`
		HomeroomTeacherUserID *string `json:"homeroomTeacherUserId"`
	}
	body, _ := json.Marshal(raw)
	if err := json.Unmarshal(body, &req); err != nil {
		validationBody(c)
		return
	}
	_, teacherSet := raw["homeroomTeacherUserId"]
	if teacherSet && req.HomeroomTeacherUserID != nil && !ValidUUID(*req.HomeroomTeacherUserID) {
		uuidError(c, "homeroomTeacherUserId")
		return
	}
	out, err := h.academic.UpdateClass(c.Request.Context(), c.Param("classId"), req.Name, req.Level, req.HomeroomTeacherUserID, teacherSet)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, classJSON(academicdomain.ClassDetail{Class: out}))
}

func (h *AcademicHandler) ListSubjects(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	items, total, err := h.academic.ListSubjects(c.Request.Context(), c.Query("q"), page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = subjectJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AcademicHandler) CreateSubject(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if !bindAcademic(c, &req) {
		return
	}
	var fields []response.FieldError
	if strings.TrimSpace(req.Code) == "" {
		fields = append(fields, response.FieldError{Field: "code", Message: "required"})
	}
	if strings.TrimSpace(req.Name) == "" {
		fields = append(fields, response.FieldError{Field: "name", Message: "required"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	out, err := h.academic.CreateSubject(c.Request.Context(), req.Code, req.Name)
	if err != nil {
		writeMappedError(c, err, "code")
		return
	}
	c.JSON(http.StatusCreated, subjectJSON(out))
}

func (h *AcademicHandler) ListStudents(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	classID := c.Query("classId")
	if classID != "" && !ValidUUID(classID) {
		uuidError(c, "classId")
		return
	}
	sort := c.Query("sort")
	if sort != "" && sort != "name" && sort != "nis" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "sort", Message: "must be name or nis"})
		return
	}
	items, total, err := h.academic.ListStudents(c.Request.Context(), classID, c.Query("q"), sort, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = studentJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AcademicHandler) GetStudent(c *gin.Context) {
	if !ValidUUID(c.Param("studentId")) {
		uuidError(c, "studentId")
		return
	}
	out, err := h.academic.GetStudent(c.Request.Context(), c.Param("studentId"))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, studentJSON(out))
}

func (h *AcademicHandler) UpdateStudent(c *gin.Context) {
	if !ValidUUID(c.Param("studentId")) {
		uuidError(c, "studentId")
		return
	}
	var raw map[string]json.RawMessage
	if !bindAcademic(c, &raw) {
		return
	}
	var req struct {
		NIS     *string `json:"nis"`
		ClassID *string `json:"classId"`
	}
	body, _ := json.Marshal(raw)
	if err := json.Unmarshal(body, &req); err != nil {
		validationBody(c)
		return
	}
	_, nisSet := raw["nis"]
	_, classSet := raw["classId"]
	if !nisSet && !classSet {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "at least one field is required"})
		return
	}
	if nisSet && (req.NIS == nil || strings.TrimSpace(*req.NIS) == "") {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "nis", Message: "required"})
		return
	}
	if classSet && req.ClassID != nil && !ValidUUID(*req.ClassID) {
		uuidError(c, "classId")
		return
	}
	out, err := h.academic.UpdateStudent(c.Request.Context(), c.Param("studentId"), req.NIS, req.ClassID, nisSet, classSet)
	if err != nil {
		writeMappedError(c, err, "nis")
		return
	}
	c.JSON(http.StatusOK, studentJSON(out))
}

func (h *AcademicHandler) ListAssignments(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	for _, key := range []string{"teacherUserId", "classId", "subjectId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	items, total, err := h.academic.ListAssignments(c.Request.Context(), c.Query("teacherUserId"), c.Query("classId"), c.Query("subjectId"), page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = assignmentJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *AcademicHandler) CreateAssignment(c *gin.Context) {
	var req struct {
		TeacherUserID string `json:"teacherUserId"`
		SubjectID     string `json:"subjectId"`
		ClassID       string `json:"classId"`
	}
	if !bindAcademic(c, &req) {
		return
	}
	var fields []response.FieldError
	for key, value := range map[string]string{"teacherUserId": req.TeacherUserID, "subjectId": req.SubjectID, "classId": req.ClassID} {
		if !ValidUUID(value) {
			fields = append(fields, response.FieldError{Field: key, Message: "must be a UUID"})
		}
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	out, err := h.academic.CreateAssignment(c.Request.Context(), req.TeacherUserID, req.SubjectID, req.ClassID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assignmentJSON(out))
}

func bindAcademic(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		validationBody(c)
		return false
	}
	return true
}

func validationBody(c *gin.Context) {
	response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
}

func uuidError(c *gin.Context, field string) {
	response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: field, Message: "must be a UUID"})
}

func classJSON(item academicdomain.ClassDetail) gin.H {
	return gin.H{"id": item.ID, "name": item.Name, "level": item.Level, "homeroomTeacherUserId": item.HomeroomTeacherUserID, "studentCount": item.StudentCount}
}

func subjectJSON(item academicdomain.Subject) gin.H {
	return gin.H{"id": item.ID, "code": item.Code, "name": item.Name}
}

func studentJSON(item academicdomain.StudentDetail) gin.H {
	return gin.H{"userId": item.UserID, "fullName": item.FullName, "nis": item.NIS, "classId": item.ClassID, "className": item.ClassName, "status": item.Status}
}

func assignmentJSON(item academicdomain.TeachingAssignment) gin.H {
	return gin.H{"id": item.ID, "teacherUserId": item.TeacherUserID, "subjectId": item.SubjectID, "classId": item.ClassID}
}
