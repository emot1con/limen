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

func (r *QuestionBankRepository) Create(ctx context.Context, bank quizdomain.QuestionBank) (string, error) {
	model := questionBankModel{
		ID:        bank.ID,
		SubjectID: bank.SubjectID,
		OwnerID:   bank.OwnerID,
		Scope:     string(bank.Scope),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
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

func (r *QuestionBankRepository) List(ctx context.Context, filter quizdomain.QuestionBankFilter, limit, offset int) ([]quizdomain.QuestionBankDetail, int64, error) {
	query := r.db.WithContext(ctx).Table("question_banks qb")
	if filter.Scope != "" {
		query = query.Where("qb.scope = ?", string(filter.Scope))
	}
	if filter.SubjectID != "" {
		query = query.Where("qb.subject_id = ?", filter.SubjectID)
	}
	if filter.OwnerID != "" {
		query = query.Where("qb.owner_id = ?", filter.OwnerID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID            string
		SubjectID     string
		OwnerID       string
		Scope         string
		CreatedAt     time.Time
		UpdatedAt     time.Time
		QuestionCount int64
	}
	err := query.
		Select("qb.id, qb.subject_id, qb.owner_id, qb.scope, qb.created_at, qb.updated_at, COUNT(q.id) AS question_count").
		Joins("LEFT JOIN questions q ON q.bank_id = qb.id").
		Group("qb.id, qb.subject_id, qb.owner_id, qb.scope, qb.created_at, qb.updated_at").
		Order("qb.created_at, qb.id").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	banks := make([]quizdomain.QuestionBankDetail, len(rows))
	for i, row := range rows {
		banks[i] = quizdomain.QuestionBankDetail{
			QuestionBank:  quizdomain.QuestionBank{ID: row.ID, SubjectID: row.SubjectID, OwnerID: row.OwnerID, Scope: quizdomain.QuestionBankScope(row.Scope), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt},
			QuestionCount: row.QuestionCount,
		}
	}
	return banks, total, nil
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
