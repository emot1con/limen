package assignment

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	assignmentdomain "limen/internal/domain/assignment"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *AssignmentSubmissionRepository) Create(ctx context.Context, submission assignmentdomain.AssignmentSubmission) (string, error) {
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
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
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

func (r *AssignmentSubmissionRepository) Upsert(ctx context.Context, submission assignmentdomain.AssignmentSubmission) error {
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
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assignment_id"}, {Name: "student_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score",
			"note",
			"graded_by",
			"graded_at",
			"updated_at",
			"deleted_at",
		}),
	}).Create(&model).Error
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

func (r *AssignmentSubmissionRepository) List(ctx context.Context, assignmentID string, limit, offset int) ([]assignmentdomain.SubmissionDetail, int64, error) {
	query := r.db.WithContext(ctx).Table("assignment_submissions s").Joins("JOIN users u ON u.id = s.student_user_id").Where("s.assignment_id = ? AND s.deleted_at IS NULL", assignmentID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []submissionRow
	if err := query.Select(submissionSelect).Order("u.full_name, s.student_user_id").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toSubmissionDetails(rows), total, nil
}

func (r *AssignmentSubmissionRepository) FindDetail(ctx context.Context, assignmentID, studentUserID string) (assignmentdomain.SubmissionDetail, error) {
	var row submissionRow
	err := r.db.WithContext(ctx).Table("assignment_submissions s").Joins("JOIN users u ON u.id = s.student_user_id").Where("s.assignment_id = ? AND s.student_user_id = ? AND s.deleted_at IS NULL", assignmentID, studentUserID).Select(submissionSelect).Scan(&row).Error
	if err != nil {
		return assignmentdomain.SubmissionDetail{}, err
	}
	if row.ID == "" {
		return assignmentdomain.SubmissionDetail{}, domain.ErrNotFound
	}
	return toSubmissionDetail(row), nil
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

const submissionSelect = "s.id, s.assignment_id, s.student_user_id, s.submitted_at, s.score, s.note, s.graded_by, s.graded_at, s.created_at, s.updated_at, u.full_name AS student_name"

type submissionRow struct {
	ID            string
	AssignmentID  string
	StudentUserID string
	StudentName   string
	SubmittedAt   *time.Time
	Score         *float64
	Note          *string
	GradedBy      *string
	GradedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func toSubmissionDetail(row submissionRow) assignmentdomain.SubmissionDetail {
	return assignmentdomain.SubmissionDetail{
		AssignmentSubmission: assignmentdomain.AssignmentSubmission{
			ID:            row.ID,
			AssignmentID:  row.AssignmentID,
			StudentUserID: row.StudentUserID,
			SubmittedAt:   row.SubmittedAt,
			Score:         row.Score,
			Note:          row.Note,
			GradedBy:      row.GradedBy,
			GradedAt:      row.GradedAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		},
		StudentName: row.StudentName,
	}
}

func toSubmissionDetails(rows []submissionRow) []assignmentdomain.SubmissionDetail {
	out := make([]assignmentdomain.SubmissionDetail, len(rows))
	for i, row := range rows {
		out[i] = toSubmissionDetail(row)
	}
	return out
}

var _ assignmentdomain.AssignmentSubmissionRepository = (*AssignmentSubmissionRepository)(nil)
