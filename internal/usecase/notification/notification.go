package notification

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	auditdomain "limen/internal/domain/audit"
)

type UseCase struct {
	repo   auditdomain.NotificationRepository
	now    func() time.Time
	logger *logrus.Logger
}

func NewUseCase(repo auditdomain.NotificationRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{repo: repo, now: time.Now, logger: logger}
}

type Notification struct {
	ID        string
	Type      string
	Payload   json.RawMessage
	ReadAt    *time.Time
	CreatedAt time.Time
}

type ListInput struct {
	UserID     string
	UnreadOnly bool
	Page       int
	PageSize   int
}

type ListOutput struct {
	Items    []Notification
	Total    int64
	Page     int
	PageSize int
}

type MarkReadInput struct {
	NotificationID string
	UserID         string
}

type CreateInput struct {
	UserID  string
	Type    string
	Payload json.RawMessage
}

func (uc *UseCase) Create(ctx context.Context, input CreateInput) (Notification, error) {
	if input.Type == "" {
		return Notification{}, domain.ErrInvalid
	}
	id, err := uc.repo.Create(ctx, auditdomain.Notification{
		UserID:  input.UserID,
		Type:    input.Type,
		Payload: input.Payload,
	})
	if err != nil {
		uc.logError(err, "notification_create", logrus.Fields{"user_id": input.UserID, "type": input.Type})
		return Notification{}, err
	}
	created, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "notification_find_created", logrus.Fields{"notification_id": id})
		return Notification{}, err
	}
	return toNotification(created), nil
}

func (uc *UseCase) List(ctx context.Context, input ListInput) (ListOutput, error) {
	page, pageSize := normalizePage(input.Page, input.PageSize)
	offset := (page - 1) * pageSize
	items, total, err := uc.repo.ListByUserID(ctx, input.UserID, input.UnreadOnly, pageSize, offset)
	if err != nil {
		uc.logError(err, "notification_list", logrus.Fields{"user_id": input.UserID})
		return ListOutput{}, err
	}
	notifications := make([]Notification, len(items))
	for i, item := range items {
		notifications[i] = toNotification(item)
	}
	return ListOutput{Items: notifications, Total: total, Page: page, PageSize: pageSize}, nil
}

func (uc *UseCase) MarkRead(ctx context.Context, input MarkReadInput) (Notification, error) {
	existing, err := uc.repo.FindByID(ctx, input.NotificationID)
	if err != nil {
		uc.logError(err, "notification_find", logrus.Fields{"notification_id": input.NotificationID})
		return Notification{}, err
	}
	if existing.UserID != input.UserID {
		return Notification{}, domain.ErrNotFound
	}
	if existing.ReadAt == nil {
		now := uc.now()
		if err := uc.repo.MarkRead(ctx, input.NotificationID, input.UserID, now); err != nil {
			uc.logError(err, "notification_mark_read", logrus.Fields{"notification_id": input.NotificationID, "user_id": input.UserID})
			return Notification{}, err
		}
		existing.ReadAt = &now
	}
	return toNotification(existing), nil
}

func toNotification(item auditdomain.Notification) Notification {
	return Notification{
		ID:        item.ID,
		Type:      item.Type,
		Payload:   item.Payload,
		ReadAt:    item.ReadAt,
		CreatedAt: item.CreatedAt,
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func (uc *UseCase) logError(err error, event string, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["event"] = event
	uc.logger.WithFields(fields).WithError(err).Error("notification use case error")
}
