package counseling

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	counselingdomain "limen/internal/domain/counseling"

	"gorm.io/gorm"
)

type passingGradeModel struct {
	ID         string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	University string    `gorm:"column:university"`
	Major      string    `gorm:"column:major"`
	Score      float64   `gorm:"column:score"`
	Year       int       `gorm:"column:year"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (passingGradeModel) TableName() string {
	return "passing_grades"
}

type PassingGradeRepository struct {
	db *gorm.DB
}

func NewPassingGradeRepository(db *gorm.DB) *PassingGradeRepository {
	return &PassingGradeRepository{db: db}
}

func (r *PassingGradeRepository) Create(ctx context.Context, grade counselingdomain.PassingGrade) (string, error) {
	model := passingGradeModel{
		ID:         grade.ID,
		University: grade.University,
		Major:      grade.Major,
		Score:      grade.Score,
		Year:       grade.Year,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *PassingGradeRepository) FindByID(ctx context.Context, id string) (counselingdomain.PassingGrade, error) {
	var model passingGradeModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return counselingdomain.PassingGrade{}, domain.ErrNotFound
		}
		return counselingdomain.PassingGrade{}, err
	}
	return r.toDomain(model), nil
}

func (r *PassingGradeRepository) ListByUniversityMajorYear(ctx context.Context, university, major string, year int) ([]counselingdomain.PassingGrade, error) {
	var models []passingGradeModel
	err := r.db.WithContext(ctx).Where("university = ? AND major = ? AND year = ?", university, major, year).Order("score").Find(&models).Error
	if err != nil {
		return nil, err
	}
	grades := make([]counselingdomain.PassingGrade, len(models))
	for i, model := range models {
		grades[i] = r.toDomain(model)
	}
	return grades, nil
}

func (r *PassingGradeRepository) Update(ctx context.Context, grade counselingdomain.PassingGrade) error {
	result := r.db.WithContext(ctx).Model(&passingGradeModel{}).Where("id = ?", grade.ID).Updates(map[string]interface{}{
		"university": grade.University,
		"major":      grade.Major,
		"score":      grade.Score,
		"year":       grade.Year,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PassingGradeRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&passingGradeModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PassingGradeRepository) toDomain(model passingGradeModel) counselingdomain.PassingGrade {
	return counselingdomain.PassingGrade{
		ID:         model.ID,
		University: model.University,
		Major:      model.Major,
		Score:      model.Score,
		Year:       model.Year,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}

var _ counselingdomain.PassingGradeRepository = (*PassingGradeRepository)(nil)
