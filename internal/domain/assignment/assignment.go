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

type AssignmentRepository interface {
	Create(context.Context, Assignment) (string, error)
	FindByID(context.Context, string) (Assignment, error)
	Update(context.Context, Assignment) error
	ListByClassSubject(context.Context, string, string) ([]Assignment, error)
	Delete(context.Context, string) error
}

type AssignmentSubmissionRepository interface {
	Create(context.Context, AssignmentSubmission) (string, error)
	FindByID(context.Context, string) (AssignmentSubmission, error)
	FindByAssignmentAndStudent(context.Context, string, string) (AssignmentSubmission, error)
	Update(context.Context, AssignmentSubmission) error
	ListByAssignmentID(context.Context, string) ([]AssignmentSubmission, error)
	ListByStudentUserID(context.Context, string) ([]AssignmentSubmission, error)
}
