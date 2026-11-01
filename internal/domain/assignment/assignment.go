package assignment

import (
	"context"
	"time"
)

type Assignment struct {
	ID            string
	ClassID       string
	SubjectID     string
	TeacherUserID string
	Title         string
	Description   *string
	DueAt         time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AssignmentSubmission struct {
	ID            string
	AssignmentID  string
	StudentUserID string
	SubmittedAt   *time.Time
	Score         *float64
	Note          *string
	GradedBy      *string
	GradedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SubmissionDetail struct {
	AssignmentSubmission
	StudentName string
}

type AssignmentSort string

const (
	AssignmentSortNewest  AssignmentSort = "newest"
	AssignmentSortOldest  AssignmentSort = "oldest"
	AssignmentSortDueAsc  AssignmentSort = "due_asc"
	AssignmentSortDueDesc AssignmentSort = "due_desc"
)

type AssignmentFilter struct {
	ClassID       string
	SubjectID     string
	TeacherUserID string
	Sort          AssignmentSort
}

type AssignmentRepository interface {
	Create(context.Context, Assignment) (string, error)
	FindByID(context.Context, string) (Assignment, error)
	Update(context.Context, Assignment) error
	List(context.Context, AssignmentFilter, int, int) ([]Assignment, int64, error)
	Delete(context.Context, string) error
}

type AssignmentSubmissionRepository interface {
	Create(context.Context, AssignmentSubmission) (string, error)
	FindByID(context.Context, string) (AssignmentSubmission, error)
	FindByAssignmentAndStudent(context.Context, string, string) (AssignmentSubmission, error)
	Upsert(context.Context, AssignmentSubmission) error
	Update(context.Context, AssignmentSubmission) error
	List(context.Context, string, int, int) ([]SubmissionDetail, int64, error)
	FindDetail(context.Context, string, string) (SubmissionDetail, error)
	ListByStudentUserID(context.Context, string) ([]AssignmentSubmission, error)
}
