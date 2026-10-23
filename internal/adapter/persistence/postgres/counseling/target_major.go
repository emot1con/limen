package counseling

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	counselingdomain "limen/internal/domain/counseling"

	"gorm.io/gorm"
)

type targetMajorModel struct {
	ID            string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	StudentUserID string    `gorm:"column:student_user_id"`
	Major         string    `gorm:"column:major"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (targetMajorModel) TableName() string {
	return "target_majors"
}

type TargetMajorRepository struct {
	db *gorm.DB
}

func NewTargetMajorRepository(db *gorm.DB) *TargetMajorRepository {
	return &TargetMajorRepository{db: db}
}

func (r *TargetMajorRepository) Create(ctx context.Context, major counselingdomain.TargetMajor) (string, error) {
	model := targetMajorModel{
		ID:            major.ID,
		StudentUserID: major.StudentUserID,
		Major:         major.Major,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *TargetMajorRepository) FindByID(ctx context.Context, id string) (counselingdomain.TargetMajor, error) {
	var model targetMajorModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return counselingdomain.TargetMajor{}, domain.ErrNotFound
		}
		return counselingdomain.TargetMajor{}, err
	}
	return r.toDomain(model), nil
}

func (r *TargetMajorRepository) ListByStudentUserID(ctx context.Context, studentUserID string) ([]counselingdomain.TargetMajor, error) {
	var models []targetMajorModel
	err := r.db.WithContext(ctx).Where("student_user_id = ?", studentUserID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	majors := make([]counselingdomain.TargetMajor, len(models))
	for i, model := range models {
		majors[i] = r.toDomain(model)
	}
	return majors, nil
}

func (r *TargetMajorRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&targetMajorModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TargetMajorRepository) toDomain(model targetMajorModel) counselingdomain.TargetMajor {
	return counselingdomain.TargetMajor{
		ID:            model.ID,
		StudentUserID: model.StudentUserID,
		Major:         model.Major,
		CreatedAt:     model.CreatedAt,
	}
}

var _ counselingdomain.TargetMajorRepository = (*TargetMajorRepository)(nil)
