package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	quizdomain "limen/internal/domain/quiz"
	quizuc "limen/internal/usecase/quiz"

	"github.com/gin-gonic/gin"
)

type QuizHandler struct{ quiz *quizuc.UseCase }

func NewQuizHandler(quiz *quizuc.UseCase) *QuizHandler {
	return &QuizHandler{quiz: quiz}
}

func (h *QuizHandler) CreateQuiz(c *gin.Context) {
	var req struct {
		SubjectID   string   `json:"subjectId"`
		Title       string   `json:"title"`
		Description *string  `json:"description"`
		PublishedAt *string  `json:"publishedAt"`
		StartTime   *string  `json:"startTime"`
		EndTime     *string  `json:"endTime"`
		QuestionIDs []string `json:"questionIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	var fields []response.FieldError
	if !ValidUUID(req.SubjectID) {
		fields = append(fields, response.FieldError{Field: "subjectId", Message: "must be a UUID"})
	}
	if strings.TrimSpace(req.Title) == "" {
		fields = append(fields, response.FieldError{Field: "title", Message: "required"})
	}
	publishedAt, ok := parseTimePtr(req.PublishedAt, "publishedAt", &fields)
	startTime, okStart := parseTimePtr(req.StartTime, "startTime", &fields)
	endTime, okEnd := parseTimePtr(req.EndTime, "endTime", &fields)
	if !ok || !okStart || !okEnd {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	if startTime != nil && endTime != nil && !startTime.Before(*endTime) {
		fields = append(fields, response.FieldError{Field: "endTime", Message: "must be after startTime"})
	}
	for i, questionID := range req.QuestionIDs {
		if !ValidUUID(questionID) {
			fields = append(fields, response.FieldError{Field: "questionIds", Message: "item " + strconv.Itoa(i) + " must be a UUID"})
		}
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
	detail, err := h.quiz.CreateQuiz(c.Request.Context(), quizuc.CreateQuizInput{
		SubjectID:   req.SubjectID,
		OwnerID:     ownerID,
		Title:       req.Title,
		Description: req.Description,
		PublishedAt: publishedAt,
		StartTime:   startTime,
		EndTime:     endTime,
		QuestionIDs: req.QuestionIDs,
	})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, quizDetailJSON(detail))
}

func parseTimePtr(raw *string, field string, fields *[]response.FieldError) (*time.Time, bool) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, true
	}
	value, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		*fields = append(*fields, response.FieldError{Field: field, Message: "must be RFC3339 date-time"})
		return nil, false
	}
	return &value, true
}

func (h *QuizHandler) ListQuizzes(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	var fields []response.FieldError
	for _, key := range []string{"subjectId", "classId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			fields = append(fields, response.FieldError{Field: key, Message: "must be a UUID"})
		}
	}
	filter := quizdomain.QuizFilter{SubjectID: c.Query("subjectId"), ClassID: c.Query("classId")}
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status := quizdomain.QuizStatus(raw)
		if !validQuizStatus(status) {
			fields = append(fields, response.FieldError{Field: "status", Message: "must be draft, published, active, or ended"})
		} else {
			filter.Status = status
		}
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	items, total, err := h.quiz.ListQuizzes(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = quizSummaryJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *QuizHandler) GetQuiz(c *gin.Context) {
	quizID := c.Param("quizId")
	if !ValidUUID(quizID) {
		uuidError(c, "quizId")
		return
	}
	attemptID := c.Query("attemptId")
	if attemptID != "" && !ValidUUID(attemptID) {
		uuidError(c, "attemptId")
		return
	}
	detail, err := h.quiz.GetQuizDetail(c.Request.Context(), quizID, attemptID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, quizDetailJSON(detail))
}

func (h *QuizHandler) StartAttempt(c *gin.Context) {
	quizID := c.Param("quizId")
	if !ValidUUID(quizID) {
		uuidError(c, "quizId")
		return
	}
	studentUserID, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	attempt, err := h.quiz.StartAttempt(c.Request.Context(), quizID, studentUserID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, attemptJSON(attempt))
}

func (h *QuizHandler) SaveAnswer(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if !ValidUUID(attemptID) {
		uuidError(c, "attemptId")
		return
	}
	questionNumber, err := strconv.Atoi(c.Param("questionId"))
	if err != nil || questionNumber < 1 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "questionId", Message: "must be a positive integer"})
		return
	}
	var body map[string]json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		validationBody(c)
		return
	}
	userAnswer, ok := body["userAnswer"]
	if !ok {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "userAnswer", Message: "required"})
		return
	}
	result, err := h.quiz.SaveAnswer(c.Request.Context(), attemptID, questionNumber, userAnswer)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"questionId": result.QuestionID, "userAnswer": rawJSON(result.UserAnswer), "isCompleted": result.IsCompleted, "updatedAt": result.UpdatedAt})
}

func (h *QuizHandler) SubmitAttempt(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if !ValidUUID(attemptID) {
		uuidError(c, "attemptId")
		return
	}
	result, err := h.quiz.SubmitAttempt(c.Request.Context(), attemptID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, attemptResultJSON(result))
}

func (h *QuizHandler) ListAttempts(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	var fields []response.FieldError
	for _, key := range []string{"quizId", "studentUserId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			fields = append(fields, response.FieldError{Field: key, Message: "must be a UUID"})
		}
	}
	filter := quizdomain.AttemptFilter{QuizID: c.Query("quizId"), StudentUserID: c.Query("studentUserId")}
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status := quizdomain.QuizAttemptStatus(raw)
		if !validAttemptStatus(status) {
			fields = append(fields, response.FieldError{Field: "status", Message: "must be in_progress, submitted, or graded"})
		} else {
			filter.Status = status
		}
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	items, total, err := h.quiz.ListAttempts(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = attemptJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *QuizHandler) GetAttempt(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if !ValidUUID(attemptID) {
		uuidError(c, "attemptId")
		return
	}
	result, err := h.quiz.GetAttempt(c.Request.Context(), attemptID)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, attemptResultJSON(result))
}

func (h *QuizHandler) GradeAnswer(c *gin.Context) {
	attemptID := c.Param("attemptId")
	if !ValidUUID(attemptID) {
		uuidError(c, "attemptId")
		return
	}
	answerID := c.Param("answerId")
	if !ValidUUID(answerID) {
		uuidError(c, "answerId")
		return
	}
	var req struct {
		IsCorrect *bool    `json:"isCorrect"`
		Score     *float64 `json:"score"`
		Feedback  *string  `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationBody(c)
		return
	}
	graderID, err := middleware.GetUserID(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	result, err := h.quiz.GradeAnswer(c.Request.Context(), attemptID, answerID, graderID, quizuc.GradeInput{IsCorrect: req.IsCorrect, Score: req.Score, Feedback: req.Feedback})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"answerId": result.AnswerID, "isCorrect": result.IsCorrect, "score": result.Score, "gradedBy": result.GradedBy, "gradedAt": result.GradedAt})
}

