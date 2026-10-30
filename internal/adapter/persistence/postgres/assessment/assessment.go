package assessment

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	assessmentdomain "limen/internal/domain/assessment"

	"gorm.io/gorm"
)

type assessmentModel struct {
	ID            string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	ClassID       string         `gorm:"column:class_id"`
	SubjectID     string         `gorm:"column:subject_id"`
	TeacherUserID string         `gorm:"column:teacher_user_id"`
	Kind          string         `gorm:"column:kind"`
	KKM           *float64       `gorm:"column:kkm"`
	IsRemedial    bool           `gorm:"column:is_remedial"`
	Period        string         `gorm:"column:period"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (assessmentModel) TableName() string {
	return "assessments"
}

type AssessmentRepository struct {
	db *gorm.DB
}

func NewAssessmentRepository(db *gorm.DB) *AssessmentRepository {
	return &AssessmentRepository{db: db}
}

func (r *AssessmentRepository) Create(ctx context.Context, assessment assessmentdomain.Assessment) (string, error) {
	model := assessmentModel{
		ID:            assessment.ID,
		ClassID:       assessment.ClassID,
		SubjectID:     assessment.SubjectID,
		TeacherUserID: assessment.TeacherUserID,
		Kind:          assessment.Kind,
		KKM:           assessment.KKM,
		IsRemedial:    assessment.IsRemedial,
		Period:        assessment.Period,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *AssessmentRepository) FindByID(ctx context.Context, id string) (assessmentdomain.Assessment, error) {
	var model assessmentModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return assessmentdomain.Assessment{}, domain.ErrNotFound
		}
		return assessmentdomain.Assessment{}, err
	}
	return r.toDomain(model), nil
}

func (r *AssessmentRepository) Update(ctx context.Context, assessment assessmentdomain.Assessment) error {
	result := r.db.WithContext(ctx).Model(&assessmentModel{}).Where("id = ?", assessment.ID).Updates(map[string]interface{}{
		"class_id":        assessment.ClassID,
		"subject_id":      assessment.SubjectID,
		"teacher_user_id": assessment.TeacherUserID,
		"kind":            assessment.Kind,
		"kkm":             assessment.KKM,
		"is_remedial":     assessment.IsRemedial,
		"period":          assessment.Period,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AssessmentRepository) List(ctx context.Context, filter assessmentdomain.AssessmentFilter, limit, offset int) ([]assessmentdomain.Assessment, int64, error) {
	query := r.db.WithContext(ctx).Model(&assessmentModel{})
	if filter.ClassID != "" {
		query = query.Where("class_id = ?", filter.ClassID)
	}
	if filter.SubjectID != "" {
		query = query.Where("subject_id = ?", filter.SubjectID)
	}
	if filter.Kind != "" {
		query = query.Where("kind = ?", filter.Kind)
	}
	if filter.Period != "" {
		query = query.Where("period = ?", filter.Period)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []assessmentModel
	if err := query.Order("created_at, id").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	assessments := make([]assessmentdomain.Assessment, len(models))
	for i, model := range models {
		assessments[i] = r.toDomain(model)
	}
	return assessments, total, nil
}

func (r *AssessmentRepository) toDomain(model assessmentModel) assessmentdomain.Assessment {
	return assessmentdomain.Assessment{
		ID:            model.ID,
		ClassID:       model.ClassID,
		SubjectID:     model.SubjectID,
		TeacherUserID: model.TeacherUserID,
		Kind:          model.Kind,
		KKM:           model.KKM,
		IsRemedial:    model.IsRemedial,
		Period:        model.Period,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ assessmentdomain.AssessmentRepository = (*AssessmentRepository)(nil)
