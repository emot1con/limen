package attendance

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"

	"gorm.io/gorm"
)

type schoolSettingsModel struct {
	ID                   int16     `gorm:"column:id;primaryKey"`
	FirstEntryTime       string    `gorm:"column:first_entry_time"`
	Timezone             string    `gorm:"column:timezone"`
	LateToleranceMinutes int       `gorm:"column:late_tolerance_minutes"`
	UpdatedBy            *string   `gorm:"column:updated_by"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (schoolSettingsModel) TableName() string {
	return "school_settings"
}

type SchoolSettingsRepository struct {
	db *gorm.DB
}

func NewSchoolSettingsRepository(db *gorm.DB) *SchoolSettingsRepository {
	return &SchoolSettingsRepository{db: db}
}

func (r *SchoolSettingsRepository) Get(ctx context.Context) (attendancedomain.SchoolSettings, error) {
	var model schoolSettingsModel
	err := r.db.WithContext(ctx).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendancedomain.SchoolSettings{}, domain.ErrNotFound
		}
		return attendancedomain.SchoolSettings{}, err
	}
	return r.toDomain(model), nil
}

func (r *SchoolSettingsRepository) Update(ctx context.Context, settings attendancedomain.SchoolSettings) error {
	result := r.db.WithContext(ctx).Model(&schoolSettingsModel{}).Where("id = ?", settings.ID).Updates(map[string]interface{}{
		"first_entry_time":       settings.FirstEntryTime.Format("15:04:05"),
		"timezone":               settings.Timezone,
		"late_tolerance_minutes": settings.LateToleranceMinutes,
		"updated_by":             settings.UpdatedBy,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SchoolSettingsRepository) toDomain(model schoolSettingsModel) attendancedomain.SchoolSettings {
	firstEntryTime, _ := time.Parse("15:04:05", model.FirstEntryTime)
	return attendancedomain.SchoolSettings{
		ID:                   model.ID,
		FirstEntryTime:       firstEntryTime,
		Timezone:             model.Timezone,
		LateToleranceMinutes: model.LateToleranceMinutes,
		UpdatedBy:            model.UpdatedBy,
		UpdatedAt:            model.UpdatedAt,
	}
}

var _ attendancedomain.SchoolSettingsRepository = (*SchoolSettingsRepository)(nil)
