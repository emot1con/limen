package quiz

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
)

type QueryRepository struct{ db *gorm.DB }

func NewQueryRepository(db *gorm.DB) *QueryRepository { return &QueryRepository{db: db} }

func (r *QueryRepository) ListQuizzes(ctx context.Context, filter quizdomain.QuizFilter, limit, offset int) ([]quizdomain.QuizSummary, int64, error) {
	query := r.db.WithContext(ctx).Table("quizzes q").Where("q.deleted_at IS NULL")
	if filter.SubjectID != "" {
		query = query.Where("q.subject_id = ?", filter.SubjectID)
	}
	if filter.ClassID != "" {
		query = query.Where("EXISTS (SELECT 1 FROM teaching_assignments ta WHERE ta.subject_id = q.subject_id AND ta.teacher_user_id = q.owner_id AND ta.class_id = ?)", filter.ClassID)
	}
	switch filter.Status {
	case quizdomain.QuizStatusDraft:
		query = query.Where("q.published_at IS NULL")
	case quizdomain.QuizStatusPublished:
		query = query.Where("q.published_at IS NOT NULL AND (q.start_time IS NULL OR now() < q.start_time)")
	case quizdomain.QuizStatusActive:
		query = query.Where("q.start_time IS NOT NULL AND q.start_time <= now() AND (q.end_time IS NULL OR now() <= q.end_time)")
	case quizdomain.QuizStatusEnded:
		query = query.Where("q.end_time IS NOT NULL AND now() > q.end_time")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID          string
		Title       string
		SubjectID   string
		Description *string
	}
	if err := query.Select("q.id, q.title, q.subject_id, q.description").Order("q.created_at DESC, q.id").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]quizdomain.QuizSummary, len(rows))
	for i, v := range rows {
		out[i] = quizdomain.QuizSummary{ID: v.ID, Title: v.Title, SubjectID: v.SubjectID, Description: v.Description}
	}
	return out, total, nil
}

func (r *QueryRepository) GetQuizDetail(ctx context.Context, quizID string) (quizdomain.QuizDetail, error) {
	var quiz struct {
		ID        string
		Title     string
		StartTime *time.Time
		EndTime   *time.Time
	}
	err := r.db.WithContext(ctx).Table("quizzes q").Select("q.id, q.title, q.start_time, q.end_time").Where("q.id = ? AND q.deleted_at IS NULL", quizID).Scan(&quiz).Error
	if err != nil {
		return quizdomain.QuizDetail{}, err
	}
	if quiz.ID == "" {
		return quizdomain.QuizDetail{}, domain.ErrNotFound
	}
	var rows []struct {
		QuestionNumber int
		Type           string
		Body           string
		Options        []byte
	}
	err = r.db.WithContext(ctx).Table("quiz_questions qq").
		Select("qq.question_number, qu.type, qu.body, qu.options").
		Joins("JOIN questions qu ON qu.id = qq.question_id").
		Where("qq.quiz_id = ?", quizID).
		Order("qq.question_number").
		Scan(&rows).Error
	if err != nil {
		return quizdomain.QuizDetail{}, err
	}
	questions := make([]quizdomain.QuizQuestionDetail, len(rows))
	for i, v := range rows {
		questions[i] = quizdomain.QuizQuestionDetail{
			ID:             v.QuestionNumber,
			QuestionNumber: v.QuestionNumber,
			Type:           quizdomain.QuestionType(v.Type),
			Question:       v.Body,
			Options:        decodeOptions(v.Options),
		}
	}
	return quizdomain.QuizDetail{ID: quiz.ID, Title: quiz.Title, StartTime: quiz.StartTime, EndTime: quiz.EndTime, Questions: questions}, nil
}

func (r *QueryRepository) ListAttempts(ctx context.Context, filter quizdomain.AttemptFilter, limit, offset int) ([]quizdomain.Attempt, int64, error) {
	query := r.db.WithContext(ctx).Table("quiz_attempts a").
		Joins("JOIN quizzes q ON q.id = a.quiz_id").
		Where("a.deleted_at IS NULL")
	if filter.QuizID != "" {
		query = query.Where("a.quiz_id = ?", filter.QuizID)
	}
	if filter.StudentUserID != "" {
		query = query.Where("a.student_user_id = ?", filter.StudentUserID)
	}
	if filter.Status != "" {
		query = query.Where("a.status = ?", string(filter.Status))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID          string
		QuizID      string
		QuizTitle   string
		StartedAt   time.Time
		SubmittedAt *time.Time
		Status      string
		Score       *float64
	}
	if err := query.Select("a.id, a.quiz_id, q.title AS quiz_title, a.started_at, a.submitted_at, a.status, a.score").
		Order("a.started_at DESC, a.id").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]quizdomain.Attempt, len(rows))
	for i, v := range rows {
		out[i] = quizdomain.Attempt{
			AttemptID:   v.ID,
			QuizID:      v.QuizID,
			QuizTitle:   v.QuizTitle,
			StartedAt:   v.StartedAt,
			SubmittedAt: v.SubmittedAt,
			Status:      quizdomain.QuizAttemptStatus(v.Status),
			Score:       v.Score,
		}
	}
	return out, total, nil
}

func decodeOptions(raw []byte) []quizdomain.QuestionOption {
	if len(raw) == 0 {
		return nil
	}
	var parsed struct {
		Choices []json.RawMessage `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return nil
	}
	out := make([]quizdomain.QuestionOption, 0, len(parsed.Choices))
	for i, choice := range parsed.Choices {
		var item struct {
			ID    json.RawMessage `json:"id"`
			Label *string         `json:"label"`
			Text  *string         `json:"text"`
			Name  *string         `json:"name"`
		}
		if err := json.Unmarshal(choice, &item); err != nil {
			continue
		}
		label := ""
		switch {
		case item.Label != nil:
			label = *item.Label
		case item.Text != nil:
			label = *item.Text
		case item.Name != nil:
			label = *item.Name
		}
		out = append(out, quizdomain.QuestionOption{ID: decodeOptionID(item.ID, i), Label: label})
	}
	return out
}

func decodeOptionID(raw json.RawMessage, index int) int {
	if len(raw) == 0 {
		return index + 1
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if parsed, err := strconv.Atoi(text); err == nil {
			return parsed
		}
	}
	return index + 1
}

var _ quizdomain.QueryRepository = (*QueryRepository)(nil)
