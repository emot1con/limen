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

type AssessmentFilter struct {
	ClassID, SubjectID, Kind, Period string
}

type ScoreDetail struct {
	AssessmentScore
	StudentName string
}

type AssessmentRepository interface {
	Create(context.Context, Assessment) (string, error)
	FindByID(context.Context, string) (Assessment, error)
	Update(context.Context, Assessment) error
	List(context.Context, AssessmentFilter, int, int) ([]Assessment, int64, error)
}

type AssessmentScoreRepository interface {
	Create(context.Context, AssessmentScore) (string, error)
	FindByID(context.Context, string) (AssessmentScore, error)
	FindByAssessmentAndStudent(context.Context, string, string) (AssessmentScore, error)
	Upsert(context.Context, AssessmentScore) error
	List(context.Context, string, int, int) ([]ScoreDetail, int64, error)
	FindDetail(context.Context, string, string) (ScoreDetail, error)
	ListByStudentUserID(context.Context, string) ([]AssessmentScore, error)
}
