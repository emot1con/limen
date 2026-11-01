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
	ID            string         `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	StudentUserID string         `gorm:"column:student_user_id"`
	AuthorID      string         `gorm:"column:author_id"`
	Note          string         `gorm:"column:note"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
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

func (r *CounselingNoteRepository) Create(ctx context.Context, note counselingdomain.CounselingNote) (string, error) {
	model := counselingNoteModel{
		ID:            note.ID,
		StudentUserID: note.StudentUserID,
		AuthorID:      note.AuthorID,
		Note:          note.Note,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
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

func (r *CounselingNoteRepository) List(ctx context.Context, filter counselingdomain.CounselingNoteFilter, limit, offset int) ([]counselingdomain.CounselingNote, int64, error) {
	query := r.db.WithContext(ctx).Model(&counselingNoteModel{})
	if filter.StudentUserID != "" {
		query = query.Where("student_user_id = ?", filter.StudentUserID)
	}
	if filter.ClassID != "" {
		query = query.Where("student_user_id IN (SELECT user_id FROM students WHERE class_id = ? AND deleted_at IS NULL)", filter.ClassID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "created_at DESC, id"
	if filter.Sort == counselingdomain.CounselingNoteSortOldest {
		order = "created_at, id"
	}
	var models []counselingNoteModel
	err := query.Order(order).Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, 0, err
	}
	notes := make([]counselingdomain.CounselingNote, len(models))
	for i, model := range models {
		notes[i] = r.toDomain(model)
	}
	return notes, total, nil
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
