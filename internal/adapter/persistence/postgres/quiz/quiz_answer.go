package quiz

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type quizAnswerModel struct {
	ID         string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	AttemptID  string         `gorm:"column:attempt_id"`
	QuestionID string         `gorm:"column:question_id"`
	Answer     []byte         `gorm:"column:answer;type:jsonb"`
	IsCorrect  *bool          `gorm:"column:is_correct"`
	Score      *float64       `gorm:"column:score"`
	Feedback   *string        `gorm:"column:feedback"`
	GradedBy   *string        `gorm:"column:graded_by"`
	GradedAt   *time.Time     `gorm:"column:graded_at"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (quizAnswerModel) TableName() string {
	return "quiz_answers"
}

type QuizAnswerRepository struct {
	db *gorm.DB
}

func NewQuizAnswerRepository(db *gorm.DB) *QuizAnswerRepository {
	return &QuizAnswerRepository{db: db}
}

func (r *QuizAnswerRepository) Upsert(ctx context.Context, answer quizdomain.QuizAnswer) error {
	model := quizAnswerModel{
		ID:         answer.ID,
		AttemptID:  answer.AttemptID,
		QuestionID: answer.QuestionID,
		Answer:     answer.Answer,
		IsCorrect:  answer.IsCorrect,
		Score:      answer.Score,
		Feedback:   answer.Feedback,
		GradedBy:   answer.GradedBy,
		GradedAt:   answer.GradedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "attempt_id"}, {Name: "question_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"answer",
			"is_correct",
			"score",
			"feedback",
			"graded_by",
			"graded_at",
			"updated_at",
		}),
	}).Create(&model).Error
}

func (r *QuizAnswerRepository) FindByAttemptAndQuestion(ctx context.Context, attemptID, questionID string) (quizdomain.QuizAnswer, error) {
	var model quizAnswerModel
	err := r.db.WithContext(ctx).Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quizdomain.QuizAnswer{}, domain.ErrNotFound
		}
		return quizdomain.QuizAnswer{}, err
	}
	return r.toDomain(model), nil
}

func (r *QuizAnswerRepository) ListByAttemptID(ctx context.Context, attemptID string) ([]quizdomain.QuizAnswer, error) {
	var models []quizAnswerModel
	err := r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	answers := make([]quizdomain.QuizAnswer, len(models))
	for i, model := range models {
		answers[i] = r.toDomain(model)
	}
	return answers, nil
}

func (r *QuizAnswerRepository) toDomain(model quizAnswerModel) quizdomain.QuizAnswer {
	return quizdomain.QuizAnswer{
		ID:         model.ID,
		AttemptID:  model.AttemptID,
		QuestionID: model.QuestionID,
		Answer:     model.Answer,
		IsCorrect:  model.IsCorrect,
		Score:      model.Score,
		Feedback:   model.Feedback,
		GradedBy:   model.GradedBy,
		GradedAt:   model.GradedAt,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}

var _ quizdomain.QuizAnswerRepository = (*QuizAnswerRepository)(nil)
