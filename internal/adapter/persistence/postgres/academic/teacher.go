package academic

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
)

type teacherModel struct {
	UserID    string         `gorm:"column:user_id;primaryKey"`
	NIP       *string        `gorm:"column:nip"`
	CreatedAt string         `gorm:"column:created_at"`
	UpdatedAt string         `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (teacherModel) TableName() string {
	return "teachers"
}

type TeacherRepository struct {
	db *gorm.DB
}

func NewTeacherRepository(db *gorm.DB) *TeacherRepository {
	return &TeacherRepository{db: db}
}

func (r *TeacherRepository) Create(ctx context.Context, teacher academicdomain.Teacher) (string, error) {
	model := teacherModel{
		UserID: teacher.UserID,
		NIP:    teacher.NIP,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.UserID, nil
}

func (r *TeacherRepository) FindByUserID(ctx context.Context, userID string) (academicdomain.Teacher, error) {
	var model teacherModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Teacher{}, domain.ErrNotFound
		}
		return academicdomain.Teacher{}, err
	}
	return r.toDomain(model), nil
}

func (r *TeacherRepository) FindByNIP(ctx context.Context, nip string) (academicdomain.Teacher, error) {
	var model teacherModel
	err := r.db.WithContext(ctx).Where("nip = ?", nip).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Teacher{}, domain.ErrNotFound
		}
		return academicdomain.Teacher{}, err
	}
	return r.toDomain(model), nil
}

func (r *TeacherRepository) Update(ctx context.Context, teacher academicdomain.Teacher) error {
	result := r.db.WithContext(ctx).Model(&teacherModel{}).Where("user_id = ?", teacher.UserID).Updates(map[string]interface{}{
		"nip": teacher.NIP,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TeacherRepository) List(ctx context.Context) ([]academicdomain.Teacher, error) {
	var models []teacherModel
	err := r.db.WithContext(ctx).Order("user_id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	teachers := make([]academicdomain.Teacher, len(models))
	for i, m := range models {
		teachers[i] = r.toDomain(m)
	}
	return teachers, nil
}

func (r *TeacherRepository) toDomain(model teacherModel) academicdomain.Teacher {
	return academicdomain.Teacher{
		UserID: model.UserID,
		NIP:    model.NIP,
	}
}

var _ academicdomain.TeacherRepository = (*TeacherRepository)(nil)
