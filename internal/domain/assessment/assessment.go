package assessment

import (
	"context"
	"time"
)

type Assessment struct {
	ID            string
	ClassID       string
	SubjectID     string
	TeacherUserID string
	Kind          string
	KKM           *float64
	IsRemedial    bool
	Period        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AssessmentScore struct {
	ID            string
	AssessmentID  string
	StudentUserID string
	Score         float64
	UpdatedBy     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AssessmentRepository interface {
	Create(context.Context, Assessment) (string, error)
	FindByID(context.Context, string) (Assessment, error)
	Update(context.Context, Assessment) error
	ListByClassSubjectPeriod(context.Context, string, string, string) ([]Assessment, error)
}

type AssessmentScoreRepository interface {
	Create(context.Context, AssessmentScore) (string, error)
	FindByID(context.Context, string) (AssessmentScore, error)
	FindByAssessmentAndStudent(context.Context, string, string) (AssessmentScore, error)
	Upsert(context.Context, AssessmentScore) error
	ListByAssessmentID(context.Context, string) ([]AssessmentScore, error)
	ListByStudentUserID(context.Context, string) ([]AssessmentScore, error)
}
