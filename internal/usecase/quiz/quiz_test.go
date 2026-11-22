package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	quizdomain "limen/internal/domain/quiz"
	"limen/internal/platform/cache"
)

type quizRepoMock struct {
	quizzes map[string]quizdomain.Quiz
	next    int
}

func (m *quizRepoMock) Create(_ context.Context, item quizdomain.Quiz) (string, error) {
	m.next++
	id := "quiz-" + string(rune('0'+m.next))
	item.ID = id
	m.quizzes[id] = item
	return id, nil
}
func (m *quizRepoMock) FindByID(_ context.Context, id string) (quizdomain.Quiz, error) {
	item, ok := m.quizzes[id]
	if !ok {
		return quizdomain.Quiz{}, domain.ErrNotFound
	}
	return item, nil
}
func (*quizRepoMock) ListByOwnerID(context.Context, string) ([]quizdomain.Quiz, error) {
	return nil, nil
}
func (*quizRepoMock) Update(context.Context, quizdomain.Quiz) error { return nil }
func (*quizRepoMock) Delete(context.Context, string) error          { return nil }

type questionRepoMock struct {
	questions map[string]quizdomain.Question
}

func (*questionRepoMock) Create(context.Context, quizdomain.Question) (string, error) { return "", nil }
func (m *questionRepoMock) FindByID(_ context.Context, id string) (quizdomain.Question, error) {
	item, ok := m.questions[id]
	if !ok {
		return quizdomain.Question{}, domain.ErrNotFound
	}
	return item, nil
}
func (m *questionRepoMock) ListByIDs(_ context.Context, ids []string) ([]quizdomain.Question, error) {
	out := make([]quizdomain.Question, 0, len(ids))
	for _, id := range ids {
		if item, ok := m.questions[id]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}
func (*questionRepoMock) List(context.Context, string, int, int) ([]quizdomain.Question, int64, error) {
	return nil, 0, nil
}
func (*questionRepoMock) Update(context.Context, quizdomain.Question) error { return nil }
func (*questionRepoMock) Delete(context.Context, string) error              { return nil }

type quizQuestionRepoMock struct{ items []quizdomain.QuizQuestion }

func (m *quizQuestionRepoMock) Add(_ context.Context, item quizdomain.QuizQuestion) error {
	m.items = append(m.items, item)
	return nil
}
func (*quizQuestionRepoMock) Remove(context.Context, string, string) error { return nil }
func (m *quizQuestionRepoMock) ListByQuizID(_ context.Context, quizID string) ([]quizdomain.QuizQuestion, error) {
	var out []quizdomain.QuizQuestion
	for _, item := range m.items {
		if item.QuizID == quizID {
			out = append(out, item)
		}
	}
	return out, nil
}
func (m *quizQuestionRepoMock) FindByNumber(ctx context.Context, quizID string, number int) (quizdomain.QuizQuestion, error) {
	items, _ := m.ListByQuizID(ctx, quizID)
	for _, item := range items {
		if item.QuestionNumber == number {
			return item, nil
		}
	}
	return quizdomain.QuizQuestion{}, domain.ErrNotFound
}
func (m *quizQuestionRepoMock) CountByQuizID(ctx context.Context, quizID string) (int64, error) {
	items, _ := m.ListByQuizID(ctx, quizID)
	return int64(len(items)), nil
}

type attemptRepoMock struct {
	items  map[string]quizdomain.QuizAttempt
	active bool
	next   int
}

func (m *attemptRepoMock) Create(_ context.Context, item quizdomain.QuizAttempt) (string, error) {
	m.next++
	id := "attempt-" + string(rune('0'+m.next))
	item.ID = id
	m.items[id] = item
	return id, nil
}
func (m *attemptRepoMock) FindByID(_ context.Context, id string) (quizdomain.QuizAttempt, error) {
	item, ok := m.items[id]
	if !ok {
		return quizdomain.QuizAttempt{}, domain.ErrNotFound
	}
	return item, nil
}
func (m *attemptRepoMock) FindActive(context.Context, string, string) (quizdomain.QuizAttempt, error) {
	if m.active {
		return quizdomain.QuizAttempt{}, nil
	}
	return quizdomain.QuizAttempt{}, domain.ErrNotFound
}
func (*attemptRepoMock) ListByQuizAndStudent(context.Context, string, string) ([]quizdomain.QuizAttempt, error) {
	return nil, nil
}
func (m *attemptRepoMock) Update(_ context.Context, item quizdomain.QuizAttempt) error {
	m.items[item.ID] = item
	return nil
}

type answerRepoMock struct {
	items map[string]quizdomain.QuizAnswer
	next  int
}

func (m *answerRepoMock) key(attemptID, questionID string) string {
	return attemptID + "/" + questionID
}
func (m *answerRepoMock) Upsert(_ context.Context, item quizdomain.QuizAnswer) error {
	if item.ID == "" {
		m.next++
		item.ID = "answer-" + string(rune('0'+m.next))
		item.UpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	m.items[m.key(item.AttemptID, item.QuestionID)] = item
	return nil
}
func (m *answerRepoMock) FindByID(_ context.Context, id string) (quizdomain.QuizAnswer, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return quizdomain.QuizAnswer{}, domain.ErrNotFound
}
func (m *answerRepoMock) FindByAttemptAndQuestion(_ context.Context, attemptID, questionID string) (quizdomain.QuizAnswer, error) {
	item, ok := m.items[m.key(attemptID, questionID)]
	if !ok {
		return quizdomain.QuizAnswer{}, domain.ErrNotFound
	}
	return item, nil
}
func (m *answerRepoMock) ListByAttemptID(_ context.Context, attemptID string) ([]quizdomain.QuizAnswer, error) {
	var out []quizdomain.QuizAnswer
	for _, item := range m.items {
		if item.AttemptID == attemptID {
			out = append(out, item)
		}
	}
	return out, nil
}
func (m *answerRepoMock) CountByAttemptID(ctx context.Context, attemptID string) (int64, error) {
	items, _ := m.ListByAttemptID(ctx, attemptID)
	return int64(len(items)), nil
}
func (m *answerRepoMock) Update(ctx context.Context, item quizdomain.QuizAnswer) error {
	return m.Upsert(ctx, item)
}

type quizQueryRepoMock struct {
	details map[string]quizdomain.QuizDetail
}

func (*quizQueryRepoMock) ListQuizzes(context.Context, quizdomain.QuizFilter, int, int) ([]quizdomain.QuizSummary, int64, error) {
	return nil, 0, nil
}
func (m *quizQueryRepoMock) GetQuizDetail(_ context.Context, id string) (quizdomain.QuizDetail, error) {
	detail, ok := m.details[id]
	if !ok {
		return quizdomain.QuizDetail{}, domain.ErrNotFound
	}
	return detail, nil
}
func (*quizQueryRepoMock) ListAttempts(context.Context, quizdomain.AttemptFilter, int, int) ([]quizdomain.Attempt, int64, error) {
	return nil, 0, nil
}

type subjectRepoMock struct{}

func (*subjectRepoMock) Create(context.Context, academicdomain.Subject) (string, error) {
	return "", nil
}
func (*subjectRepoMock) FindByID(context.Context, string) (academicdomain.Subject, error) {
	return academicdomain.Subject{}, nil
}
func (*subjectRepoMock) FindByCode(context.Context, string) (academicdomain.Subject, error) {
	return academicdomain.Subject{}, nil
}
func (*subjectRepoMock) Update(context.Context, academicdomain.Subject) error   { return nil }
func (*subjectRepoMock) List(context.Context) ([]academicdomain.Subject, error) { return nil, nil }

type quizCacheMock struct{ bumps int }

func (*quizCacheMock) Get(context.Context, string, any) (bool, error)        { return false, nil }
func (*quizCacheMock) Set(context.Context, string, any, time.Duration) error { return nil }
func (*quizCacheMock) Delete(context.Context, ...string) error               { return nil }
func (*quizCacheMock) GetVersion(context.Context, string) (int64, error)     { return 1, nil }
func (m *quizCacheMock) BumpVersion(context.Context, string) (int64, error)  { m.bumps++; return 2, nil }

func newQuizUseCase() (*UseCase, *quizRepoMock, *questionRepoMock, *quizQuestionRepoMock, *attemptRepoMock, *answerRepoMock, *quizCacheMock) {
	quizzes := &quizRepoMock{quizzes: map[string]quizdomain.Quiz{}}
	questions := &questionRepoMock{questions: map[string]quizdomain.Question{}}
	quizQuestions := &quizQuestionRepoMock{}
	attempts := &attemptRepoMock{items: map[string]quizdomain.QuizAttempt{}}
	answers := &answerRepoMock{items: map[string]quizdomain.QuizAnswer{}}
	cacheMock := &quizCacheMock{}
	query := &quizQueryRepoMock{details: map[string]quizdomain.QuizDetail{"quiz-1": {ID: "quiz-1", Title: "Quiz"}}}
	uc := NewUseCase(quizzes, questions, quizQuestions, attempts, answers, query, &subjectRepoMock{}, cacheMock, time.Hour, logrus.New())
	return uc, quizzes, questions, quizQuestions, attempts, answers, cacheMock
}

func TestQuizFlowCreateStartAnswerAndSubmit(t *testing.T) {
	uc, _, questions, _, attempts, _, cacheMock := newQuizUseCase()
	questions.questions["q-1"] = quizdomain.Question{ID: "q-1", Type: quizdomain.QuestionTypeMultipleChoice, AnswerKey: json.RawMessage(`{"correct":"A"}`)}
	questions.questions["q-2"] = quizdomain.Question{ID: "q-2", Type: quizdomain.QuestionTypeEssay}
	detail, err := uc.CreateQuiz(context.Background(), CreateQuizInput{SubjectID: "subject-1", OwnerID: "teacher-1", Title: "  Quiz  ", QuestionIDs: []string{"q-1", "q-2"}})
	if err != nil {
		t.Fatal(err)
	}
	if cacheMock.bumps != 1 || detail.ID == "" {
		t.Fatalf("unexpected create result: %#v bumps=%d", detail, cacheMock.bumps)
	}
	attempt, err := uc.StartAttempt(context.Background(), "quiz-1", "student-1")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := uc.SaveAnswer(context.Background(), attempt.AttemptID, 1, json.RawMessage(`"A"`))
	if err != nil {
		t.Fatal(err)
	}
	if saved.QuestionID != 1 || saved.IsCompleted {
		t.Fatalf("unexpected save result: %#v", saved)
	}
	result, err := uc.SubmitAttempt(context.Background(), attempt.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != quizdomain.QuizAttemptStatusSubmitted || result.Score == nil || *result.Score != 50 || !result.EssayPending || len(result.Answers) != 2 {
		t.Fatalf("unexpected submit result: %#v", result)
	}
	if attempts.items[attempt.AttemptID].Status != quizdomain.QuizAttemptStatusSubmitted {
		t.Fatal("expected persisted submitted status")
	}
}

func TestStartAttemptRejectsActiveAttempt(t *testing.T) {
	uc, quizzes, _, _, attempts, _, _ := newQuizUseCase()
	quizzes.quizzes["quiz-1"] = quizdomain.Quiz{ID: "quiz-1", Title: "Quiz"}
	attempts.active = true
	_, err := uc.StartAttempt(context.Background(), "quiz-1", "student-1")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestGradeAnswerRecomputesAttempt(t *testing.T) {
	uc, quizzes, questions, quizQuestions, attempts, answers, _ := newQuizUseCase()
	quizzes.quizzes["quiz-1"] = quizdomain.Quiz{ID: "quiz-1", Title: "Quiz"}
	questions.questions["q-essay"] = quizdomain.Question{ID: "q-essay", Type: quizdomain.QuestionTypeEssay}
	quizQuestions.items = []quizdomain.QuizQuestion{{QuizID: "quiz-1", QuestionID: "q-essay", QuestionNumber: 1}}
	now := time.Now()
	attempts.items["attempt-1"] = quizdomain.QuizAttempt{ID: "attempt-1", QuizID: "quiz-1", Status: quizdomain.QuizAttemptStatusSubmitted, SubmittedAt: &now}
	answers.items["attempt-1/q-essay"] = quizdomain.QuizAnswer{ID: "answer-1", AttemptID: "attempt-1", QuestionID: "q-essay", Answer: json.RawMessage(`"essay"`)}
	score := 75.0
	result, err := uc.GradeAnswer(context.Background(), "attempt-1", "answer-1", "teacher-1", GradeInput{Score: &score})
	if err != nil {
		t.Fatal(err)
	}
	if result.Score == nil || *result.Score != 75 || result.GradedBy != "teacher-1" {
		t.Fatalf("unexpected grade result: %#v", result)
	}
	if attempts.items["attempt-1"].Status != quizdomain.QuizAttemptStatusGraded || attempts.items["attempt-1"].Score == nil || *attempts.items["attempt-1"].Score != 75 {
		t.Fatalf("unexpected recomputed attempt: %#v", attempts.items["attempt-1"])
	}
}

var _ cache.Cache = (*quizCacheMock)(nil)
