package aptitude

import (
	"context"
	"fmt"
	"strings"
	"time"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	counselingdomain "limen/internal/domain/counseling"
	"limen/internal/platform/cache"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	aptitudes     counselingdomain.AptitudeResultRepository
	targetMajors  counselingdomain.TargetMajorRepository
	passingGrades counselingdomain.PassingGradeRepository
	students      academicdomain.StudentRepository
	cache         cache.Cache
	ttl           time.Duration
	logger        *logrus.Logger
}

func NewUseCase(aptitudes counselingdomain.AptitudeResultRepository, targetMajors counselingdomain.TargetMajorRepository, passingGrades counselingdomain.PassingGradeRepository, students academicdomain.StudentRepository, c cache.Cache, ttl time.Duration, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{aptitudes: aptitudes, targetMajors: targetMajors, passingGrades: passingGrades, students: students, cache: c, ttl: ttl, logger: logger}
}

func (uc *UseCase) ListAptitudeResults(ctx context.Context, studentUserID string) ([]counselingdomain.AptitudeResult, error) {
	if err := uc.ensureStudent(ctx, studentUserID); err != nil {
		return nil, err
	}
	items, err := uc.aptitudes.ListByStudentUserID(ctx, studentUserID)
	if err != nil {
		uc.logError(err, "aptitude_list_results", logrus.Fields{"student_id": studentUserID})
	}
	return items, err
}

func (uc *UseCase) AddTargetMajor(ctx context.Context, studentUserID, major string) (counselingdomain.TargetMajor, error) {
	major = strings.TrimSpace(major)
	if major == "" {
		return counselingdomain.TargetMajor{}, domain.ErrInvalid
	}
	if err := uc.ensureStudent(ctx, studentUserID); err != nil {
		return counselingdomain.TargetMajor{}, err
	}
	id, err := uc.targetMajors.Create(ctx, counselingdomain.TargetMajor{StudentUserID: studentUserID, Major: major})
	if err != nil {
		uc.logError(err, "aptitude_create_target_major", logrus.Fields{"student_id": studentUserID})
		return counselingdomain.TargetMajor{}, err
	}
	item, err := uc.targetMajors.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "aptitude_get_created_target_major", logrus.Fields{"target_major_id": id})
	}
	return item, err
}

const groupPassingGrades = "passing-grades"

func (uc *UseCase) ListPassingGrades(ctx context.Context, filter counselingdomain.PassingGradeFilter, page, pageSize int) ([]counselingdomain.PassingGrade, int64, error) {
	filter.University = strings.TrimSpace(filter.University)
	filter.Major = strings.TrimSpace(filter.Major)

	if uc.cache != nil {
		ver, _ := uc.cache.GetVersion(ctx, groupPassingGrades)
		params := fmt.Sprintf("uni=%s&major=%s&page=%d&pageSize=%d", filter.University, filter.Major, page, pageSize)
		key := cache.KeyWithHash(groupPassingGrades, ver, params)

		var cached struct {
			Items []counselingdomain.PassingGrade
			Total int64
		}
		if hit, err := uc.cache.Get(ctx, key, &cached); err == nil && hit {
			return cached.Items, cached.Total, nil
		}
	}

	items, total, err := uc.passingGrades.List(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "aptitude_list_passing_grades", nil)
		return items, total, err
	}

	if uc.cache != nil {
		ver, _ := uc.cache.GetVersion(ctx, groupPassingGrades)
		params := fmt.Sprintf("uni=%s&major=%s&page=%d&pageSize=%d", filter.University, filter.Major, page, pageSize)
		key := cache.KeyWithHash(groupPassingGrades, ver, params)
		_ = uc.cache.Set(ctx, key, struct {
			Items []counselingdomain.PassingGrade
			Total int64
		}{items, total}, uc.ttl)
	}

	return items, total, nil
}

func (uc *UseCase) GetPassingGrade(ctx context.Context, id string) (counselingdomain.PassingGrade, error) {
	if uc.cache != nil {
		ver, _ := uc.cache.GetVersion(ctx, groupPassingGrades)
		key := cache.Key(groupPassingGrades, ver, "id:"+id)

		var cached counselingdomain.PassingGrade
		if hit, err := uc.cache.Get(ctx, key, &cached); err == nil && hit {
			return cached, nil
		}
	}

	item, err := uc.passingGrades.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "aptitude_get_passing_grade", logrus.Fields{"passing_grade_id": id})
		return item, err
	}

	if uc.cache != nil {
		ver, _ := uc.cache.GetVersion(ctx, groupPassingGrades)
		key := cache.Key(groupPassingGrades, ver, "id:"+id)
		_ = uc.cache.Set(ctx, key, item, uc.ttl)
	}

	return item, nil
}

func (uc *UseCase) ensureStudent(ctx context.Context, studentUserID string) error {
	if _, err := uc.students.FindByUserID(ctx, studentUserID); err != nil {
		uc.logError(err, "aptitude_validate_student", logrus.Fields{"student_id": studentUserID})
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
