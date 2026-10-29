package academic

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	academicdomain "limen/internal/domain/academic"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (scheduleModel) TableName() string { return "schedules" }

type ScheduleRepository struct{ db *gorm.DB }

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository { return &ScheduleRepository{db: db} }

func (r *ScheduleRepository) Create(ctx context.Context, schedule academicdomain.Schedule) (string, error) {
	model := scheduleModel{ClassID: schedule.ClassID, SubjectID: schedule.SubjectID, TeacherUserID: schedule.TeacherUserID, DayOfWeek: schedule.DayOfWeek, StartTime: schedule.StartTime.Format("15:04:05"), EndTime: schedule.EndTime.Format("15:04:05"), Kind: string(schedule.Kind)}
	if err := r.db.WithContext(ctx).Omit("ID", "CreatedAt", "UpdatedAt").Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *ScheduleRepository) FindByID(ctx context.Context, id string) (academicdomain.Schedule, error) {
	var model scheduleModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return academicdomain.Schedule{}, domain.ErrNotFound
		}
		return academicdomain.Schedule{}, err
	}
	return r.toDomain(model), nil
}

func (r *ScheduleRepository) List(ctx context.Context, filter academicdomain.ScheduleFilter, limit, offset int) ([]academicdomain.Schedule, int64, error) {
	query := r.db.WithContext(ctx).Model(&scheduleModel{})
	if filter.ClassID != "" {
		query = query.Where("class_id = ?", filter.ClassID)
	}
	if filter.SubjectID != "" {
		query = query.Where("subject_id = ?", filter.SubjectID)
	}
	if filter.Kind != "" {
		query = query.Where("kind = ?", filter.Kind)
	}
	if len(filter.DayOfWeeks) > 0 {
		query = query.Where("day_of_week IN ?", filter.DayOfWeeks)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []scheduleModel
	if err := query.Order("day_of_week, start_time, id").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]academicdomain.Schedule, len(models))
	for i, model := range models {
		out[i] = r.toDomain(model)
	}
	return out, total, nil
}

func (r *ScheduleRepository) Update(ctx context.Context, schedule academicdomain.Schedule) error {
	result := r.db.WithContext(ctx).Model(&scheduleModel{}).Where("id = ?", schedule.ID).Updates(map[string]interface{}{"class_id": schedule.ClassID, "subject_id": schedule.SubjectID, "teacher_user_id": schedule.TeacherUserID, "day_of_week": schedule.DayOfWeek, "start_time": schedule.StartTime.Format("15:04:05"), "end_time": schedule.EndTime.Format("15:04:05"), "kind": string(schedule.Kind), "updated_at": gorm.Expr("now()")})
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
	return academicdomain.Schedule{ID: model.ID, ClassID: model.ClassID, SubjectID: model.SubjectID, TeacherUserID: model.TeacherUserID, DayOfWeek: model.DayOfWeek, StartTime: startTime, EndTime: endTime, Kind: academicdomain.ScheduleKind(model.Kind), CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}
}

var _ academicdomain.ScheduleRepository = (*ScheduleRepository)(nil)
