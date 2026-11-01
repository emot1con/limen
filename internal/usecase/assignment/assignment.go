package assignment

import (
	"context"
	"math"
	"strings"
	"time"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	assignmentdomain "limen/internal/domain/assignment"
	userdomain "limen/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	assignments assignmentdomain.AssignmentRepository
	submissions assignmentdomain.AssignmentSubmissionRepository
	classes     academicdomain.ClassRepository
	subjects    academicdomain.SubjectRepository
	students    academicdomain.StudentRepository
	users       userdomain.UserRepository
	logger      *logrus.Logger
}

func NewUseCase(assignments assignmentdomain.AssignmentRepository, submissions assignmentdomain.AssignmentSubmissionRepository, classes academicdomain.ClassRepository, subjects academicdomain.SubjectRepository, students academicdomain.StudentRepository, users userdomain.UserRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{assignments: assignments, submissions: submissions, classes: classes, subjects: subjects, students: students, users: users, logger: logger}
}

type CreateInput struct {
	ClassID, SubjectID, TeacherUserID, Title string
	Description                              *string
	DueAt                                    time.Time
}

func (uc *UseCase) List(ctx context.Context, filter assignmentdomain.AssignmentFilter, page, pageSize int) ([]assignmentdomain.Assignment, int64, error) {
	items, total, err := uc.assignments.List(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "assignment_list", nil)
	}
	return items, total, err
}

func (uc *UseCase) Create(ctx context.Context, input CreateInput) (assignmentdomain.Assignment, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.ClassID == "" || input.SubjectID == "" || input.TeacherUserID == "" || input.Title == "" || input.DueAt.IsZero() {
		return assignmentdomain.Assignment{}, domain.ErrInvalid
	}
	input.Description = trim(input.Description)
	if err := uc.validateReferences(ctx, input.ClassID, input.SubjectID, input.TeacherUserID); err != nil {
		return assignmentdomain.Assignment{}, err
	}
	id, err := uc.assignments.Create(ctx, assignmentdomain.Assignment{ClassID: input.ClassID, SubjectID: input.SubjectID, TeacherUserID: input.TeacherUserID, Title: input.Title, Description: input.Description, DueAt: input.DueAt})
	if err != nil {
		uc.logError(err, "assignment_create", nil)
		return assignmentdomain.Assignment{}, err
	}
	item, err := uc.assignments.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "assignment_get_created", logrus.Fields{"assignment_id": id})
	}
	return item, err
}

func (uc *UseCase) ListSubmissions(ctx context.Context, assignmentID string, page, pageSize int) ([]assignmentdomain.SubmissionDetail, int64, error) {
	if _, err := uc.assignments.FindByID(ctx, assignmentID); err != nil {
		uc.logError(err, "assignment_get_for_submissions", logrus.Fields{"assignment_id": assignmentID})
		return nil, 0, err
	}
	items, total, err := uc.submissions.List(ctx, assignmentID, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "assignment_list_submissions", logrus.Fields{"assignment_id": assignmentID})
	}
	return items, total, err
}

func (uc *UseCase) GradeSubmission(ctx context.Context, assignmentID, studentUserID, gradedBy string, score float64, note *string) (assignmentdomain.SubmissionDetail, error) {
	if !isFinite(score) {
		return assignmentdomain.SubmissionDetail{}, domain.ErrInvalid
	}
	assignment, err := uc.assignments.FindByID(ctx, assignmentID)
	if err != nil {
		uc.logError(err, "assignment_get_for_grade", logrus.Fields{"assignment_id": assignmentID})
		return assignmentdomain.SubmissionDetail{}, err
	}
	student, err := uc.students.FindByUserID(ctx, studentUserID)
	if err != nil {
		uc.logError(err, "assignment_get_student", logrus.Fields{"student_id": studentUserID})
		return assignmentdomain.SubmissionDetail{}, err
	}
	if student.ClassID == nil || *student.ClassID != assignment.ClassID {
		return assignmentdomain.SubmissionDetail{}, domain.ErrInvalid
	}
	now := time.Now()
	submission := assignmentdomain.AssignmentSubmission{AssignmentID: assignmentID, StudentUserID: studentUserID, Score: &score, Note: trim(note), GradedBy: optional(gradedBy), GradedAt: &now}
	if err := uc.submissions.Upsert(ctx, submission); err != nil {
		uc.logError(err, "assignment_grade_submission", logrus.Fields{"assignment_id": assignmentID, "student_id": studentUserID})
		return assignmentdomain.SubmissionDetail{}, err
	}
	item, err := uc.submissions.FindDetail(ctx, assignmentID, studentUserID)
	if err != nil {
		uc.logError(err, "assignment_get_graded_submission", logrus.Fields{"assignment_id": assignmentID, "student_id": studentUserID})
	}
	return item, err
}

func (uc *UseCase) validateReferences(ctx context.Context, classID, subjectID, teacherID string) error {
	if _, err := uc.classes.FindByID(ctx, classID); err != nil {
		uc.logError(err, "assignment_validate_class", logrus.Fields{"class_id": classID})
		return err
	}
	if _, err := uc.subjects.FindByID(ctx, subjectID); err != nil {
		uc.logError(err, "assignment_validate_subject", logrus.Fields{"subject_id": subjectID})
		return err
	}
	if _, err := uc.users.FindByID(ctx, teacherID); err != nil {
		uc.logError(err, "assignment_validate_teacher", logrus.Fields{"teacher_id": teacherID})
		return err
	}
	return nil
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

func isFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func trim(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}
