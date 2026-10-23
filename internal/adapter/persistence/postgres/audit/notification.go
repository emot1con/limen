package audit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domain "limen/internal/domain"
	auditdomain "limen/internal/domain/audit"

	"gorm.io/gorm"
)

type notificationModel struct {
	ID        string          `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	UserID    string          `gorm:"column:user_id"`
	Type      string          `gorm:"column:type"`
	Payload   json.RawMessage `gorm:"column:payload;type:jsonb"`
	ReadAt    *time.Time      `gorm:"column:read_at"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	DeletedAt gorm.DeletedAt  `gorm:"column:deleted_at;index"`
}

func (notificationModel) TableName() string {
	return "notifications"
}

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, notification auditdomain.Notification) (string, error) {
	model := notificationModel{
		ID:      notification.ID,
		UserID:  notification.UserID,
		Type:    notification.Type,
		Payload: notification.Payload,
		ReadAt:  notification.ReadAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *NotificationRepository) FindByID(ctx context.Context, id string) (auditdomain.Notification, error) {
	var model notificationModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return auditdomain.Notification{}, domain.ErrNotFound
		}
		return auditdomain.Notification{}, err
	}
	return r.toDomain(model), nil
}

func (r *NotificationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]auditdomain.Notification, error) {
	var models []notificationModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}
	notifications := make([]auditdomain.Notification, len(models))
	for i, m := range models {
		notifications[i] = r.toDomain(m)
	}
	return notifications, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string, readAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&notificationModel{}).Where("id = ? AND user_id = ?", id, userID).Update("read_at", readAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) toDomain(model notificationModel) auditdomain.Notification {
	return auditdomain.Notification{
		ID:        model.ID,
		UserID:    model.UserID,
		Type:      model.Type,
		Payload:   model.Payload,
		ReadAt:    model.ReadAt,
		CreatedAt: model.CreatedAt,
	}
}

var _ auditdomain.NotificationRepository = (*NotificationRepository)(nil)
