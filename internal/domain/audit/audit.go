package audit

import (
	"context"
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID          int64
	ActorUserID *string
	EntityType  string
	EntityID    string
	Action      string
	BeforeData  json.RawMessage
	AfterData   json.RawMessage
	CreatedAt   time.Time
}

type Notification struct {
	ID        string
	UserID    string
	Type      string
	Payload   json.RawMessage
	ReadAt    *time.Time
	CreatedAt time.Time
}

type AuditLogRepository interface {
	Append(context.Context, AuditLog) error
	ListByEntity(context.Context, string, string) ([]AuditLog, error)
	ListByCreatedAt(context.Context, time.Time, time.Time) ([]AuditLog, error)
}

type NotificationRepository interface {
	Create(context.Context, Notification) (string, error)
	FindByID(context.Context, string) (Notification, error)
	ListByUserID(context.Context, string, int, int) ([]Notification, error)
	MarkRead(context.Context, string, string, time.Time) error
}
