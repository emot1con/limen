package assignment

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	assignmentdomain "limen/internal/domain/assignment"

	"gorm.io/gorm"
)

type assignmentModel struct {
	ID            string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ClassID       string         `gorm:"column:class_id"`
	SubjectID     string         `gorm:"column:subject_id"`
	TeacherUserID string         `gorm:"column:teacher_user_id"`
	Title         string         `gorm:"column:title"`
	Description   *string        `gorm:"column:description"`
	DueAt         time.Time      `gorm:"column:due_at"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (assignmentModel) TableName() string {
	return "assignments"
}

type AssignmentRepository struct {
	db *gorm.DB
}

func NewAssignmentRepository(db *gorm.DB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

func (r *AssignmentRepository) Create(ctx context.Context, assignment assignmentdomain.Assignment) (string, error) {
	model := assignmentModel{
		ID:            assignment.ID,
		ClassID:       assignment.ClassID,
		SubjectID:     assignment.SubjectID,
		TeacherUserID: assignment.TeacherUserID,
		Title:         assignment.Title,
		Description:   assignment.Description,
		DueAt:         assignment.DueAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *AssignmentRepository) FindByID(ctx context.Context, id string) (assignmentdomain.Assignment, error) {
	var model assignmentModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return assignmentdomain.Assignment{}, domain.ErrNotFound
		}
		return assignmentdomain.Assignment{}, err
	}
	return r.toDomain(model), nil
}

func (r *AssignmentRepository) Update(ctx context.Context, assignment assignmentdomain.Assignment) error {
	result := r.db.WithContext(ctx).Model(&assignmentModel{}).Where("id = ?", assignment.ID).Updates(map[string]interface{}{
		"class_id":        assignment.ClassID,
		"subject_id":      assignment.SubjectID,
		"teacher_user_id": assignment.TeacherUserID,
		"title":           assignment.Title,
		"description":     assignment.Description,
		"due_at":          assignment.DueAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AssignmentRepository) List(ctx context.Context, filter assignmentdomain.AssignmentFilter, limit, offset int) ([]assignmentdomain.Assignment, int64, error) {
	query := r.db.WithContext(ctx).Model(&assignmentModel{})
	if filter.ClassID != "" {
		query = query.Where("class_id = ?", filter.ClassID)
	}
	if filter.SubjectID != "" {
		query = query.Where("subject_id = ?", filter.SubjectID)
	}
	if filter.TeacherUserID != "" {
		query = query.Where("teacher_user_id = ?", filter.TeacherUserID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "created_at DESC, id"
	switch filter.Sort {
	case assignmentdomain.AssignmentSortOldest:
		order = "created_at, id"
	case assignmentdomain.AssignmentSortDueAsc:
		order = "due_at, id"
	case assignmentdomain.AssignmentSortDueDesc:
		order = "due_at DESC, id"
	}
	var models []assignmentModel
	err := query.Order(order).Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, 0, err
	}
	assignments := make([]assignmentdomain.Assignment, len(models))
	for i, m := range models {
		assignments[i] = r.toDomain(m)
	}
	return assignments, total, nil
}

func (r *AssignmentRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&assignmentModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AssignmentRepository) toDomain(model assignmentModel) assignmentdomain.Assignment {
	return assignmentdomain.Assignment{
		ID:            model.ID,
		ClassID:       model.ClassID,
		SubjectID:     model.SubjectID,
		TeacherUserID: model.TeacherUserID,
		Title:         model.Title,
		Description:   model.Description,
		DueAt:         model.DueAt,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ assignmentdomain.AssignmentRepository = (*AssignmentRepository)(nil)