func validQuizStatus(status quizdomain.QuizStatus) bool {
	switch status {
	case quizdomain.QuizStatusDraft, quizdomain.QuizStatusPublished, quizdomain.QuizStatusActive, quizdomain.QuizStatusEnded:
		return true
	default:
		return false
	}
}

func validAttemptStatus(status quizdomain.QuizAttemptStatus) bool {
	switch status {
	case quizdomain.QuizAttemptStatusInProgress, quizdomain.QuizAttemptStatusSubmitted, quizdomain.QuizAttemptStatusGraded:
		return true
	default:
		return false
	}
}

func quizSummaryJSON(item quizdomain.QuizSummary) gin.H {
	return gin.H{"id": item.ID, "title": item.Title, "subjectId": item.SubjectID, "description": item.Description}
}

func quizDetailJSON(item quizdomain.QuizDetail) gin.H {
	questions := make([]gin.H, len(item.Questions))
	for i, question := range item.Questions {
		options := make([]gin.H, len(question.Options))
		for j, option := range question.Options {
			options[j] = gin.H{"id": option.ID, "label": option.Label}
		}
		questions[i] = gin.H{
			"id":             question.ID,
			"questionNumber": question.QuestionNumber,
			"type":           question.Type,
			"question":       question.Question,
			"options":        options,
		}
	}
	return gin.H{"id": item.ID, "title": item.Title, "startTime": item.StartTime, "endTime": item.EndTime, "questions": questions}
}

func attemptJSON(item quizdomain.Attempt) gin.H {
	return gin.H{
		"attemptId":   item.AttemptID,
		"quizId":      item.QuizID,
		"quizTitle":   item.QuizTitle,
		"startedAt":   item.StartedAt,
		"submittedAt": item.SubmittedAt,
		"status":      item.Status,
		"score":       item.Score,
	}
}

func attemptResultJSON(item quizdomain.AttemptResult) gin.H {
	answers := make([]gin.H, len(item.Answers))
	for i, answer := range item.Answers {
		answers[i] = gin.H{"answerId": answer.AnswerID, "questionId": answer.QuestionID, "userAnswer": rawJSON(answer.UserAnswer), "isCorrect": answer.IsCorrect}
	}
	out := attemptJSON(item.Attempt)
	out["essayPending"] = item.EssayPending
	out["answers"] = answers
	return out
}

func rawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
