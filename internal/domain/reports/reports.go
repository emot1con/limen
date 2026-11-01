package reports

import (
	"context"
	"time"

	attendancedomain "limen/internal/domain/attendance"
)

type AttendanceGroupBy string

const (
	AttendanceGroupByStudent AttendanceGroupBy = "student"
	AttendanceGroupByClass   AttendanceGroupBy = "class"
	AttendanceGroupByDay     AttendanceGroupBy = "day"
)

type AttendanceFilter struct {
	ClassID       string
	StudentUserID string
	SubjectType   attendancedomain.SubjectType
	DateFrom      *time.Time
	DateTo        *time.Time
	GroupBy       AttendanceGroupBy
}

type AttendanceSummary struct {
	Key           string
	StudentUserID *string
	StudentName   *string
	ClassID       *string
	ClassName     *string
	Date          *time.Time
	Present       int64
	Late          int64
	Excused       int64
	Sick          int64
	Absent        int64
	Total         int64
}

type GradeFilter struct {
	ClassID   string
	SubjectID string
	Period    string
}

type GradeSummary struct {
	StudentUserID   string
	StudentName     string
	ClassID         string
	ClassName       string
	SubjectID       string
	SubjectName     string
	AverageScore    float64
	AssessmentCount int64
}

type ReportCardSubject struct {
	SubjectID   string
	SubjectName string
	Score       float64
	KKM         *float64
}

type QueryRepository interface {
	ListAttendanceSummaries(context.Context, AttendanceFilter) ([]AttendanceSummary, error)
	ListGradeSummaries(context.Context, GradeFilter, int, int) ([]GradeSummary, int64, error)
	ListReportCardSubjects(context.Context, string, string) ([]ReportCardSubject, error)
}
