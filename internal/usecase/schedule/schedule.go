package schedule

import (
	"context"
	"strings"
	"time"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	userdomain "limen/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	schedules academicdomain.ScheduleRepository
	classes   academicdomain.ClassRepository
	subjects  academicdomain.SubjectRepository
	users     userdomain.UserRepository
	logger    *logrus.Logger
}

func NewUseCase(schedules academicdomain.ScheduleRepository, classes academicdomain.ClassRepository, subjects academicdomain.SubjectRepository, users userdomain.UserRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{schedules: schedules, classes: classes, subjects: subjects, users: users, logger: logger}
}

type ScheduleInput struct {
	ClassID, SubjectID, StartTime, EndTime string
	TeacherUserID                          *string
	DayOfWeek                              int16
	Kind                                   academicdomain.ScheduleKind
	TeacherSet                             bool
}

func (uc *UseCase) ListSchedules(ctx context.Context, filter academicdomain.ScheduleFilter, page, pageSize int) ([]academicdomain.Schedule, int64, error) {
	items, total, err := uc.schedules.List(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "schedule_list", nil)
	}
	return items, total, err
}

func (uc *UseCase) CreateSchedule(ctx context.Context, input ScheduleInput) (academicdomain.Schedule, error) {
	schedule, err := uc.scheduleFromInput(academicdomain.Schedule{}, input, true, true, true, true, true, true)
	if err != nil {
		return academicdomain.Schedule{}, err
	}
	if !validScheduleKind(input.Kind) {
		return academicdomain.Schedule{}, domain.ErrInvalid
	}
	schedule.Kind = input.Kind
	if !schedule.StartTime.Before(schedule.EndTime) {
		return academicdomain.Schedule{}, domain.ErrInvalid
	}
	if err := uc.validateScheduleReferences(ctx, schedule); err != nil {
		return academicdomain.Schedule{}, err
	}
	id, err := uc.schedules.Create(ctx, schedule)
	if err != nil {
		uc.logError(err, "schedule_create", nil)
		return academicdomain.Schedule{}, err
	}
	item, err := uc.schedules.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "schedule_get_created", logrus.Fields{"schedule_id": id})
	}
	return item, err
}

func (uc *UseCase) UpdateSchedule(ctx context.Context, id string, input ScheduleInput, classSet, subjectSet, teacherSet, daySet, startSet, endSet, kindSet bool) (academicdomain.Schedule, error) {
	item, err := uc.schedules.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "schedule_get", logrus.Fields{"schedule_id": id})
		return item, err
	}
	item, err = uc.scheduleFromInput(item, input, classSet, subjectSet, teacherSet, daySet, startSet, endSet)
	if err != nil {
		return item, err
	}
	if kindSet {
		if !validScheduleKind(input.Kind) {
			return item, domain.ErrInvalid
		}
		item.Kind = input.Kind
	}
	if !item.StartTime.Before(item.EndTime) {
		return item, domain.ErrInvalid
	}
	if err := uc.validateScheduleReferences(ctx, item); err != nil {
		return item, err
	}
	if err := uc.schedules.Update(ctx, item); err != nil {
		uc.logError(err, "schedule_update", logrus.Fields{"schedule_id": id})
		return item, err
	}
	item, err = uc.schedules.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "schedule_get_updated", logrus.Fields{"schedule_id": id})
	}
	return item, err
}

func (uc *UseCase) DeleteSchedule(ctx context.Context, id string) error {
	if err := uc.schedules.Delete(ctx, id); err != nil {
		uc.logError(err, "schedule_delete", logrus.Fields{"schedule_id": id})
		return err
	}
	return nil
}

func (uc *UseCase) scheduleFromInput(item academicdomain.Schedule, input ScheduleInput, classSet, subjectSet, teacherSet, daySet, startSet, endSet bool) (academicdomain.Schedule, error) {
	if classSet {
		item.ClassID = strings.TrimSpace(input.ClassID)
	}
	if subjectSet {
		item.SubjectID = strings.TrimSpace(input.SubjectID)
	}
	if teacherSet {
		item.TeacherUserID = trim(input.TeacherUserID)
	}
	if daySet {
		if input.DayOfWeek < 1 || input.DayOfWeek > 7 {
			return item, domain.ErrInvalid
		}
		item.DayOfWeek = input.DayOfWeek
	}
	if startSet {
		value, err := time.Parse("15:04", strings.TrimSpace(input.StartTime))
		if err != nil {
			return item, domain.ErrInvalid
		}
		item.StartTime = value
	}
	if endSet {
		value, err := time.Parse("15:04", strings.TrimSpace(input.EndTime))
		if err != nil {
			return item, domain.ErrInvalid
		}
		item.EndTime = value
	}
	if item.ClassID == "" || item.SubjectID == "" {
		return item, domain.ErrInvalid
	}
	return item, nil
}

func (uc *UseCase) validateScheduleReferences(ctx context.Context, item academicdomain.Schedule) error {
	if _, err := uc.classes.FindByID(ctx, item.ClassID); err != nil {
		uc.logError(err, "schedule_validate_class", logrus.Fields{"class_id": item.ClassID})
		return err
	}
	if _, err := uc.subjects.FindByID(ctx, item.SubjectID); err != nil {
		uc.logError(err, "schedule_validate_subject", logrus.Fields{"subject_id": item.SubjectID})
		return err
	}
	if item.TeacherUserID != nil {
		if _, err := uc.users.FindByID(ctx, *item.TeacherUserID); err != nil {
			uc.logError(err, "schedule_validate_teacher", logrus.Fields{"teacher_id": *item.TeacherUserID})
			return err
		}
	}
	return nil
}

func validScheduleKind(kind academicdomain.ScheduleKind) bool {
	return kind == academicdomain.ScheduleKindLesson || kind == academicdomain.ScheduleKindExam || kind == academicdomain.ScheduleKindTryout
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
