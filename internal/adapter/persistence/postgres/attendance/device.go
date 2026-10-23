package attendance

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"

	"gorm.io/gorm"
)

type deviceModel struct {
	ID            string     `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	Name          string     `gorm:"column:name"`
	Location      *string    `gorm:"column:location"`
	Serial        *string    `gorm:"column:serial"`
	IsActive      bool       `gorm:"column:is_active"`
	CredentialRef *string    `gorm:"column:credential_ref"`
	LastSeenAt    *time.Time `gorm:"column:last_seen_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (deviceModel) TableName() string {
	return "devices"
}

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, device attendancedomain.Device) (string, error) {
	model := deviceModel{
		ID:            device.ID,
		Name:          device.Name,
		Location:      device.Location,
		Serial:        device.Serial,
		IsActive:      device.IsActive,
		CredentialRef: device.CredentialRef,
		LastSeenAt:    device.LastSeenAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *DeviceRepository) FindByID(ctx context.Context, id string) (attendancedomain.Device, error) {
	var model deviceModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendancedomain.Device{}, domain.ErrNotFound
		}
		return attendancedomain.Device{}, err
	}
	return r.toDomain(model), nil
}

func (r *DeviceRepository) Update(ctx context.Context, device attendancedomain.Device) error {
	result := r.db.WithContext(ctx).Model(&deviceModel{}).Where("id = ?", device.ID).Updates(map[string]interface{}{
		"name":           device.Name,
		"location":       device.Location,
		"serial":         device.Serial,
		"is_active":      device.IsActive,
		"credential_ref": device.CredentialRef,
		"last_seen_at":   device.LastSeenAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeviceRepository) List(ctx context.Context) ([]attendancedomain.Device, error) {
	var models []deviceModel
	err := r.db.WithContext(ctx).Order("name").Find(&models).Error
	if err != nil {
		return nil, err
	}
	devices := make([]attendancedomain.Device, len(models))
	for i, model := range models {
		devices[i] = r.toDomain(model)
	}
	return devices, nil
}

func (r *DeviceRepository) toDomain(model deviceModel) attendancedomain.Device {
	return attendancedomain.Device{
		ID:            model.ID,
		Name:          model.Name,
		Location:      model.Location,
		Serial:        model.Serial,
		IsActive:      model.IsActive,
		CredentialRef: model.CredentialRef,
		LastSeenAt:    model.LastSeenAt,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ attendancedomain.DeviceRepository = (*DeviceRepository)(nil)
