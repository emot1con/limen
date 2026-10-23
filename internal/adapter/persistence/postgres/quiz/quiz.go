package quiz

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
)

type quizModel struct {
	ID          string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	SubjectID   string         `gorm:"column:subject_id"`
	OwnerID     string         `gorm:"column:owner_id"`
	Title       string         `gorm:"column:title"`
	Description *string        `gorm:"column:description"`
	PublishedAt *time.Time     `gorm:"column:published_at"`
	StartTime   *time.Time     `gorm:"column:start_time"`
	EndTime     *time.Time     `gorm:"column:end_time"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (quizModel) TableName() string {
	return "quizzes"
}

type QuizRepository struct {
	db *gorm.DB
}

func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

func (r *QuizRepository) Create(ctx context.Context, quiz quizdomain.Quiz) (string, error) {
	model := quizModel{
		ID:          quiz.ID,
		SubjectID:   quiz.SubjectID,
		OwnerID:     quiz.OwnerID,
		Title:       quiz.Title,
		Description: quiz.Description,
		PublishedAt: quiz.PublishedAt,
		StartTime:   quiz.StartTime,
		EndTime:     quiz.EndTime,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *QuizRepository) FindByID(ctx context.Context, id string) (quizdomain.Quiz, error) {
	var model quizModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quizdomain.Quiz{}, domain.ErrNotFound
		}
		return quizdomain.Quiz{}, err
	}
	return r.toDomain(model), nil
}

func (r *QuizRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]quizdomain.Quiz, error) {
	var models []quizModel
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	quizzes := make([]quizdomain.Quiz, len(models))
	for i, model := range models {
		quizzes[i] = r.toDomain(model)
	}
	return quizzes, nil
}

func (r *QuizRepository) Update(ctx context.Context, quiz quizdomain.Quiz) error {
	result := r.db.WithContext(ctx).Model(&quizModel{}).Where("id = ?", quiz.ID).Updates(map[string]interface{}{
		"subject_id":   quiz.SubjectID,
		"owner_id":     quiz.OwnerID,
		"title":        quiz.Title,
		"description":  quiz.Description,
		"published_at": quiz.PublishedAt,
		"start_time":   quiz.StartTime,
		"end_time":     quiz.EndTime,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&quizModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizRepository) toDomain(model quizModel) quizdomain.Quiz {
	return quizdomain.Quiz{
		ID:          model.ID,
		SubjectID:   model.SubjectID,
		OwnerID:     model.OwnerID,
		Title:       model.Title,
		Description: model.Description,
		PublishedAt: model.PublishedAt,
		StartTime:   model.StartTime,
		EndTime:     model.EndTime,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

var _ quizdomain.QuizRepository = (*QuizRepository)(nil)
