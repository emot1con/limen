package academic

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
)

type subjectModel struct {
	ID        string `gorm:"column:id;primaryKey"`
	Code      string `gorm:"column:code"`
	Name      string `gorm:"column:name"`
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}

func (subjectModel) TableName() string {
	return "subjects"
}

type SubjectRepository struct {
	db *gorm.DB
}

func NewSubjectRepository(db *gorm.DB) *SubjectRepository {
	return &SubjectRepository{db: db}
}

func (r *SubjectRepository) Create(ctx context.Context, subject academicdomain.Subject) error {
	model := subjectModel{
		ID:   subject.ID,
		Code: subject.Code,
		Name: subject.Name,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *SubjectRepository) FindByID(ctx context.Context, id string) (academicdomain.Subject, error) {
	var model subjectModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Subject{}, domain.ErrNotFound
		}
		return academicdomain.Subject{}, err
	}
	return r.toDomain(model), nil
}

func (r *SubjectRepository) FindByCode(ctx context.Context, code string) (academicdomain.Subject, error) {
	var model subjectModel
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Subject{}, domain.ErrNotFound
		}
		return academicdomain.Subject{}, err
	}
	return r.toDomain(model), nil
}

func (r *SubjectRepository) Update(ctx context.Context, subject academicdomain.Subject) error {
	result := r.db.WithContext(ctx).Model(&subjectModel{}).Where("id = ?", subject.ID).Updates(map[string]interface{}{
		"code": subject.Code,
		"name": subject.Name,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SubjectRepository) List(ctx context.Context) ([]academicdomain.Subject, error) {
	var models []subjectModel
	err := r.db.WithContext(ctx).Order("code").Find(&models).Error
	if err != nil {
		return nil, err
	}
	subjects := make([]academicdomain.Subject, len(models))
	for i, m := range models {
		subjects[i] = r.toDomain(m)
	}
	return subjects, nil
}

func (r *SubjectRepository) toDomain(model subjectModel) academicdomain.Subject {
	return academicdomain.Subject{
		ID:   model.ID,
		Code: model.Code,
		Name: model.Name,
	}
}

var _ academicdomain.SubjectRepository = (*SubjectRepository)(nil)
