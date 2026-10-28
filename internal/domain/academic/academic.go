package academic

import (
	"context"
	"time"

	userdomain "limen/internal/domain/user"
)

// ClassDetail is a read-model that enriches Class with aggregated data.
type ClassDetail struct {
	Class
	StudentCount int64
}

// StudentDetail is a read-model that joins student, user, and class data.
type StudentDetail struct {
	UserID    string
	FullName  string
	NIS       *string
	ClassID   *string
	ClassName *string
	Status    userdomain.Status
}

// QueryRepository exposes complex cross-table read queries.
// It lives in the domain so adapters only need to depend on domain, not usecase.
type QueryRepository interface {
	ListClasses(ctx context.Context, level, q string, limit, offset int) ([]ClassDetail, int64, error)
	ListSubjects(ctx context.Context, q string, limit, offset int) ([]Subject, int64, error)
	ListStudents(ctx context.Context, classID, q, sort string, limit, offset int) ([]StudentDetail, int64, error)
	FindStudent(ctx context.Context, id string) (StudentDetail, error)
	ListAssignments(ctx context.Context, teacherID, classID, subjectID string, limit, offset int) ([]TeachingAssignment, int64, error)
	AssignmentExists(ctx context.Context, teacherID, subjectID, classID string) (bool, error)
}

type ScheduleKind string

const (
	ScheduleKindLesson ScheduleKind = "pelajaran"
	ScheduleKindExam   ScheduleKind = "ujian"
	ScheduleKindTryout ScheduleKind = "tryout"
)

type Student struct {
	UserID    string
	NIS       *string
	ClassID   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Teacher struct {
	UserID    string
	NIP       *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Class struct {
	ID                    string
	Name                  string
	Level                 *string
	HomeroomTeacherUserID *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Subject struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TeachingAssignment struct {
	ID            string
	TeacherUserID string
	SubjectID     string
	ClassID       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Schedule struct {
	ID            string
	ClassID       string
	SubjectID     string
	TeacherUserID *string
	DayOfWeek     int16
	StartTime     time.Time
	EndTime       time.Time
	Kind          ScheduleKind
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StudentRepository interface {
	Create(context.Context, Student) (string, error)
	FindByUserID(context.Context, string) (Student, error)
	FindByNIS(context.Context, string) (Student, error)
	Update(context.Context, Student) error
	ListByClassID(context.Context, string) ([]Student, error)
}

type TeacherRepository interface {
	Create(context.Context, Teacher) (string, error)
	FindByUserID(context.Context, string) (Teacher, error)
	FindByNIP(context.Context, string) (Teacher, error)
	Update(context.Context, Teacher) error
	List(context.Context) ([]Teacher, error)
}

type ClassRepository interface {
	Create(context.Context, Class) (string, error)
	FindByID(context.Context, string) (Class, error)
	Update(context.Context, Class) error
	List(context.Context) ([]Class, error)
}

type SubjectRepository interface {
	Create(context.Context, Subject) (string, error)
	FindByID(context.Context, string) (Subject, error)
	FindByCode(context.Context, string) (Subject, error)
	Update(context.Context, Subject) error
	List(context.Context) ([]Subject, error)
}

type TeachingAssignmentRepository interface {
	Create(context.Context, TeachingAssignment) (string, error)
	FindByID(context.Context, string) (TeachingAssignment, error)
	ListByTeacherUserID(context.Context, string) ([]TeachingAssignment, error)
	ListByClassID(context.Context, string) ([]TeachingAssignment, error)
	Delete(context.Context, string) error
}

type ScheduleRepository interface {
	Create(context.Context, Schedule) (string, error)
	FindByID(context.Context, string) (Schedule, error)
	ListByClassID(context.Context, string) ([]Schedule, error)
	Update(context.Context, Schedule) error
	Delete(context.Context, string) error
}
