package counseling

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	counselingdomain "limen/internal/domain/counseling"

	"gorm.io/gorm"
)

type counselingNoteModel struct {
	ID            string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	StudentUserID string    `gorm:"column:student_user_id"`
	AuthorID      string    `gorm:"column:author_id"`
	Note          string    `gorm:"column:note"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (counselingNoteModel) TableName() string {
	return "counseling_notes"
}

type CounselingNoteRepository struct {
	db *gorm.DB
}

func NewCounselingNoteRepository(db *gorm.DB) *CounselingNoteRepository {
	return &CounselingNoteRepository{db: db}
}

func (r *CounselingNoteRepository) Create(ctx context.Context, note counselingdomain.CounselingNote) error {
	model := counselingNoteModel{
		ID:            note.ID,
		StudentUserID: note.StudentUserID,
		AuthorID:      note.AuthorID,
		Note:          note.Note,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *CounselingNoteRepository) FindByID(ctx context.Context, id string) (counselingdomain.CounselingNote, error) {
	var model counselingNoteModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return counselingdomain.CounselingNote{}, domain.ErrNotFound
		}
		return counselingdomain.CounselingNote{}, err
	}
	return r.toDomain(model), nil
}

func (r *CounselingNoteRepository) Update(ctx context.Context, note counselingdomain.CounselingNote) error {
	result := r.db.WithContext(ctx).Model(&counselingNoteModel{}).Where("id = ?", note.ID).Updates(map[string]interface{}{
		"student_user_id": note.StudentUserID,
		"author_id":       note.AuthorID,
		"note":            note.Note,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CounselingNoteRepository) ListByStudentUserID(ctx context.Context, studentUserID string) ([]counselingdomain.CounselingNote, error) {
	var models []counselingNoteModel
	err := r.db.WithContext(ctx).Where("student_user_id = ?", studentUserID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	notes := make([]counselingdomain.CounselingNote, len(models))
	for i, model := range models {
		notes[i] = r.toDomain(model)
	}
	return notes, nil
}

func (r *CounselingNoteRepository) toDomain(model counselingNoteModel) counselingdomain.CounselingNote {
	return counselingdomain.CounselingNote{
		ID:            model.ID,
		StudentUserID: model.StudentUserID,
		AuthorID:      model.AuthorID,
		Note:          model.Note,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ counselingdomain.CounselingNoteRepository = (*CounselingNoteRepository)(nil)
