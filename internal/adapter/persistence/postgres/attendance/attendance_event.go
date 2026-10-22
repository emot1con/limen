package attendance

import (
	"context"
	"encoding/json"
	"time"

	attendancedomain "limen/internal/domain/attendance"

	"gorm.io/gorm"
)

type attendanceEventModel struct {
	ID            int64                                `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID      string                               `gorm:"column:device_id"`
	Identifier    string                               `gorm:"column:identifier"`
	SubjectUserID *string                              `gorm:"column:subject_user_id"`
	EventType     attendancedomain.AttendanceEventType `gorm:"column:event_type"`
	OccurredAt    time.Time                            `gorm:"column:occurred_at"`
	ReceivedAt    time.Time                            `gorm:"column:received_at"`
	RawPayload    json.RawMessage                      `gorm:"column:raw_payload;type:jsonb"`
}

func (attendanceEventModel) TableName() string {
	return "attendance_events"
}

type AttendanceEventRepository struct {
	db *gorm.DB
}

func NewAttendanceEventRepository(db *gorm.DB) *AttendanceEventRepository {
	return &AttendanceEventRepository{db: db}
}

func (r *AttendanceEventRepository) Append(ctx context.Context, event attendancedomain.AttendanceEvent) error {
	model := attendanceEventModel{
		ID:            event.ID,
		DeviceID:      event.DeviceID,
		Identifier:    event.Identifier,
		SubjectUserID: event.SubjectUserID,
		EventType:     event.EventType,
		OccurredAt:    event.OccurredAt,
		ReceivedAt:    event.ReceivedAt,
		RawPayload:    event.RawPayload,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *AttendanceEventRepository) ListBySubjectUserID(ctx context.Context, subjectUserID string, startTime, endTime time.Time) ([]attendancedomain.AttendanceEvent, error) {
	var models []attendanceEventModel
	err := r.db.WithContext(ctx).
		Where("subject_user_id = ? AND occurred_at >= ? AND occurred_at < ?", subjectUserID, startTime, endTime).
		Order("occurred_at, id").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	events := make([]attendancedomain.AttendanceEvent, len(models))
	for i, model := range models {
		events[i] = r.toDomain(model)
	}
	return events, nil
}

func (r *AttendanceEventRepository) toDomain(model attendanceEventModel) attendancedomain.AttendanceEvent {
	return attendancedomain.AttendanceEvent{
		ID:            model.ID,
		DeviceID:      model.DeviceID,
		Identifier:    model.Identifier,
		SubjectUserID: model.SubjectUserID,
		EventType:     model.EventType,
		OccurredAt:    model.OccurredAt,
		ReceivedAt:    model.ReceivedAt,
		RawPayload:    model.RawPayload,
	}
}

var _ attendancedomain.AttendanceEventRepository = (*AttendanceEventRepository)(nil)
