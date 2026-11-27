package questionbank

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	quizdomain "limen/internal/domain/quiz"
)

type mockSubjectRepo struct {
	subjects map[string]academicdomain.Subject
}

func (m *mockSubjectRepo) Create(context.Context, academicdomain.Subject) (string, error) {
	return "", nil
}
func (m *mockSubjectRepo) FindByID(_ context.Context, id string) (academicdomain.Subject, error) {
	s, ok := m.subjects[id]
	if !ok {
		return academicdomain.Subject{}, domain.ErrNotFound
	}
	return s, nil
}
func (m *mockSubjectRepo) FindByCode(context.Context, string) (academicdomain.Subject, error) {
	return academicdomain.Subject{}, nil
}
func (m *mockSubjectRepo) List(context.Context) ([]academicdomain.Subject, error) {
	var list []academicdomain.Subject
	for _, s := range m.subjects {
		list = append(list, s)
	}
	return list, nil
}
func (m *mockSubjectRepo) Update(context.Context, academicdomain.Subject) error { return nil }

type mockBankRepo struct {
	banks map[string]quizdomain.QuestionBank
	next  int
}

func (m *mockBankRepo) Create(_ context.Context, b quizdomain.QuestionBank) (string, error) {
	m.next++
	id := "bank-1"
	b.ID = id
	m.banks[id] = b
	return id, nil
}
func (m *mockBankRepo) FindByID(_ context.Context, id string) (quizdomain.QuestionBank, error) {
	b, ok := m.banks[id]
	if !ok {
		return quizdomain.QuestionBank{}, domain.ErrNotFound
	}
	return b, nil
}
func (m *mockBankRepo) List(_ context.Context, _ quizdomain.QuestionBankFilter, _, _ int) ([]quizdomain.QuestionBankDetail, int64, error) {
	var list []quizdomain.QuestionBankDetail
	for _, b := range m.banks {
		list = append(list, quizdomain.QuestionBankDetail{QuestionBank: b})
	}
	return list, int64(len(list)), nil
}
func (m *mockBankRepo) Update(_ context.Context, b quizdomain.QuestionBank) error {
	m.banks[b.ID] = b
	return nil
}
func (m *mockBankRepo) Delete(context.Context, string) error { return nil }

type mockQuestionRepo struct {
	questions map[string]quizdomain.Question
	deleteErr error
	next      int
}

func (m *mockQuestionRepo) Create(_ context.Context, q quizdomain.Question) (string, error) {
	m.next++
	id := "q-1"
	q.ID = id
	m.questions[id] = q
	return id, nil
}
func (m *mockQuestionRepo) FindByID(_ context.Context, id string) (quizdomain.Question, error) {
	q, ok := m.questions[id]
	if !ok {
		return quizdomain.Question{}, domain.ErrNotFound
	}
	return q, nil
}
func (m *mockQuestionRepo) FindByIDIncludingDeleted(ctx context.Context, id string) (quizdomain.Question, error) {
	return m.FindByID(ctx, id)
}
func (m *mockQuestionRepo) ListByIDs(context.Context, []string) ([]quizdomain.Question, error) {
	return nil, nil
}
func (m *mockQuestionRepo) ListByIDsIncludingDeleted(ctx context.Context, ids []string) ([]quizdomain.Question, error) {
	return m.ListByIDs(ctx, ids)
}
func (m *mockQuestionRepo) List(_ context.Context, bankID string, _, _ int) ([]quizdomain.Question, int64, error) {
	var list []quizdomain.Question
	for _, q := range m.questions {
		if q.BankID == bankID {
			list = append(list, q)
		}
	}
	return list, int64(len(list)), nil
}
func (m *mockQuestionRepo) Update(_ context.Context, q quizdomain.Question) error {
	m.questions[q.ID] = q
	return nil
}
func (m *mockQuestionRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.questions, id)
	return nil
}

func setupQuestionBankUseCase() (*UseCase, *mockBankRepo, *mockQuestionRepo, *mockSubjectRepo) {
	bankRepo := &mockBankRepo{banks: make(map[string]quizdomain.QuestionBank)}
	qRepo := &mockQuestionRepo{questions: make(map[string]quizdomain.Question)}
	subRepo := &mockSubjectRepo{subjects: make(map[string]academicdomain.Subject)}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(bankRepo, qRepo, subRepo, logger)
	return uc, bankRepo, qRepo, subRepo
}

