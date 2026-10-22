package counseling

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	counselingdomain "limen/internal/domain/counseling"

	"gorm.io/gorm"
)

type aptitudeResultModel struct {
	ID            string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	StudentUserID string    `gorm:"column:student_user_id"`
	Result        string    `gorm:"column:result"`
	AssessedAt    time.Time `gorm:"column:assessed_at"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (aptitudeResultModel) TableName() string {
	return "aptitude_results"
}

type AptitudeResultRepository struct {
	db *gorm.DB
}

func NewAptitudeResultRepository(db *gorm.DB) *AptitudeResultRepository {
	return &AptitudeResultRepository{db: db}
}

func (r *AptitudeResultRepository) Create(ctx context.Context, result counselingdomain.AptitudeResult) error {
	model := aptitudeResultModel{
		ID:            result.ID,
		StudentUserID: result.StudentUserID,
		Result:        result.Result,
		AssessedAt:    result.AssessedAt,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *AptitudeResultRepository) FindByID(ctx context.Context, id string) (counselingdomain.AptitudeResult, error) {
	var model aptitudeResultModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return counselingdomain.AptitudeResult{}, domain.ErrNotFound
		}
		return counselingdomain.AptitudeResult{}, err
	}
	return r.toDomain(model), nil
}

func (r *AptitudeResultRepository) ListByStudentUserID(ctx context.Context, studentUserID string) ([]counselingdomain.AptitudeResult, error) {
	var models []aptitudeResultModel
	err := r.db.WithContext(ctx).Where("student_user_id = ?", studentUserID).Order("assessed_at DESC").Find(&models).Error
	if err != nil {
		return nil, err
	}
	results := make([]counselingdomain.AptitudeResult, len(models))
	for i, model := range models {
		results[i] = r.toDomain(model)
	}
	return results, nil
}

func (r *AptitudeResultRepository) toDomain(model aptitudeResultModel) counselingdomain.AptitudeResult {
	return counselingdomain.AptitudeResult{
		ID:            model.ID,
		StudentUserID: model.StudentUserID,
		Result:        model.Result,
		AssessedAt:    model.AssessedAt,
		CreatedAt:     model.CreatedAt,
	}
}

var _ counselingdomain.AptitudeResultRepository = (*AptitudeResultRepository)(nil)
