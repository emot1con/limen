package device

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"
)

type UseCase struct {
	repo attendancedomain.DeviceRepository
	now  func() time.Time
}

func NewUseCase(repo attendancedomain.DeviceRepository) *UseCase {
	return &UseCase{repo: repo, now: time.Now}
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

func (uc *UseCase) Create(ctx context.Context, input CreateInput) (CreateOutput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateOutput{}, domain.ErrInvalid
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
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
		return CreateOutput{}, err
	}
	saved, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return CreateOutput{}, err
	}
	return CreateOutput{Device: toDevice(saved), DeviceKey: plaintext}, nil
}

func (uc *UseCase) Get(ctx context.Context, id string) (Device, error) {
	item, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return Device{}, err
	}
	return toDevice(item), nil
}

func (uc *UseCase) List(ctx context.Context, page, pageSize int) (Page, error) {
	page, pageSize = normalizePage(page, pageSize)
	items, total, err := uc.repo.List(ctx, pageSize, (page-1)*pageSize)
	if err != nil {
		return Page{}, err
	}
	out := make([]Device, len(items))
	for i, item := range items {
		out[i] = toDevice(item)
	}
	return Page{Items: out, Total: total, Page: page, PageSize: pageSize}, nil
}

func (uc *UseCase) Update(ctx context.Context, input UpdateInput) (Device, error) {
	item, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
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
		return Device{}, err
	}
	updated, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return Device{}, err
	}
	return toDevice(updated), nil
}

func (uc *UseCase) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
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
