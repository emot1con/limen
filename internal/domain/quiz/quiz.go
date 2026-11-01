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

type QuestionBankDetail struct {
	QuestionBank
	QuestionCount int64
}

type QuestionBankFilter struct {
	Scope     QuestionBankScope
	SubjectID string
	OwnerID   string
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

type QuizStatus string

const (
	QuizStatusDraft     QuizStatus = "draft"
	QuizStatusPublished QuizStatus = "published"
	QuizStatusActive    QuizStatus = "active"
	QuizStatusEnded     QuizStatus = "ended"
)

type QuizSummary struct {
	ID          string
	Title       string
	SubjectID   string
	Description *string
}

type QuizFilter struct {
	SubjectID string
	ClassID   string
	Status    QuizStatus
}

type QuizQuestionDetail struct {
	ID             int
	QuestionNumber int
	Type           QuestionType
	Question       string
	Options        []QuestionOption
}

type QuestionOption struct {
	ID    int
	Label string
}

type QuizDetail struct {
	ID        string
	Title     string
	StartTime *time.Time
	EndTime   *time.Time
	Questions []QuizQuestionDetail
}

type AttemptFilter struct {
	QuizID        string
	StudentUserID string
	Status        QuizAttemptStatus
}

type Attempt struct {
	AttemptID   string
	QuizID      string
	QuizTitle   string
	StartedAt   time.Time
	SubmittedAt *time.Time
	Status      QuizAttemptStatus
	Score       *float64
}

type AttemptAnswer struct {
	AnswerID   string
	QuestionID int
	UserAnswer json.RawMessage
	IsCorrect  *bool
}

type AttemptResult struct {
	Attempt
	EssayPending bool
	Answers      []AttemptAnswer
}

type QueryRepository interface {
	ListQuizzes(context.Context, QuizFilter, int, int) ([]QuizSummary, int64, error)
	GetQuizDetail(context.Context, string) (QuizDetail, error)
	ListAttempts(context.Context, AttemptFilter, int, int) ([]Attempt, int64, error)
}

type QuestionBankRepository interface {
	Create(context.Context, QuestionBank) (string, error)
	FindByID(context.Context, string) (QuestionBank, error)
	List(context.Context, QuestionBankFilter, int, int) ([]QuestionBankDetail, int64, error)
	Update(context.Context, QuestionBank) error
	Delete(context.Context, string) error
}

type QuestionRepository interface {
	Create(context.Context, Question) (string, error)
	FindByID(context.Context, string) (Question, error)
	List(context.Context, string, int, int) ([]Question, int64, error)
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
	FindByNumber(context.Context, string, int) (QuizQuestion, error)
}

type QuizAttemptRepository interface {
	Create(context.Context, QuizAttempt) (string, error)
	FindByID(context.Context, string) (QuizAttempt, error)
	FindActive(context.Context, string, string) (QuizAttempt, error)
	ListByQuizAndStudent(context.Context, string, string) ([]QuizAttempt, error)
	Update(context.Context, QuizAttempt) error
}

type QuizAnswerRepository interface {
	Upsert(context.Context, QuizAnswer) error
	FindByID(context.Context, string) (QuizAnswer, error)
	FindByAttemptAndQuestion(context.Context, string, string) (QuizAnswer, error)
	ListByAttemptID(context.Context, string) ([]QuizAnswer, error)
	Update(context.Context, QuizAnswer) error
}
