package quiz

import (
	"context"
	"encoding/json"
	"time"
)

type QuestionBankScope string

type QuestionType string

type QuizAttemptStatus string

const (
	QuestionBankScopePractice QuestionBankScope = "latihan"
	QuestionBankScopeExam     QuestionBankScope = "ujian"

	QuestionTypeMultipleChoice QuestionType = "pilihan_ganda"
	QuestionTypeMultipleSelect QuestionType = "pilihan_ganda_kompleks"
	QuestionTypeTrueFalse      QuestionType = "benar_salah"
	QuestionTypeRange          QuestionType = "rentang_nilai"
	QuestionTypeEssay          QuestionType = "essay"

	QuizAttemptStatusInProgress QuizAttemptStatus = "in_progress"
	QuizAttemptStatusSubmitted  QuizAttemptStatus = "submitted"
	QuizAttemptStatusGraded     QuizAttemptStatus = "graded"
)

type QuestionBank struct {
	ID        string
	SubjectID string
	OwnerID   string
	Scope     QuestionBankScope
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Question struct {
	ID        string
	BankID    string
	Type      QuestionType
	Body      string
	Options   json.RawMessage
	AnswerKey json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Quiz struct {
	ID          string
	SubjectID   string
	OwnerID     string
	Title       string
	Description *string
	PublishedAt *time.Time
	StartTime   *time.Time
	EndTime     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type QuizQuestion struct {
	QuizID         string
	QuestionID     string
	QuestionNumber int
}

type QuizAttempt struct {
	ID            string
	QuizID        string
	StudentUserID string
	StartedAt     time.Time
	SubmittedAt   *time.Time
	Status        QuizAttemptStatus
	Score         *float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type QuizAnswer struct {
	ID         string
	AttemptID  string
	QuestionID string
	Answer     json.RawMessage
	IsCorrect  *bool
	Score      *float64
	Feedback   *string
	GradedBy   *string
	GradedAt   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type QuestionBankRepository interface {
	Create(context.Context, QuestionBank) (string, error)
	FindByID(context.Context, string) (QuestionBank, error)
	ListByOwnerID(context.Context, string) ([]QuestionBank, error)
	Update(context.Context, QuestionBank) error
	Delete(context.Context, string) error
}

type QuestionRepository interface {
	Create(context.Context, Question) (string, error)
	FindByID(context.Context, string) (Question, error)
	ListByBankID(context.Context, string) ([]Question, error)
	Update(context.Context, Question) error
	Delete(context.Context, string) error
}

type QuizRepository interface {
	Create(context.Context, Quiz) (string, error)
	FindByID(context.Context, string) (Quiz, error)
	ListByOwnerID(context.Context, string) ([]Quiz, error)
	Update(context.Context, Quiz) error
	Delete(context.Context, string) error
}

type QuizQuestionRepository interface {
	Add(context.Context, QuizQuestion) error
	Remove(context.Context, string, string) error
	ListByQuizID(context.Context, string) ([]QuizQuestion, error)
}

type QuizAttemptRepository interface {
	Create(context.Context, QuizAttempt) (string, error)
	FindByID(context.Context, string) (QuizAttempt, error)
	ListByQuizAndStudent(context.Context, string, string) ([]QuizAttempt, error)
	Update(context.Context, QuizAttempt) error
}

type QuizAnswerRepository interface {
	Upsert(context.Context, QuizAnswer) error
	FindByAttemptAndQuestion(context.Context, string, string) (QuizAnswer, error)
	ListByAttemptID(context.Context, string) ([]QuizAnswer, error)
}
