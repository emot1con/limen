package quiz

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	quizdomain "limen/internal/domain/quiz"

	"gorm.io/gorm"
)

type quizQuestionModel struct {
	QuizID         string `gorm:"column:quiz_id;primaryKey"`
	QuestionID     string `gorm:"column:question_id;primaryKey"`
	QuestionNumber int    `gorm:"column:question_number"`
}

func (quizQuestionModel) TableName() string {
	return "quiz_questions"
}

type QuizQuestionRepository struct {
	db *gorm.DB
}

func NewQuizQuestionRepository(db *gorm.DB) *QuizQuestionRepository {
	return &QuizQuestionRepository{db: db}
}

func (r *QuizQuestionRepository) Add(ctx context.Context, qq quizdomain.QuizQuestion) error {
	model := quizQuestionModel{
		QuizID:         qq.QuizID,
		QuestionID:     qq.QuestionID,
		QuestionNumber: qq.QuestionNumber,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *QuizQuestionRepository) Remove(ctx context.Context, quizID, questionID string) error {
	result := r.db.WithContext(ctx).Delete(&quizQuestionModel{}, "quiz_id = ? AND question_id = ?", quizID, questionID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *QuizQuestionRepository) ListByQuizID(ctx context.Context, quizID string) ([]quizdomain.QuizQuestion, error) {
	var models []quizQuestionModel
	err := r.db.WithContext(ctx).Where("quiz_id = ?", quizID).Order("question_number").Find(&models).Error
	if err != nil {
		return nil, err
	}
	questions := make([]quizdomain.QuizQuestion, len(models))
	for i, model := range models {
		questions[i] = r.toDomain(model)
	}
	return questions, nil
}

func (r *QuizQuestionRepository) FindByNumber(ctx context.Context, quizID string, number int) (quizdomain.QuizQuestion, error) {
	var model quizQuestionModel
	err := r.db.WithContext(ctx).Where("quiz_id = ? AND question_number = ?", quizID, number).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return quizdomain.QuizQuestion{}, domain.ErrNotFound
		}
		return quizdomain.QuizQuestion{}, err
	}
	return r.toDomain(model), nil
}

func (r *QuizQuestionRepository) toDomain(model quizQuestionModel) quizdomain.QuizQuestion {
	return quizdomain.QuizQuestion{
		QuizID:         model.QuizID,
		QuestionID:     model.QuestionID,
		QuestionNumber: model.QuestionNumber,
	}
}

var _ quizdomain.QuizQuestionRepository = (*QuizQuestionRepository)(nil)
