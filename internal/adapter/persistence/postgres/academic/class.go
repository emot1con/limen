package academic

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type classModel struct {
	ID                    string         `gorm:"column:id;primaryKey"`
	Name                  string         `gorm:"column:name"`
	Level                 *string        `gorm:"column:level"`
	HomeroomTeacherUserID *string        `gorm:"column:homeroom_teacher_user_id"`
	CreatedAt             string         `gorm:"column:created_at"`
	UpdatedAt             string         `gorm:"column:updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (classModel) TableName() string {
	return "classes"
}

type ClassRepository struct {
	db *gorm.DB
}

func NewClassRepository(db *gorm.DB) *ClassRepository {
	return &ClassRepository{db: db}
}

func (r *ClassRepository) Create(ctx context.Context, class academicdomain.Class) (string, error) {
	model := classModel{
		Name:                  class.Name,
		Level:                 class.Level,
		HomeroomTeacherUserID: class.HomeroomTeacherUserID,
	}
	if err := r.db.WithContext(ctx).Omit("ID", "CreatedAt", "UpdatedAt").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *ClassRepository) FindByID(ctx context.Context, id string) (academicdomain.Class, error) {
	var model classModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Class{}, domain.ErrNotFound
		}
		return academicdomain.Class{}, err
	}
	return r.toDomain(model), nil
}

func (r *ClassRepository) Update(ctx context.Context, class academicdomain.Class) error {
	result := r.db.WithContext(ctx).Model(&classModel{}).Where("id = ?", class.ID).Updates(map[string]interface{}{
		"name":                     class.Name,
		"level":                    class.Level,
		"homeroom_teacher_user_id": class.HomeroomTeacherUserID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ClassRepository) List(ctx context.Context) ([]academicdomain.Class, error) {
	var models []classModel
	err := r.db.WithContext(ctx).Order("name").Find(&models).Error
	if err != nil {
		return nil, err
	}
	classes := make([]academicdomain.Class, len(models))
	for i, m := range models {
		classes[i] = r.toDomain(m)
	}
	return classes, nil
}

func (r *ClassRepository) toDomain(model classModel) academicdomain.Class {
	return academicdomain.Class{
		ID:                    model.ID,
		Name:                  model.Name,
		Level:                 model.Level,
		HomeroomTeacherUserID: model.HomeroomTeacherUserID,
	}
}

var _ academicdomain.ClassRepository = (*ClassRepository)(nil)
