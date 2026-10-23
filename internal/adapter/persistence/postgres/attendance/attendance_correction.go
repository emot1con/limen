package attendance

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"

	"gorm.io/gorm"
)

type attendanceCorrectionModel struct {
	ID                 string                             `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	AttendanceRecordID string                             `gorm:"column:attendance_record_id"`
	RequestedBy        string                             `gorm:"column:requested_by"`
	Reason             string                             `gorm:"column:reason"`
	OldStatus          *attendancedomain.AttendanceStatus `gorm:"column:old_status"`
	NewStatus          attendancedomain.AttendanceStatus  `gorm:"column:new_status"`
	Status             attendancedomain.CorrectionStatus  `gorm:"column:status"`
	ReviewNote         *string                            `gorm:"column:review_note"`
	ReviewedBy         *string                            `gorm:"column:reviewed_by"`
	ReviewedAt         *time.Time                         `gorm:"column:reviewed_at"`
	CreatedAt          time.Time                          `gorm:"column:created_at"`
}

func (attendanceCorrectionModel) TableName() string {
	return "attendance_corrections"
}

type AttendanceCorrectionRepository struct {
	db *gorm.DB
}

func NewAttendanceCorrectionRepository(db *gorm.DB) *AttendanceCorrectionRepository {
	return &AttendanceCorrectionRepository{db: db}
}

func (r *AttendanceCorrectionRepository) Create(ctx context.Context, correction attendancedomain.AttendanceCorrection) (string, error) {
	model := attendanceCorrectionModel{
		ID:                 correction.ID,
		AttendanceRecordID: correction.AttendanceRecordID,
		RequestedBy:        correction.RequestedBy,
		Reason:             correction.Reason,
		OldStatus:          correction.OldStatus,
		NewStatus:          correction.NewStatus,
		Status:             correction.Status,
		ReviewNote:         correction.ReviewNote,
		ReviewedBy:         correction.ReviewedBy,
		ReviewedAt:         correction.ReviewedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *AttendanceCorrectionRepository) FindByID(ctx context.Context, id string) (attendancedomain.AttendanceCorrection, error) {
	var model attendanceCorrectionModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendancedomain.AttendanceCorrection{}, domain.ErrNotFound
		}
		return attendancedomain.AttendanceCorrection{}, err
	}
	return r.toDomain(model), nil
}

func (r *AttendanceCorrectionRepository) Update(ctx context.Context, correction attendancedomain.AttendanceCorrection) error {
	result := r.db.WithContext(ctx).Model(&attendanceCorrectionModel{}).Where("id = ?", correction.ID).Updates(map[string]interface{}{
		"attendance_record_id": correction.AttendanceRecordID,
		"requested_by":         correction.RequestedBy,
		"reason":               correction.Reason,
		"old_status":           correction.OldStatus,
		"new_status":           correction.NewStatus,
		"status":               correction.Status,
		"review_note":          correction.ReviewNote,
		"reviewed_by":          correction.ReviewedBy,
		"reviewed_at":          correction.ReviewedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AttendanceCorrectionRepository) ListByRecordID(ctx context.Context, attendanceRecordID string) ([]attendancedomain.AttendanceCorrection, error) {
	var models []attendanceCorrectionModel
	err := r.db.WithContext(ctx).Where("attendance_record_id = ?", attendanceRecordID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	corrections := make([]attendancedomain.AttendanceCorrection, len(models))
	for i, model := range models {
		corrections[i] = r.toDomain(model)
	}
	return corrections, nil
}

func (r *AttendanceCorrectionRepository) ListByStatus(ctx context.Context, status attendancedomain.CorrectionStatus) ([]attendancedomain.AttendanceCorrection, error) {
	var models []attendanceCorrectionModel
	err := r.db.WithContext(ctx).Where("status = ?", status).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	corrections := make([]attendancedomain.AttendanceCorrection, len(models))
	for i, model := range models {
		corrections[i] = r.toDomain(model)
	}
	return corrections, nil
}

func (r *AttendanceCorrectionRepository) toDomain(model attendanceCorrectionModel) attendancedomain.AttendanceCorrection {
	return attendancedomain.AttendanceCorrection{
		ID:                 model.ID,
		AttendanceRecordID: model.AttendanceRecordID,
		RequestedBy:        model.RequestedBy,
		Reason:             model.Reason,
		OldStatus:          model.OldStatus,
		NewStatus:          model.NewStatus,
		Status:             model.Status,
		ReviewNote:         model.ReviewNote,
		ReviewedBy:         model.ReviewedBy,
		ReviewedAt:         model.ReviewedAt,
		CreatedAt:          model.CreatedAt,
	}
}

var _ attendancedomain.AttendanceCorrectionRepository = (*AttendanceCorrectionRepository)(nil)
