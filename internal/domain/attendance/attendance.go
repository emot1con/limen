package attendance

import (
	"context"
	"encoding/json"
	"time"
)

type SubjectType string

type AttendanceEventType string

type AttendanceStatus string

type AttendanceSource string

type CorrectionStatus string

const (
	SubjectTypeStudent SubjectType = "student"
	SubjectTypeTeacher SubjectType = "teacher"

	AttendanceEventCheckIn  AttendanceEventType = "check_in"
	AttendanceEventCheckOut AttendanceEventType = "check_out"

	AttendanceStatusPresent AttendanceStatus = "hadir"
	AttendanceStatusLate    AttendanceStatus = "terlambat"
	AttendanceStatusExcused AttendanceStatus = "izin"
	AttendanceStatusSick    AttendanceStatus = "sakit"
	AttendanceStatusAbsent  AttendanceStatus = "alpa"

	AttendanceSourceDevice     AttendanceSource = "device"
	AttendanceSourceManual     AttendanceSource = "manual"
	AttendanceSourceCorrection AttendanceSource = "correction"

	CorrectionStatusPending  CorrectionStatus = "pending"
	CorrectionStatusApproved CorrectionStatus = "approved"
	CorrectionStatusRejected CorrectionStatus = "rejected"
)

type SchoolSettings struct {
	ID                   int16
	FirstEntryTime       time.Time
	Timezone             string
	LateToleranceMinutes int
	UpdatedBy            *string
	UpdatedAt            time.Time
}

type Device struct {
	ID            string
	Name          string
	Location      *string
	Serial        *string
	IsActive      bool
	CredentialRef *string
	LastSeenAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BiometricIdentity struct {
	ID            string
	Identifier    string
	SubjectUserID string
	SubjectType   SubjectType
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AttendanceEvent struct {
	ID            int64
	DeviceID      string
	Identifier    string
	SubjectUserID *string
	EventType     AttendanceEventType
	OccurredAt    time.Time
	ReceivedAt    time.Time
	RawPayload    json.RawMessage
}

type AttendanceRecord struct {
	ID               string
	SubjectUserID    string
	SubjectType      SubjectType
	AttendanceDate   time.Time
	CheckInAt        *time.Time
	CheckOutAt       *time.Time
	Status           AttendanceStatus
	Source           AttendanceSource
	CheckInDeviceID  *string
	CheckOutDeviceID *string
	LateMinutes      *int
	Note             *string
	CreatedBy        *string
	UpdatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AttendanceCorrection struct {
	ID                 string
	AttendanceRecordID string
	RequestedBy        string
	Reason             string
	OldStatus          *AttendanceStatus
	NewStatus          AttendanceStatus
	Status             CorrectionStatus
	ReviewNote         *string
	ReviewedBy         *string
	ReviewedAt         *time.Time
	CreatedAt          time.Time
}

type SchoolSettingsRepository interface {
	Get(context.Context) (SchoolSettings, error)
	Update(context.Context, SchoolSettings) error
}

type DeviceRepository interface {
	Create(context.Context, Device) error
	FindByID(context.Context, string) (Device, error)
	Update(context.Context, Device) error
	List(context.Context) ([]Device, error)
}

type BiometricIdentityRepository interface {
	Create(context.Context, BiometricIdentity) error
	FindByIdentifier(context.Context, string) (BiometricIdentity, error)
	ListBySubjectUserID(context.Context, string) ([]BiometricIdentity, error)
	Update(context.Context, BiometricIdentity) error
}

type AttendanceEventRepository interface {
	Append(context.Context, AttendanceEvent) error
	ListBySubjectUserID(context.Context, string, time.Time, time.Time) ([]AttendanceEvent, error)
}

type AttendanceRecordRepository interface {
	Create(context.Context, AttendanceRecord) error
	FindByID(context.Context, string) (AttendanceRecord, error)
	FindBySubjectAndDate(context.Context, string, time.Time) (AttendanceRecord, error)
	Update(context.Context, AttendanceRecord) error
	ListByDate(context.Context, time.Time) ([]AttendanceRecord, error)
	ListBySubjectUserID(context.Context, string, time.Time, time.Time) ([]AttendanceRecord, error)
}

type AttendanceCorrectionRepository interface {
	Create(context.Context, AttendanceCorrection) error
	FindByID(context.Context, string) (AttendanceCorrection, error)
	Update(context.Context, AttendanceCorrection) error
	ListByRecordID(context.Context, string) ([]AttendanceCorrection, error)
	ListByStatus(context.Context, CorrectionStatus) ([]AttendanceCorrection, error)
}
