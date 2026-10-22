package assessment

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	assessmentdomain "limen/internal/domain/assessment"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assessmentScoreModel struct {
	ID            string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	AssessmentID  string         `gorm:"column:assessment_id"`
	StudentUserID string         `gorm:"column:student_user_id"`
	Score         float64        `gorm:"column:score"`
	UpdatedBy     *string        `gorm:"column:updated_by"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (assessmentScoreModel) TableName() string {
	return "assessment_scores"
}

type AssessmentScoreRepository struct {
	db *gorm.DB
}

func NewAssessmentScoreRepository(db *gorm.DB) *AssessmentScoreRepository {
	return &AssessmentScoreRepository{db: db}
}

func (r *AssessmentScoreRepository) Create(ctx context.Context, score assessmentdomain.AssessmentScore) error {
	model := assessmentScoreModel{
		ID:            score.ID,
		AssessmentID:  score.AssessmentID,
		StudentUserID: score.StudentUserID,
		Score:         score.Score,
		UpdatedBy:     score.UpdatedBy,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *AssessmentScoreRepository) FindByID(ctx context.Context, id string) (assessmentdomain.AssessmentScore, error) {
	var model assessmentScoreModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return assessmentdomain.AssessmentScore{}, domain.ErrNotFound
		}
		return assessmentdomain.AssessmentScore{}, err
	}
	return r.toDomain(model), nil
}

func (r *AssessmentScoreRepository) FindByAssessmentAndStudent(ctx context.Context, assessmentID, studentUserID string) (assessmentdomain.AssessmentScore, error) {
	var model assessmentScoreModel
	err := r.db.WithContext(ctx).Where("assessment_id = ? AND student_user_id = ?", assessmentID, studentUserID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return assessmentdomain.AssessmentScore{}, domain.ErrNotFound
		}
		return assessmentdomain.AssessmentScore{}, err
	}
	return r.toDomain(model), nil
}

func (r *AssessmentScoreRepository) Upsert(ctx context.Context, score assessmentdomain.AssessmentScore) error {
	model := assessmentScoreModel{
		ID:            score.ID,
		AssessmentID:  score.AssessmentID,
		StudentUserID: score.StudentUserID,
		Score:         score.Score,
		UpdatedBy:     score.UpdatedBy,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assessment_id"}, {Name: "student_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score",
			"updated_by",
			"updated_at",
		}),
	}).Create(&model).Error
}

func (r *AssessmentScoreRepository) ListByAssessmentID(ctx context.Context, assessmentID string) ([]assessmentdomain.AssessmentScore, error) {
	var models []assessmentScoreModel
	err := r.db.WithContext(ctx).Where("assessment_id = ?", assessmentID).Order("student_user_id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	scores := make([]assessmentdomain.AssessmentScore, len(models))
	for i, model := range models {
		scores[i] = r.toDomain(model)
	}
	return scores, nil
}

func (r *AssessmentScoreRepository) ListByStudentUserID(ctx context.Context, studentUserID string) ([]assessmentdomain.AssessmentScore, error) {
	var models []assessmentScoreModel
	err := r.db.WithContext(ctx).Where("student_user_id = ?", studentUserID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	scores := make([]assessmentdomain.AssessmentScore, len(models))
	for i, model := range models {
		scores[i] = r.toDomain(model)
	}
	return scores, nil
}

func (r *AssessmentScoreRepository) toDomain(model assessmentScoreModel) assessmentdomain.AssessmentScore {
	return assessmentdomain.AssessmentScore{
		ID:            model.ID,
		AssessmentID:  model.AssessmentID,
		StudentUserID: model.StudentUserID,
		Score:         model.Score,
		UpdatedBy:     model.UpdatedBy,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ assessmentdomain.AssessmentScoreRepository = (*AssessmentScoreRepository)(nil)
