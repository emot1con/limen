package academic

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
)

type studentModel struct {
	UserID    string         `gorm:"column:user_id;primaryKey"`
	NIS       string         `gorm:"column:nis"`
	ClassID   *string        `gorm:"column:class_id"`
	CreatedAt string         `gorm:"column:created_at"`
	UpdatedAt string         `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (studentModel) TableName() string {
	return "students"
}

type StudentRepository struct {
	db *gorm.DB
}

func NewStudentRepository(db *gorm.DB) *StudentRepository {
	return &StudentRepository{db: db}
}

func (r *StudentRepository) Create(ctx context.Context, student academicdomain.Student) (string, error) {
	model := studentModel{
		UserID:  student.UserID,
		NIS:     student.NIS,
		ClassID: student.ClassID,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.UserID, nil
}

func (r *StudentRepository) FindByUserID(ctx context.Context, userID string) (academicdomain.Student, error) {
	var model studentModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Student{}, domain.ErrNotFound
		}
		return academicdomain.Student{}, err
	}
	return r.toDomain(model), nil
}

func (r *StudentRepository) FindByNIS(ctx context.Context, nis string) (academicdomain.Student, error) {
	var model studentModel
	err := r.db.WithContext(ctx).Where("nis = ?", nis).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Student{}, domain.ErrNotFound
		}
		return academicdomain.Student{}, err
	}
	return r.toDomain(model), nil
}

func (r *StudentRepository) Update(ctx context.Context, student academicdomain.Student) error {
	result := r.db.WithContext(ctx).Model(&studentModel{}).Where("user_id = ?", student.UserID).Updates(map[string]interface{}{
		"nis":      student.NIS,
		"class_id": student.ClassID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *StudentRepository) ListByClassID(ctx context.Context, classID string) ([]academicdomain.Student, error) {
	var models []studentModel
	err := r.db.WithContext(ctx).Where("class_id = ?", classID).Order("nis").Find(&models).Error
	if err != nil {
		return nil, err
	}
	students := make([]academicdomain.Student, len(models))
	for i, m := range models {
		students[i] = r.toDomain(m)
	}
	return students, nil
}

func (r *StudentRepository) toDomain(model studentModel) academicdomain.Student {
	return academicdomain.Student{
		UserID:  model.UserID,
		NIS:     model.NIS,
		ClassID: model.ClassID,
	}
}

var _ academicdomain.StudentRepository = (*StudentRepository)(nil)
