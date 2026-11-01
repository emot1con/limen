package reports

import (
	"context"
	"strings"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	reportsdomain "limen/internal/domain/reports"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	reports  reportsdomain.QueryRepository
	students academicdomain.StudentRepository
	logger   *logrus.Logger
}

func NewUseCase(reports reportsdomain.QueryRepository, students academicdomain.StudentRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{reports: reports, students: students, logger: logger}
}

func (uc *UseCase) Attendance(ctx context.Context, filter reportsdomain.AttendanceFilter) ([]reportsdomain.AttendanceSummary, error) {
	items, err := uc.reports.ListAttendanceSummaries(ctx, filter)
	if err != nil {
		uc.logError(err, "reports_attendance", nil)
	}
	return items, err
}

func (uc *UseCase) Grades(ctx context.Context, filter reportsdomain.GradeFilter, page, pageSize int) ([]reportsdomain.GradeSummary, int64, error) {
	filter.Period = strings.TrimSpace(filter.Period)
	items, total, err := uc.reports.ListGradeSummaries(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "reports_grades", nil)
	}
	return items, total, err
}

func (uc *UseCase) ReportCard(ctx context.Context, studentUserID, period string) ([]reportsdomain.ReportCardSubject, error) {
	if _, err := uc.students.FindByUserID(ctx, studentUserID); err != nil {
		uc.logError(err, "reports_validate_student", logrus.Fields{"student_id": studentUserID})
		return nil, err
	}
	items, err := uc.reports.ListReportCardSubjects(ctx, studentUserID, strings.TrimSpace(period))
	if err != nil {
		uc.logError(err, "reports_report_card", logrus.Fields{"student_id": studentUserID})
	}
	return items, err
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}
