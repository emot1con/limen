package quiz

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
)

type questionModel struct {
	ID        string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	BankID    string    `gorm:"column:bank_id"`
	Type      string    `gorm:"column:type"`
	Body      string    `gorm:"column:body"`
	Options   []byte    `gorm:"column:options;type:jsonb"`
	AnswerKey []byte    `gorm:"column:answer_key;type:jsonb"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (questionModel) TableName() string {
	return "questions"
}

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

func (r *QuestionRepository) Create(ctx context.Context, question quizdomain.Question) error {
	model := questionModel{
		ID:        question.ID,
		BankID:    question.BankID,
		Type:      string(question.Type),
		Body:      question.Body,
		Options:   question.Options,
		AnswerKey: question.AnswerKey,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *QuestionRepository) FindByID(ctx context.Context, id string) (quizdomain.Question, error) {
	var model questionModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quizdomain.Question{}, domain.ErrNotFound
		}
		return quizdomain.Question{}, err
	}
	return r.toDomain(model), nil
}

func (r *QuestionRepository) ListByBankID(ctx context.Context, bankID string) ([]quizdomain.Question, error) {
	var models []questionModel
	err := r.db.WithContext(ctx).Where("bank_id = ?", bankID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	questions := make([]quizdomain.Question, len(models))
	for i, model := range models {
		questions[i] = r.toDomain(model)
	}
	return questions, nil
}

func (r *QuestionRepository) Update(ctx context.Context, question quizdomain.Question) error {
	result := r.db.WithContext(ctx).Model(&questionModel{}).Where("id = ?", question.ID).Updates(map[string]interface{}{
		"bank_id":    question.BankID,
		"type":       string(question.Type),
		"body":       question.Body,
		"options":    nullBytes(question.Options),
		"answer_key": nullBytes(question.AnswerKey),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&questionModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuestionRepository) toDomain(model questionModel) quizdomain.Question {
	return quizdomain.Question{
		ID:        model.ID,
		BankID:    model.BankID,
		Type:      quizdomain.QuestionType(model.Type),
		Body:      model.Body,
		Options:   model.Options,
		AnswerKey: model.AnswerKey,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func nullBytes(value []byte) interface{} {
	if len(value) == 0 {
		return nil
	}
	return value
}

var _ quizdomain.QuestionRepository = (*QuestionRepository)(nil)
