package assessment

import (
	"context"
	"math"
	"strings"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	assessmentdomain "limen/internal/domain/assessment"
	userdomain "limen/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	assessments assessmentdomain.AssessmentRepository
	scores      assessmentdomain.AssessmentScoreRepository
	classes     academicdomain.ClassRepository
	subjects    academicdomain.SubjectRepository
	students    academicdomain.StudentRepository
	users       userdomain.UserRepository
	logger      *logrus.Logger
}

func NewUseCase(assessments assessmentdomain.AssessmentRepository, scores assessmentdomain.AssessmentScoreRepository, classes academicdomain.ClassRepository, subjects academicdomain.SubjectRepository, students academicdomain.StudentRepository, users userdomain.UserRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{assessments: assessments, scores: scores, classes: classes, subjects: subjects, students: students, users: users, logger: logger}
}

type CreateInput struct {
	ClassID, SubjectID, Kind, Period, TeacherUserID string
	KKM                                             *float64
}

func (uc *UseCase) List(ctx context.Context, filter assessmentdomain.AssessmentFilter, page, pageSize int) ([]assessmentdomain.Assessment, int64, error) {
	items, total, err := uc.assessments.List(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "assessment_list", nil)
	}
	return items, total, err
}

func (uc *UseCase) Create(ctx context.Context, input CreateInput) (assessmentdomain.Assessment, error) {
	input.Kind, input.Period = strings.TrimSpace(input.Kind), strings.TrimSpace(input.Period)
	if input.ClassID == "" || input.SubjectID == "" || input.TeacherUserID == "" || input.Kind == "" || input.Period == "" || (input.KKM != nil && (!isFinite(*input.KKM) || *input.KKM < 0)) {
		return assessmentdomain.Assessment{}, domain.ErrInvalid
	}
	if err := uc.validateReferences(ctx, input.ClassID, input.SubjectID, input.TeacherUserID); err != nil {
		return assessmentdomain.Assessment{}, err
	}
	id, err := uc.assessments.Create(ctx, assessmentdomain.Assessment{ClassID: input.ClassID, SubjectID: input.SubjectID, TeacherUserID: input.TeacherUserID, Kind: input.Kind, KKM: input.KKM, Period: input.Period})
	if err != nil {
		uc.logError(err, "assessment_create", nil)
		return assessmentdomain.Assessment{}, err
	}
	item, err := uc.assessments.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "assessment_get_created", logrus.Fields{"assessment_id": id})
	}
	return item, err
}

func (uc *UseCase) ListScores(ctx context.Context, assessmentID string, page, pageSize int) ([]assessmentdomain.ScoreDetail, int64, error) {
	if _, err := uc.assessments.FindByID(ctx, assessmentID); err != nil {
		uc.logError(err, "assessment_get_for_scores", logrus.Fields{"assessment_id": assessmentID})
		return nil, 0, err
	}
	items, total, err := uc.scores.List(ctx, assessmentID, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "assessment_list_scores", logrus.Fields{"assessment_id": assessmentID})
	}
	return items, total, err
}

func (uc *UseCase) UpsertScore(ctx context.Context, assessmentID, studentUserID, updatedBy string, score float64) (assessmentdomain.ScoreDetail, error) {
	if !isFinite(score) {
		return assessmentdomain.ScoreDetail{}, domain.ErrInvalid
	}

	assessment, err := uc.assessments.FindByID(ctx, assessmentID)
	if err != nil {
		uc.logError(err, "assessment_get_for_score", logrus.Fields{"assessment_id": assessmentID})
		return assessmentdomain.ScoreDetail{}, err
	}
	student, err := uc.students.FindByUserID(ctx, studentUserID)
	if err != nil {
		uc.logError(err, "assessment_get_student", logrus.Fields{"student_id": studentUserID})
		return assessmentdomain.ScoreDetail{}, err
	}
	if student.ClassID == nil || *student.ClassID != assessment.ClassID {
		return assessmentdomain.ScoreDetail{}, domain.ErrInvalid
	}
	if err := uc.scores.Upsert(ctx, assessmentdomain.AssessmentScore{AssessmentID: assessmentID, StudentUserID: studentUserID, Score: score, UpdatedBy: optional(updatedBy)}); err != nil {
		uc.logError(err, "assessment_upsert_score", logrus.Fields{"assessment_id": assessmentID, "student_id": studentUserID})
		return assessmentdomain.ScoreDetail{}, err
	}
	item, err := uc.scores.FindDetail(ctx, assessmentID, studentUserID)
	if err != nil {
		uc.logError(err, "assessment_get_updated_score", logrus.Fields{"assessment_id": assessmentID, "student_id": studentUserID})
	}
	return item, err
}

func (uc *UseCase) validateReferences(ctx context.Context, classID, subjectID, teacherID string) error {
	if _, err := uc.classes.FindByID(ctx, classID); err != nil {
		uc.logError(err, "assessment_validate_class", logrus.Fields{"class_id": classID})
		return err
	}
	if _, err := uc.subjects.FindByID(ctx, subjectID); err != nil {
		uc.logError(err, "assessment_validate_subject", logrus.Fields{"subject_id": subjectID})
		return err
	}
	if _, err := uc.users.FindByID(ctx, teacherID); err != nil {
		uc.logError(err, "assessment_validate_teacher", logrus.Fields{"teacher_id": teacherID})
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
