package quiz

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
)

type quizAttemptModel struct {
	ID            string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	QuizID        string         `gorm:"column:quiz_id"`
	StudentUserID string         `gorm:"column:student_user_id"`
	StartedAt     time.Time      `gorm:"column:started_at"`
	SubmittedAt   *time.Time     `gorm:"column:submitted_at"`
	Status        string         `gorm:"column:status"`
	Score         *float64       `gorm:"column:score"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (quizAttemptModel) TableName() string {
	return "quiz_attempts"
}

type QuizAttemptRepository struct {
	db *gorm.DB
}

func NewQuizAttemptRepository(db *gorm.DB) *QuizAttemptRepository {
	return &QuizAttemptRepository{db: db}
}

func (r *QuizAttemptRepository) Create(ctx context.Context, attempt quizdomain.QuizAttempt) (string, error) {
	model := quizAttemptModel{
		ID:            attempt.ID,
		QuizID:        attempt.QuizID,
		StudentUserID: attempt.StudentUserID,
		StartedAt:     attempt.StartedAt,
		SubmittedAt:   attempt.SubmittedAt,
		Status:        string(attempt.Status),
		Score:         attempt.Score,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *QuizAttemptRepository) FindByID(ctx context.Context, id string) (quizdomain.QuizAttempt, error) {
	var model quizAttemptModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quizdomain.QuizAttempt{}, domain.ErrNotFound
		}
		return quizdomain.QuizAttempt{}, err
	}
	return r.toDomain(model), nil
}

func (r *QuizAttemptRepository) ListByQuizAndStudent(ctx context.Context, quizID, studentUserID string) ([]quizdomain.QuizAttempt, error) {
	var models []quizAttemptModel
	err := r.db.WithContext(ctx).Where("quiz_id = ? AND student_user_id = ?", quizID, studentUserID).Order("started_at").Find(&models).Error
	if err != nil {
		return nil, err
	}
	attempts := make([]quizdomain.QuizAttempt, len(models))
	for i, model := range models {
		attempts[i] = r.toDomain(model)
	}
	return attempts, nil
}

func (r *QuizAttemptRepository) Update(ctx context.Context, attempt quizdomain.QuizAttempt) error {
	result := r.db.WithContext(ctx).Model(&quizAttemptModel{}).Where("id = ?", attempt.ID).Updates(map[string]interface{}{
		"quiz_id":         attempt.QuizID,
		"student_user_id": attempt.StudentUserID,
		"started_at":      attempt.StartedAt,
		"submitted_at":    attempt.SubmittedAt,
		"status":          string(attempt.Status),
		"score":           attempt.Score,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizAttemptRepository) toDomain(model quizAttemptModel) quizdomain.QuizAttempt {
	return quizdomain.QuizAttempt{
		ID:            model.ID,
		QuizID:        model.QuizID,
		StudentUserID: model.StudentUserID,
		StartedAt:     model.StartedAt,
		SubmittedAt:   model.SubmittedAt,
		Status:        quizdomain.QuizAttemptStatus(model.Status),
		Score:         model.Score,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ quizdomain.QuizAttemptRepository = (*QuizAttemptRepository)(nil)
