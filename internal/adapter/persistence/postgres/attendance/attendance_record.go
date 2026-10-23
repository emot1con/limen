package attendance

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"

	"gorm.io/gorm"
)

type attendanceRecordModel struct {
	ID               string                            `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	SubjectUserID    string                            `gorm:"column:subject_user_id"`
	SubjectType      attendancedomain.SubjectType      `gorm:"column:subject_type"`
	AttendanceDate   time.Time                         `gorm:"column:attendance_date;type:date"`
	CheckInAt        *time.Time                        `gorm:"column:check_in_at"`
	CheckOutAt       *time.Time                        `gorm:"column:check_out_at"`
	Status           attendancedomain.AttendanceStatus `gorm:"column:status"`
	Source           attendancedomain.AttendanceSource `gorm:"column:source"`
	CheckInDeviceID  *string                           `gorm:"column:check_in_device_id"`
	CheckOutDeviceID *string                           `gorm:"column:check_out_device_id"`
	LateMinutes      *int                              `gorm:"column:late_minutes"`
	Note             *string                           `gorm:"column:note"`
	CreatedBy        *string                           `gorm:"column:created_by"`
	UpdatedBy        *string                           `gorm:"column:updated_by"`
	CreatedAt        time.Time                         `gorm:"column:created_at"`
	UpdatedAt        time.Time                         `gorm:"column:updated_at"`
	DeletedAt        gorm.DeletedAt                    `gorm:"column:deleted_at;index"`
}

func (attendanceRecordModel) TableName() string {
	return "attendance_records"
}

type AttendanceRecordRepository struct {
	db *gorm.DB
}

func NewAttendanceRecordRepository(db *gorm.DB) *AttendanceRecordRepository {
	return &AttendanceRecordRepository{db: db}
}

func (r *AttendanceRecordRepository) Create(ctx context.Context, record attendancedomain.AttendanceRecord) (string, error) {
	model := attendanceRecordModel{
		ID:               record.ID,
		SubjectUserID:    record.SubjectUserID,
		SubjectType:      record.SubjectType,
		AttendanceDate:   record.AttendanceDate,
		CheckInAt:        record.CheckInAt,
		CheckOutAt:       record.CheckOutAt,
		Status:           record.Status,
		Source:           record.Source,
		CheckInDeviceID:  record.CheckInDeviceID,
		CheckOutDeviceID: record.CheckOutDeviceID,
		LateMinutes:      record.LateMinutes,
		Note:             record.Note,
		CreatedBy:        record.CreatedBy,
		UpdatedBy:        record.UpdatedBy,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *AttendanceRecordRepository) FindByID(ctx context.Context, id string) (attendancedomain.AttendanceRecord, error) {
	var model attendanceRecordModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendancedomain.AttendanceRecord{}, domain.ErrNotFound
		}
		return attendancedomain.AttendanceRecord{}, err
	}
	return r.toDomain(model), nil
}

func (r *AttendanceRecordRepository) FindBySubjectAndDate(ctx context.Context, subjectUserID string, attendanceDate time.Time) (attendancedomain.AttendanceRecord, error) {
	var model attendanceRecordModel
	err := r.db.WithContext(ctx).Where("subject_user_id = ? AND attendance_date = ?", subjectUserID, attendanceDate.Format("2006-01-02")).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendancedomain.AttendanceRecord{}, domain.ErrNotFound
		}
		return attendancedomain.AttendanceRecord{}, err
	}
	return r.toDomain(model), nil
}

func (r *AttendanceRecordRepository) Update(ctx context.Context, record attendancedomain.AttendanceRecord) error {
	result := r.db.WithContext(ctx).Model(&attendanceRecordModel{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
		"subject_user_id":     record.SubjectUserID,
		"subject_type":        record.SubjectType,
		"attendance_date":     record.AttendanceDate,
		"check_in_at":         record.CheckInAt,
		"check_out_at":        record.CheckOutAt,
		"status":              record.Status,
		"source":              record.Source,
		"check_in_device_id":  record.CheckInDeviceID,
		"check_out_device_id": record.CheckOutDeviceID,
		"late_minutes":        record.LateMinutes,
		"note":                record.Note,
		"updated_by":          record.UpdatedBy,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AttendanceRecordRepository) ListByDate(ctx context.Context, attendanceDate time.Time) ([]attendancedomain.AttendanceRecord, error) {
	var models []attendanceRecordModel
	err := r.db.WithContext(ctx).Where("attendance_date = ?", attendanceDate.Format("2006-01-02")).Order("subject_user_id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	records := make([]attendancedomain.AttendanceRecord, len(models))
	for i, model := range models {
		records[i] = r.toDomain(model)
	}
	return records, nil
}

func (r *AttendanceRecordRepository) ListBySubjectUserID(ctx context.Context, subjectUserID string, startDate, endDate time.Time) ([]attendancedomain.AttendanceRecord, error) {
	var models []attendanceRecordModel
	err := r.db.WithContext(ctx).
		Where("subject_user_id = ? AND attendance_date >= ? AND attendance_date < ?", subjectUserID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Order("attendance_date").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	records := make([]attendancedomain.AttendanceRecord, len(models))
	for i, model := range models {
		records[i] = r.toDomain(model)
	}
	return records, nil
}

func (r *AttendanceRecordRepository) toDomain(model attendanceRecordModel) attendancedomain.AttendanceRecord {
	return attendancedomain.AttendanceRecord{
		ID:               model.ID,
		SubjectUserID:    model.SubjectUserID,
		SubjectType:      model.SubjectType,
		AttendanceDate:   model.AttendanceDate,
		CheckInAt:        model.CheckInAt,
		CheckOutAt:       model.CheckOutAt,
		Status:           model.Status,
		Source:           model.Source,
		CheckInDeviceID:  model.CheckInDeviceID,
		CheckOutDeviceID: model.CheckOutDeviceID,
		LateMinutes:      model.LateMinutes,
		Note:             model.Note,
		CreatedBy:        model.CreatedBy,
		UpdatedBy:        model.UpdatedBy,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

var _ attendancedomain.AttendanceRecordRepository = (*AttendanceRecordRepository)(nil)
