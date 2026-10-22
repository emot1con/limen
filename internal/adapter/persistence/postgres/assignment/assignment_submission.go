package assignment

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	assignmentdomain "limen/internal/domain/assignment"

	"gorm.io/gorm"
)

type assignmentSubmissionModel struct {
	ID            string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	AssignmentID  string         `gorm:"column:assignment_id"`
	StudentUserID string         `gorm:"column:student_user_id"`
	SubmittedAt   *time.Time     `gorm:"column:submitted_at"`
	Score         *float64       `gorm:"column:score"`
	Note          *string        `gorm:"column:note"`
	GradedBy      *string        `gorm:"column:graded_by"`
	GradedAt      *time.Time     `gorm:"column:graded_at"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (assignmentSubmissionModel) TableName() string {
	return "assignment_submissions"
}

type AssignmentSubmissionRepository struct {
	db *gorm.DB
}

func NewAssignmentSubmissionRepository(db *gorm.DB) *AssignmentSubmissionRepository {
	return &AssignmentSubmissionRepository{db: db}
}

func (r *AssignmentSubmissionRepository) Create(ctx context.Context, submission assignmentdomain.AssignmentSubmission) error {
	model := assignmentSubmissionModel{
		ID:            submission.ID,
		AssignmentID:  submission.AssignmentID,
		StudentUserID: submission.StudentUserID,
		SubmittedAt:   submission.SubmittedAt,
		Score:         submission.Score,
		Note:          submission.Note,
		GradedBy:      submission.GradedBy,
		GradedAt:      submission.GradedAt,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *AssignmentSubmissionRepository) FindByID(ctx context.Context, id string) (assignmentdomain.AssignmentSubmission, error) {
	var model assignmentSubmissionModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return assignmentdomain.AssignmentSubmission{}, domain.ErrNotFound
		}
		return assignmentdomain.AssignmentSubmission{}, err
	}
	return r.toDomain(model), nil
}

func (r *AssignmentSubmissionRepository) FindByAssignmentAndStudent(ctx context.Context, assignmentID, studentUserID string) (assignmentdomain.AssignmentSubmission, error) {
	var model assignmentSubmissionModel
	err := r.db.WithContext(ctx).Where("assignment_id = ? AND student_user_id = ?", assignmentID, studentUserID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return assignmentdomain.AssignmentSubmission{}, domain.ErrNotFound
		}
		return assignmentdomain.AssignmentSubmission{}, err
	}
	return r.toDomain(model), nil
}

func (r *AssignmentSubmissionRepository) Update(ctx context.Context, submission assignmentdomain.AssignmentSubmission) error {
	result := r.db.WithContext(ctx).Model(&assignmentSubmissionModel{}).Where("id = ?", submission.ID).Updates(map[string]interface{}{
		"assignment_id":   submission.AssignmentID,
		"student_user_id": submission.StudentUserID,
		"submitted_at":    submission.SubmittedAt,
		"score":           submission.Score,
		"note":            submission.Note,
		"graded_by":       submission.GradedBy,
		"graded_at":       submission.GradedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AssignmentSubmissionRepository) ListByAssignmentID(ctx context.Context, assignmentID string) ([]assignmentdomain.AssignmentSubmission, error) {
	var models []assignmentSubmissionModel
	err := r.db.WithContext(ctx).Where("assignment_id = ?", assignmentID).Order("student_user_id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	submissions := make([]assignmentdomain.AssignmentSubmission, len(models))
	for i, m := range models {
		submissions[i] = r.toDomain(m)
	}
	return submissions, nil
}

func (r *AssignmentSubmissionRepository) ListByStudentUserID(ctx context.Context, studentUserID string) ([]assignmentdomain.AssignmentSubmission, error) {
	var models []assignmentSubmissionModel
	err := r.db.WithContext(ctx).Where("student_user_id = ?", studentUserID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	submissions := make([]assignmentdomain.AssignmentSubmission, len(models))
	for i, m := range models {
		submissions[i] = r.toDomain(m)
	}
	return submissions, nil
}

func (r *AssignmentSubmissionRepository) toDomain(model assignmentSubmissionModel) assignmentdomain.AssignmentSubmission {
	return assignmentdomain.AssignmentSubmission{
		ID:            model.ID,
		AssignmentID:  model.AssignmentID,
		StudentUserID: model.StudentUserID,
		SubmittedAt:   model.SubmittedAt,
		Score:         model.Score,
		Note:          model.Note,
		GradedBy:      model.GradedBy,
		GradedAt:      model.GradedAt,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ assignmentdomain.AssignmentSubmissionRepository = (*AssignmentSubmissionRepository)(nil)
