package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	quizdomain "limen/internal/domain/quiz"
	questionbankuc "limen/internal/usecase/questionbank"

	"github.com/gin-gonic/gin"
)

type QuestionBankHandler struct{ questionbank *questionbankuc.UseCase }

func NewQuestionBankHandler(questionbank *questionbankuc.UseCase) *QuestionBankHandler {
	return &QuestionBankHandler{questionbank: questionbank}
}

func (h *QuestionBankHandler) ListBanks(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	for _, key := range []string{"subjectId", "ownerId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	filter := quizdomain.QuestionBankFilter{SubjectID: c.Query("subjectId"), OwnerID: c.Query("ownerId")}
	if raw := strings.TrimSpace(c.Query("scope")); raw != "" {
		if raw != string(quizdomain.QuestionBankScopePractice) && raw != string(quizdomain.QuestionBankScopeExam) {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "scope", Message: "must be latihan or ujian"})
			return
		}
		filter.Scope = quizdomain.QuestionBankScope(raw)
	}
	items, total, err := h.questionbank.ListBanks(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = questionBankJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *QuestionBankHandler) CreateBank(c *gin.Context) {
	var req struct {
		SubjectID string `json:"subjectId"`
		Scope     string `json:"scope"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	var fields []response.FieldError
	if !ValidUUID(req.SubjectID) {
		fields = append(fields, response.FieldError{Field: "subjectId", Message: "must be a UUID"})
	}
	scope := quizdomain.QuestionBankScope(strings.TrimSpace(req.Scope))
	if scope != quizdomain.QuestionBankScopePractice && scope != quizdomain.QuestionBankScopeExam {
		fields = append(fields, response.FieldError{Field: "scope", Message: "must be latihan or ujian"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	ownerID, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	item, err := h.questionbank.CreateBank(c.Request.Context(), req.SubjectID, ownerID, scope)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, questionBankJSON(item))
}

func (h *QuestionBankHandler) ListQuestions(c *gin.Context) {
	bankID := c.Param("bankId")
	if !ValidUUID(bankID) {
		uuidError(c, "bankId")
		return
	}
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	items, total, err := h.questionbank.ListQuestions(c.Request.Context(), bankID, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = questionJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *QuestionBankHandler) CreateQuestion(c *gin.Context) {
	bankID := c.Param("bankId")
	if !ValidUUID(bankID) {
		uuidError(c, "bankId")
		return
	}
	input, ok := bindQuestionInput(c)
	if !ok {
		return
	}
	item, err := h.questionbank.CreateQuestion(c.Request.Context(), bankID, input)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, questionJSON(item))
}

func (h *QuestionBankHandler) UpdateQuestion(c *gin.Context) {
	questionID := c.Param("questionId")
	if !ValidUUID(questionID) {
		uuidError(c, "questionId")
		return
	}
	input, ok := bindQuestionInput(c)
	if !ok {
		return
	}
	item, err := h.questionbank.UpdateQuestion(c.Request.Context(), questionID, input)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, questionJSON(item))
}

func (h *QuestionBankHandler) DeleteQuestion(c *gin.Context) {
	questionID := c.Param("questionId")
	if !ValidUUID(questionID) {
		uuidError(c, "questionId")
		return
	}
	if err := h.questionbank.DeleteQuestion(c.Request.Context(), questionID); err != nil {
		writeMappedError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindQuestionInput(c *gin.Context) (questionbankuc.QuestionInput, bool) {
	var req struct {
		Type      string          `json:"type"`
		Body      string          `json:"body"`
		Options   json.RawMessage `json:"options"`
		AnswerKey json.RawMessage `json:"answerKey"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return questionbankuc.QuestionInput{}, false
	}
	var fields []response.FieldError
	if !validQuestionType(quizdomain.QuestionType(strings.TrimSpace(req.Type))) {
		fields = append(fields, response.FieldError{Field: "type", Message: "invalid question type"})
	}
	if strings.TrimSpace(req.Body) == "" {
		fields = append(fields, response.FieldError{Field: "body", Message: "required"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return questionbankuc.QuestionInput{}, false
	}
	return questionbankuc.QuestionInput{Type: quizdomain.QuestionType(strings.TrimSpace(req.Type)), Body: req.Body, Options: req.Options, AnswerKey: req.AnswerKey}, true
}

func validQuestionType(value quizdomain.QuestionType) bool {
	switch value {
	case quizdomain.QuestionTypeMultipleChoice, quizdomain.QuestionTypeMultipleSelect, quizdomain.QuestionTypeTrueFalse, quizdomain.QuestionTypeRange, quizdomain.QuestionTypeEssay:
		return true
	default:
		return false
	}
}

func questionBankJSON(item quizdomain.QuestionBankDetail) gin.H {
	return gin.H{"id": item.ID, "subjectId": item.SubjectID, "scope": item.Scope, "ownerId": item.OwnerID, "questionCount": item.QuestionCount}
}

func questionJSON(item quizdomain.Question) gin.H {
	return gin.H{"id": item.ID, "bankId": item.BankID, "type": item.Type, "body": item.Body, "options": item.Options, "answerKey": item.AnswerKey}
}
