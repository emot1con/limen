package device

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"
	"limen/internal/platform/cache"
)

type UseCase struct {
	repo   attendancedomain.DeviceRepository
	cache  cache.Cache
	ttl    time.Duration
	now    func() time.Time
	logger *logrus.Logger
}

func NewUseCase(repo attendancedomain.DeviceRepository, c cache.Cache, ttl time.Duration, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{repo: repo, cache: c, ttl: ttl, now: time.Now, logger: logger}
}

type Device struct {
	ID         string
	Name       string
	Location   *string
	Serial     *string
	IsActive   bool
	LastSeenAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateInput struct {
	Name     string
	Location *string
	Serial   *string
	IsActive *bool
}

type CreateOutput struct {
	Device
	DeviceKey string
}

type UpdateInput struct {
	ID       string
	Name     *string
	Location *string
	Serial   *string
	IsActive *bool
}

type Page struct {
	Items    []Device
	Total    int64
	Page     int
	PageSize int
}

const groupDevices = "devices"

func (uc *UseCase) Create(ctx context.Context, input CreateInput) (CreateOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateOutput{}, domain.ErrInvalid
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		uc.logError(err, "device_generate_key", nil)
		return CreateOutput{}, err
	}
	plaintext := base64.RawURLEncoding.EncodeToString(key)
	sum := sha256.Sum256([]byte(plaintext))
	hash := hex.EncodeToString(sum[:])
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	id, err := uc.repo.Create(ctx, attendancedomain.Device{
		Name:          name,
		Location:      trimOptional(input.Location),
		Serial:        trimOptional(input.Serial),
		IsActive:      active,
		CredentialRef: &hash,
	})
	if err != nil {
		uc.logError(err, "device_create", nil)
		return CreateOutput{}, err
	}
	saved, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "device_get_created", logrus.Fields{"device_id": id})
		return CreateOutput{}, err
	}

	if uc.cache != nil {
		if _, err := uc.cache.BumpVersion(ctx, groupDevices); err != nil {
			uc.logger.WithError(err).Warn("cache invalidation failed")
		}
	}

	return CreateOutput{Device: toDevice(saved), DeviceKey: plaintext}, nil
}

func (uc *UseCase) Get(ctx context.Context, id string) (Device, error) {
	item, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "device_get", logrus.Fields{"device_id": id})
		return Device{}, err
	}
	return toDevice(item), nil
}

func (uc *UseCase) List(ctx context.Context, page, pageSize int, isActive *bool) (Page, error) {
	page, pageSize = normalizePage(page, pageSize)

	if uc.cache != nil {
		ver, _ := uc.cache.GetVersion(ctx, groupDevices)
		params := fmt.Sprintf("page=%d&pageSize=%d&isActive=%v", page, pageSize, isActive)
		key := cache.KeyWithHash(groupDevices, ver, params)

		var cached Page
		if hit, err := uc.cache.Get(ctx, key, &cached); err == nil && hit {
			return cached, nil
		}
	}

	items, total, err := uc.repo.List(ctx, pageSize, (page-1)*pageSize, isActive)
	if err != nil {
		uc.logError(err, "device_list", nil)
		return Page{}, err
	}
	out := make([]Device, len(items))
	for i, item := range items {
		out[i] = toDevice(item)
	}

	result := Page{Items: out, Total: total, Page: page, PageSize: pageSize}

	if uc.cache != nil {
		ver, _ := uc.cache.GetVersion(ctx, groupDevices)
		params := fmt.Sprintf("page=%d&pageSize=%d&isActive=%v", page, pageSize, isActive)
		key := cache.KeyWithHash(groupDevices, ver, params)
		_ = uc.cache.Set(ctx, key, result, uc.ttl)
	}

	return result, nil
}

func (uc *UseCase) Update(ctx context.Context, input UpdateInput) (Device, error) {
	item, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		uc.logError(err, "device_get_for_update", logrus.Fields{"device_id": input.ID})
		return Device{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return Device{}, domain.ErrInvalid
		}
		item.Name = name
	}
	if input.Location != nil {
		item.Location = trimOptional(input.Location)
	}
	if input.Serial != nil {
		item.Serial = trimOptional(input.Serial)
	}
	if input.IsActive != nil {
		item.IsActive = *input.IsActive
	}
	item.UpdatedAt = uc.now()
	if err := uc.repo.Update(ctx, item); err != nil {
		uc.logError(err, "device_update", logrus.Fields{"device_id": input.ID})
		return Device{}, err
	}

	if uc.cache != nil {
		if _, err := uc.cache.BumpVersion(ctx, groupDevices); err != nil {
			uc.logger.WithError(err).Warn("cache invalidation failed")
		}
	}

	updated, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		uc.logError(err, "device_get_updated", logrus.Fields{"device_id": input.ID})
		return Device{}, err
	}
	return toDevice(updated), nil
}

func (uc *UseCase) Delete(ctx context.Context, id string) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.logError(err, "device_delete", logrus.Fields{"device_id": id})
		return err
	}
	return nil
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}

func toDevice(item attendancedomain.Device) Device {
	return Device{ID: item.ID, Name: item.Name, Location: item.Location, Serial: item.Serial, IsActive: item.IsActive, LastSeenAt: item.LastSeenAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