func TestQuestionBank_CreateBank(t *testing.T) {
	uc, _, _, subRepo := setupQuestionBankUseCase()

	// Validation error: invalid scope or empty fields
	_, err := uc.CreateBank(context.Background(), "Practice Bank", "", "owner-1", quizdomain.QuestionBankScopePractice)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	_, err = uc.CreateBank(context.Background(), "Practice Bank", "sub-1", "owner-1", "invalid-scope")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	// Subject not found
	_, err = uc.CreateBank(context.Background(), "Practice Bank", "sub-notfound", "owner-1", quizdomain.QuestionBankScopePractice)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Success
	subRepo.subjects["sub-1"] = academicdomain.Subject{ID: "sub-1", Name: "Math"}
	detail, err := uc.CreateBank(context.Background(), "Practice Bank", "sub-1", "owner-1", quizdomain.QuestionBankScopeExam)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.ID != "bank-1" || detail.Scope != quizdomain.QuestionBankScopeExam {
		t.Fatalf("unexpected bank detail: %+v", detail)
	}
}

func TestQuestionBank_ListBanks(t *testing.T) {
	uc, bankRepo, _, _ := setupQuestionBankUseCase()
	bankRepo.banks["bank-1"] = quizdomain.QuestionBank{ID: "bank-1"}

	list, total, err := uc.ListBanks(context.Background(), quizdomain.QuestionBankFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 bank, got %d, total %d", len(list), total)
	}
}

func TestQuestionBank_Questions(t *testing.T) {
	uc, bankRepo, qRepo, _ := setupQuestionBankUseCase()
	bankRepo.banks["bank-1"] = quizdomain.QuestionBank{ID: "bank-1"}

	// Create Question - invalid type or body
	_, err := uc.CreateQuestion(context.Background(), "bank-1", QuestionInput{
		Type: "invalid_type",
		Body: "What is 2+2?",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	// Create Question - bank not found
	_, err = uc.CreateQuestion(context.Background(), "bank-none", QuestionInput{
		Type: quizdomain.QuestionTypeMultipleChoice,
		Body: "What is 2+2?",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Create Question - success
	created, err := uc.CreateQuestion(context.Background(), "bank-1", QuestionInput{
		Type:      quizdomain.QuestionTypeMultipleChoice,
		Body:      "What is 2+2?",
		Options:   json.RawMessage(`["1","2","3","4"]`),
		AnswerKey: json.RawMessage(`"4"`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "q-1" || created.Body != "What is 2+2?" {
		t.Fatalf("unexpected question: %+v", created)
	}

	// List Questions
	qList, qTotal, err := uc.ListQuestions(context.Background(), "bank-1", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qTotal != 1 || len(qList) != 1 {
		t.Fatalf("expected 1 question, got total %d, list %d", qTotal, len(qList))
	}

	// Update Question - validation
	_, err = uc.UpdateQuestion(context.Background(), "q-1", QuestionInput{
		Type: quizdomain.QuestionTypeEssay,
		Body: "   ",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid on empty body, got %v", err)
	}

	// Update Question - success
	updated, err := uc.UpdateQuestion(context.Background(), "q-1", QuestionInput{
		Type: quizdomain.QuestionTypeEssay,
		Body: "Explain photosynthesis",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Body != "Explain photosynthesis" || updated.Type != quizdomain.QuestionTypeEssay {
		t.Fatalf("unexpected updated question: %+v", updated)
	}

	// Delete Question - FK violation error check
	qRepo.deleteErr = errors.New("pq: update or delete on table violates foreign key constraint (23503)")
	err = uc.DeleteQuestion(context.Background(), "q-1")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict on foreign key violation, got %v", err)
	}

	// Delete Question - success
	qRepo.deleteErr = nil
	err = uc.DeleteQuestion(context.Background(), "q-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := qRepo.questions["q-1"]; ok {
		t.Fatal("expected question to be deleted")
	}
}
