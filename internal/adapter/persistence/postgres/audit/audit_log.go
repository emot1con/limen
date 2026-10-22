package audit

import (
	"context"
	"encoding/json"
	"time"

	auditdomain "limen/internal/domain/audit"

	"gorm.io/gorm"
)

type auditLogModel struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement"`
	ActorUserID *string         `gorm:"column:actor_user_id"`
	EntityType  string          `gorm:"column:entity_type"`
	EntityID    string          `gorm:"column:entity_id"`
	Action      string          `gorm:"column:action"`
	BeforeData  json.RawMessage `gorm:"column:before_data;type:jsonb"`
	AfterData   json.RawMessage `gorm:"column:after_data;type:jsonb"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
}

func (auditLogModel) TableName() string {
	return "audit_logs"
}

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Append(ctx context.Context, log auditdomain.AuditLog) error {
	model := auditLogModel{
		ActorUserID: log.ActorUserID,
		EntityType:  log.EntityType,
		EntityID:    log.EntityID,
		Action:      log.Action,
		BeforeData:  log.BeforeData,
		AfterData:   log.AfterData,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *AuditLogRepository) ListByEntity(ctx context.Context, entityType, entityID string) ([]auditdomain.AuditLog, error) {
	var models []auditLogModel
	err := r.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ?", entityType, entityID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	logs := make([]auditdomain.AuditLog, len(models))
	for i, m := range models {
		logs[i] = r.toDomain(m)
	}
	return logs, nil
}

func (r *AuditLogRepository) ListByCreatedAt(ctx context.Context, start, end time.Time) ([]auditdomain.AuditLog, error) {
	var models []auditLogModel
	err := r.db.WithContext(ctx).Where("created_at >= ? AND created_at < ?", start, end).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	logs := make([]auditdomain.AuditLog, len(models))
	for i, m := range models {
		logs[i] = r.toDomain(m)
	}
	return logs, nil
}

func (r *AuditLogRepository) toDomain(model auditLogModel) auditdomain.AuditLog {
	return auditdomain.AuditLog{
		ID:          model.ID,
		ActorUserID: model.ActorUserID,
		EntityType:  model.EntityType,
		EntityID:    model.EntityID,
		Action:      model.Action,
		BeforeData:  model.BeforeData,
		AfterData:   model.AfterData,
		CreatedAt:   model.CreatedAt,
	}
}

var _ auditdomain.AuditLogRepository = (*AuditLogRepository)(nil)
