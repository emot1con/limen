package questionbank

import (
	"context"
	"encoding/json"
	"strings"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	quizdomain "limen/internal/domain/quiz"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	banks     quizdomain.QuestionBankRepository
	questions quizdomain.QuestionRepository
	subjects  academicdomain.SubjectRepository
	logger    *logrus.Logger
}

func NewUseCase(banks quizdomain.QuestionBankRepository, questions quizdomain.QuestionRepository, subjects academicdomain.SubjectRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{banks: banks, questions: questions, subjects: subjects, logger: logger}
}

type QuestionInput struct {
	Type      quizdomain.QuestionType
	Body      string
	Options   json.RawMessage
	AnswerKey json.RawMessage
}

func (uc *UseCase) CreateBank(ctx context.Context, subjectID, ownerID string, scope quizdomain.QuestionBankScope) (quizdomain.QuestionBankDetail, error) {
	if !validBankScope(scope) || subjectID == "" || ownerID == "" {
		return quizdomain.QuestionBankDetail{}, domain.ErrInvalid
	}
	if _, err := uc.subjects.FindByID(ctx, subjectID); err != nil {
		uc.logError(err, "question_bank_validate_subject", logrus.Fields{"subject_id": subjectID})
		return quizdomain.QuestionBankDetail{}, err
	}
	id, err := uc.banks.Create(ctx, quizdomain.QuestionBank{SubjectID: subjectID, OwnerID: ownerID, Scope: scope})
	if err != nil {
		uc.logError(err, "question_bank_create", nil)
		return quizdomain.QuestionBankDetail{}, err
	}
	bank, err := uc.banks.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "question_bank_get_created", logrus.Fields{"bank_id": id})
		return quizdomain.QuestionBankDetail{}, err
	}
	return quizdomain.QuestionBankDetail{QuestionBank: bank}, nil
}

func (uc *UseCase) ListBanks(ctx context.Context, filter quizdomain.QuestionBankFilter, page, pageSize int) ([]quizdomain.QuestionBankDetail, int64, error) {
	items, total, err := uc.banks.List(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "question_bank_list", nil)
	}
	return items, total, err
}

func (uc *UseCase) ListQuestions(ctx context.Context, bankID string, page, pageSize int) ([]quizdomain.Question, int64, error) {
	if _, err := uc.banks.FindByID(ctx, bankID); err != nil {
		uc.logError(err, "question_bank_get", logrus.Fields{"bank_id": bankID})
		return nil, 0, err
	}
	items, total, err := uc.questions.List(ctx, bankID, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "question_bank_list_questions", logrus.Fields{"bank_id": bankID})
	}
	return items, total, err
}

func (uc *UseCase) CreateQuestion(ctx context.Context, bankID string, input QuestionInput) (quizdomain.Question, error) {
	input.Body = strings.TrimSpace(input.Body)
	if !validQuestionType(input.Type) || input.Body == "" {
		return quizdomain.Question{}, domain.ErrInvalid
	}
	if _, err := uc.banks.FindByID(ctx, bankID); err != nil {
		uc.logError(err, "question_bank_get_for_question", logrus.Fields{"bank_id": bankID})
		return quizdomain.Question{}, err
	}
	id, err := uc.questions.Create(ctx, quizdomain.Question{BankID: bankID, Type: input.Type, Body: input.Body, Options: input.Options, AnswerKey: input.AnswerKey})
	if err != nil {
		uc.logError(err, "question_bank_create_question", logrus.Fields{"bank_id": bankID})
		return quizdomain.Question{}, err
	}
	item, err := uc.questions.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "question_bank_get_created_question", logrus.Fields{"question_id": id})
	}
	return item, err
}

func (uc *UseCase) UpdateQuestion(ctx context.Context, questionID string, input QuestionInput) (quizdomain.Question, error) {
	input.Body = strings.TrimSpace(input.Body)
	if !validQuestionType(input.Type) || input.Body == "" {
		return quizdomain.Question{}, domain.ErrInvalid
	}
	item, err := uc.questions.FindByID(ctx, questionID)
	if err != nil {
		uc.logError(err, "question_bank_get_question", logrus.Fields{"question_id": questionID})
		return quizdomain.Question{}, err
	}
	item.Type = input.Type
	item.Body = input.Body
	item.Options = input.Options
	item.AnswerKey = input.AnswerKey
	if err := uc.questions.Update(ctx, item); err != nil {
		uc.logError(err, "question_bank_update_question", logrus.Fields{"question_id": questionID})
		return quizdomain.Question{}, err
	}
	item, err = uc.questions.FindByID(ctx, questionID)
	if err != nil {
		uc.logError(err, "question_bank_get_updated_question", logrus.Fields{"question_id": questionID})
	}
	return item, err
}

func (uc *UseCase) DeleteQuestion(ctx context.Context, questionID string) error {
	if err := uc.questions.Delete(ctx, questionID); err != nil {
		if isForeignKeyViolation(err) {
			return domain.ErrConflict
		}
		uc.logError(err, "question_bank_delete_question", logrus.Fields{"question_id": questionID})
		return err
	}
	return nil
}

func validBankScope(scope quizdomain.QuestionBankScope) bool {
	return scope == quizdomain.QuestionBankScopePractice || scope == quizdomain.QuestionBankScopeExam
}

func validQuestionType(value quizdomain.QuestionType) bool {
	switch value {
	case quizdomain.QuestionTypeMultipleChoice, quizdomain.QuestionTypeMultipleSelect, quizdomain.QuestionTypeTrueFalse, quizdomain.QuestionTypeRange, quizdomain.QuestionTypeEssay:
		return true
	default:
		return false
	}
}

func isForeignKeyViolation(err error) bool {
	message := err.Error()
	return strings.Contains(message, "23503") || strings.Contains(message, "foreign key")
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists || err == domain.ErrConflict {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}
