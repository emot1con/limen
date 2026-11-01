package handler

import (
	"net/http"
	"strings"

	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	counselingdomain "limen/internal/domain/counseling"
	counselinguc "limen/internal/usecase/counseling"

	"github.com/gin-gonic/gin"
)

type CounselingHandler struct{ counseling *counselinguc.UseCase }

func NewCounselingHandler(counseling *counselinguc.UseCase) *CounselingHandler {
	return &CounselingHandler{counseling: counseling}
}

func (h *CounselingHandler) ListNotes(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	for _, key := range []string{"studentUserId", "classId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	sort, ok := parseNoteSort(c.Query("sort"))
	if !ok {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "sort", Message: "must be createdAt or -createdAt"})
		return
	}
	filter := counselingdomain.CounselingNoteFilter{
		StudentUserID: c.Query("studentUserId"),
		ClassID:       c.Query("classId"),
		Sort:          sort,
	}
	items, total, err := h.counseling.ListNotes(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = counselingNoteJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *CounselingHandler) CreateNote(c *gin.Context) {
	var req struct {
		StudentUserID string `json:"studentUserId"`
		Note          string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	var fields []response.FieldError
	if !ValidUUID(req.StudentUserID) {
		fields = append(fields, response.FieldError{Field: "studentUserId", Message: "must be a UUID"})
	}
	if strings.TrimSpace(req.Note) == "" {
		fields = append(fields, response.FieldError{Field: "note", Message: "required"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	authorID, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	item, err := h.counseling.CreateNote(c.Request.Context(), req.StudentUserID, authorID, req.Note)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, counselingNoteJSON(item))
}

func (h *CounselingHandler) UpdateNote(c *gin.Context) {
	noteID := c.Param("noteId")
	if !ValidUUID(noteID) {
		uuidError(c, "noteId")
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	if strings.TrimSpace(req.Note) == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "note", Message: "required"})
		return
	}
	item, err := h.counseling.UpdateNote(c.Request.Context(), noteID, req.Note)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, counselingNoteJSON(item))
}

func (h *CounselingHandler) AddFollowup(c *gin.Context) {
	noteID := c.Param("noteId")
	if !ValidUUID(noteID) {
		uuidError(c, "noteId")
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	if strings.TrimSpace(req.Note) == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "note", Message: "required"})
		return
	}
	createdBy, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	item, err := h.counseling.AddFollowup(c.Request.Context(), noteID, createdBy, req.Note)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, followupJSON(item))
}

func parseNoteSort(raw string) (counselingdomain.CounselingNoteSort, bool) {
	switch strings.TrimSpace(raw) {
	case "", "-createdAt":
		return counselingdomain.CounselingNoteSortNewest, true
	case "createdAt":
		return counselingdomain.CounselingNoteSortOldest, true
	default:
		return "", false
	}
}

func counselingNoteJSON(item counselingdomain.CounselingNote) gin.H {
	return gin.H{"id": item.ID, "studentUserId": item.StudentUserID, "authorId": item.AuthorID, "note": item.Note, "createdAt": item.CreatedAt}
}

func followupJSON(item counselingdomain.CounselingFollowup) gin.H {
	return gin.H{"id": item.ID, "noteId": item.NoteID, "note": item.Note, "createdBy": item.CreatedBy, "createdAt": item.CreatedAt}
}
