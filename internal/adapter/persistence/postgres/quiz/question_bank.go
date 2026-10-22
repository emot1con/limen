package quiz

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
)

type questionBankModel struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	SubjectID string    `gorm:"column:subject_id"`
	OwnerID   string    `gorm:"column:owner_id"`
	Scope     string    `gorm:"column:scope"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (questionBankModel) TableName() string {
	return "question_banks"
}

type QuestionBankRepository struct {
	db *gorm.DB
}

func NewQuestionBankRepository(db *gorm.DB) *QuestionBankRepository {
	return &QuestionBankRepository{db: db}
}

func (r *QuestionBankRepository) Create(ctx context.Context, bank quizdomain.QuestionBank) error {
	model := questionBankModel{
		ID:        bank.ID,
		SubjectID: bank.SubjectID,
		OwnerID:   bank.OwnerID,
		Scope:     string(bank.Scope),
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *QuestionBankRepository) FindByID(ctx context.Context, id string) (quizdomain.QuestionBank, error) {
	var model questionBankModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quizdomain.QuestionBank{}, domain.ErrNotFound
		}
		return quizdomain.QuestionBank{}, err
	}
	return r.toDomain(model), nil
}

func (r *QuestionBankRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]quizdomain.QuestionBank, error) {
	var models []questionBankModel
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	banks := make([]quizdomain.QuestionBank, len(models))
	for i, model := range models {
		banks[i] = r.toDomain(model)
	}
	return banks, nil
}

func (r *QuestionBankRepository) Update(ctx context.Context, bank quizdomain.QuestionBank) error {
	result := r.db.WithContext(ctx).Model(&questionBankModel{}).Where("id = ?", bank.ID).Updates(map[string]interface{}{
		"subject_id": bank.SubjectID,
		"owner_id":   bank.OwnerID,
		"scope":      string(bank.Scope),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionBankRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&questionBankModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionBankRepository) toDomain(model questionBankModel) quizdomain.QuestionBank {
	return quizdomain.QuestionBank{
		ID:        model.ID,
		SubjectID: model.SubjectID,
		OwnerID:   model.OwnerID,
		Scope:     quizdomain.QuestionBankScope(model.Scope),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

var _ quizdomain.QuestionBankRepository = (*QuestionBankRepository)(nil)
