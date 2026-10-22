package academic

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
)

type teachingAssignmentModel struct {
	ID            string `gorm:"column:id;primaryKey"`
	TeacherUserID string `gorm:"column:teacher_user_id"`
	SubjectID     string `gorm:"column:subject_id"`
	ClassID       string `gorm:"column:class_id"`
	CreatedAt     string `gorm:"column:created_at"`
	UpdatedAt     string `gorm:"column:updated_at"`
}

func (teachingAssignmentModel) TableName() string {
	return "teaching_assignments"
}

type TeachingAssignmentRepository struct {
	db *gorm.DB
}

func NewTeachingAssignmentRepository(db *gorm.DB) *TeachingAssignmentRepository {
	return &TeachingAssignmentRepository{db: db}
}

func (r *TeachingAssignmentRepository) Create(ctx context.Context, assignment academicdomain.TeachingAssignment) error {
	model := teachingAssignmentModel{
		ID:            assignment.ID,
		TeacherUserID: assignment.TeacherUserID,
		SubjectID:     assignment.SubjectID,
		ClassID:       assignment.ClassID,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *TeachingAssignmentRepository) FindByID(ctx context.Context, id string) (academicdomain.TeachingAssignment, error) {
	var model teachingAssignmentModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.TeachingAssignment{}, domain.ErrNotFound
		}
		return academicdomain.TeachingAssignment{}, err
	}
	return r.toDomain(model), nil
}

func (r *TeachingAssignmentRepository) ListByTeacherUserID(ctx context.Context, teacherUserID string) ([]academicdomain.TeachingAssignment, error) {
	var models []teachingAssignmentModel
	err := r.db.WithContext(ctx).Where("teacher_user_id = ?", teacherUserID).Order("created_at").Find(&models).Error
	if err != nil {
		return nil, err
	}
	assignments := make([]academicdomain.TeachingAssignment, len(models))
	for i, m := range models {
		assignments[i] = r.toDomain(m)
	}
	return assignments, nil
}

func (r *TeachingAssignmentRepository) ListByClassID(ctx context.Context, classID string) ([]academicdomain.TeachingAssignment, error) {
	var models []teachingAssignmentModel
	err := r.db.WithContext(ctx).Where("class_id = ?", classID).Order("created_at").Find(&models).Error
	if err != nil {
		return nil, err
	}
	assignments := make([]academicdomain.TeachingAssignment, len(models))
	for i, m := range models {
		assignments[i] = r.toDomain(m)
	}
	return assignments, nil
}

func (r *TeachingAssignmentRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&teachingAssignmentModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TeachingAssignmentRepository) toDomain(model teachingAssignmentModel) academicdomain.TeachingAssignment {
	return academicdomain.TeachingAssignment{
		ID:            model.ID,
		TeacherUserID: model.TeacherUserID,
		SubjectID:     model.SubjectID,
		ClassID:       model.ClassID,
	}
}

var _ academicdomain.TeachingAssignmentRepository = (*TeachingAssignmentRepository)(nil)
