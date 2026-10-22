package counseling

import (
	"context"
	"time"

	counselingdomain "limen/internal/domain/counseling"

	"gorm.io/gorm"
)

type counselingFollowupModel struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	NoteID    string    `gorm:"column:note_id"`
	Note      string    `gorm:"column:note"`
	CreatedBy string    `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (counselingFollowupModel) TableName() string {
	return "counseling_followups"
}

type CounselingFollowupRepository struct {
	db *gorm.DB
}

func NewCounselingFollowupRepository(db *gorm.DB) *CounselingFollowupRepository {
	return &CounselingFollowupRepository{db: db}
}

func (r *CounselingFollowupRepository) Create(ctx context.Context, followup counselingdomain.CounselingFollowup) error {
	model := counselingFollowupModel{
		ID:        followup.ID,
		NoteID:    followup.NoteID,
		Note:      followup.Note,
		CreatedBy: followup.CreatedBy,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *CounselingFollowupRepository) ListByNoteID(ctx context.Context, noteID string) ([]counselingdomain.CounselingFollowup, error) {
	var models []counselingFollowupModel
	err := r.db.WithContext(ctx).Where("note_id = ?", noteID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	followups := make([]counselingdomain.CounselingFollowup, len(models))
	for i, model := range models {
		followups[i] = r.toDomain(model)
	}
	return followups, nil
}

func (r *CounselingFollowupRepository) toDomain(model counselingFollowupModel) counselingdomain.CounselingFollowup {
	return counselingdomain.CounselingFollowup{
		ID:        model.ID,
		NoteID:    model.NoteID,
		Note:      model.Note,
		CreatedBy: model.CreatedBy,
		CreatedAt: model.CreatedAt,
	}
}

var _ counselingdomain.CounselingFollowupRepository = (*CounselingFollowupRepository)(nil)
