package counseling

import (
	"context"
	"time"
)

type CounselingNote struct {
	ID            string
	StudentUserID string
	AuthorID      string
	Note          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CounselingFollowup struct {
	ID        string
	NoteID    string
	Note      string
	CreatedBy string
	CreatedAt time.Time
}

type AptitudeResult struct {
	ID            string
	StudentUserID string
	Result        string
	AssessedAt    time.Time
	CreatedAt     time.Time
}

type TargetMajor struct {
	ID            string
	StudentUserID string
	Major         string
	CreatedAt     time.Time
}

type PassingGrade struct {
	ID         string
	University string
	Major      string
	Score      float64
	Year       int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CounselingNoteRepository interface {
	Create(context.Context, CounselingNote) (string, error)
	FindByID(context.Context, string) (CounselingNote, error)
	Update(context.Context, CounselingNote) error
	ListByStudentUserID(context.Context, string) ([]CounselingNote, error)
}

type CounselingFollowupRepository interface {
	Create(context.Context, CounselingFollowup) (string, error)
	ListByNoteID(context.Context, string) ([]CounselingFollowup, error)
}

type AptitudeResultRepository interface {
	Create(context.Context, AptitudeResult) (string, error)
	FindByID(context.Context, string) (AptitudeResult, error)
	ListByStudentUserID(context.Context, string) ([]AptitudeResult, error)
}

type TargetMajorRepository interface {
	Create(context.Context, TargetMajor) (string, error)
	FindByID(context.Context, string) (TargetMajor, error)
	ListByStudentUserID(context.Context, string) ([]TargetMajor, error)
	Delete(context.Context, string) error
}

type PassingGradeFilter struct {
	University string
	Major      string
	Year       int
	YearSet    bool
}

type PassingGradeRepository interface {
	Create(context.Context, PassingGrade) (string, error)
	FindByID(context.Context, string) (PassingGrade, error)
	List(context.Context, PassingGradeFilter, int, int) ([]PassingGrade, int64, error)
	Update(context.Context, PassingGrade) error
	Delete(context.Context, string) error
}
