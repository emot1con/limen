package academic

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
)

type scheduleModel struct {
	ID            string    `gorm:"column:id;primaryKey"`
	ClassID       string    `gorm:"column:class_id"`
	SubjectID     string    `gorm:"column:subject_id"`
	TeacherUserID *string   `gorm:"column:teacher_user_id"`
	DayOfWeek     int16     `gorm:"column:day_of_week"`
	StartTime     string    `gorm:"column:start_time"`
	EndTime       string    `gorm:"column:end_time"`
	Kind          string    `gorm:"column:kind"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (scheduleModel) TableName() string {
	return "schedules"
}

type ScheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Create(ctx context.Context, schedule academicdomain.Schedule) error {
	model := scheduleModel{
		ID:            schedule.ID,
		ClassID:       schedule.ClassID,
		SubjectID:     schedule.SubjectID,
		TeacherUserID: schedule.TeacherUserID,
		DayOfWeek:     schedule.DayOfWeek,
		StartTime:     schedule.StartTime.Format("15:04:05"),
		EndTime:       schedule.EndTime.Format("15:04:05"),
		Kind:          string(schedule.Kind),
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *ScheduleRepository) FindByID(ctx context.Context, id string) (academicdomain.Schedule, error) {
	var model scheduleModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Schedule{}, domain.ErrNotFound
		}
		return academicdomain.Schedule{}, err
	}
	return r.toDomain(model), nil
}

func (r *ScheduleRepository) ListByClassID(ctx context.Context, classID string) ([]academicdomain.Schedule, error) {
	var models []scheduleModel
	err := r.db.WithContext(ctx).Where("class_id = ?", classID).Order("day_of_week, start_time").Find(&models).Error
	if err != nil {
		return nil, err
	}
	schedules := make([]academicdomain.Schedule, len(models))
	for i, m := range models {
		schedules[i] = r.toDomain(m)
	}
	return schedules, nil
}

func (r *ScheduleRepository) Update(ctx context.Context, schedule academicdomain.Schedule) error {
	result := r.db.WithContext(ctx).Model(&scheduleModel{}).Where("id = ?", schedule.ID).Updates(map[string]interface{}{
		"class_id":        schedule.ClassID,
		"subject_id":      schedule.SubjectID,
		"teacher_user_id": schedule.TeacherUserID,
		"day_of_week":     schedule.DayOfWeek,
		"start_time":      schedule.StartTime.Format("15:04:05"),
		"end_time":        schedule.EndTime.Format("15:04:05"),
		"kind":            string(schedule.Kind),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&scheduleModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ScheduleRepository) toDomain(model scheduleModel) academicdomain.Schedule {
	startTime, _ := time.Parse("15:04:05", model.StartTime)
	endTime, _ := time.Parse("15:04:05", model.EndTime)
	return academicdomain.Schedule{
		ID:            model.ID,
		ClassID:       model.ClassID,
		SubjectID:     model.SubjectID,
		TeacherUserID: model.TeacherUserID,
		DayOfWeek:     model.DayOfWeek,
		StartTime:     startTime,
		EndTime:       endTime,
		Kind:          academicdomain.ScheduleKind(model.Kind),
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ academicdomain.ScheduleRepository = (*ScheduleRepository)(nil)
