package attendance

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"

	"gorm.io/gorm"
)

type biometricIdentityModel struct {
	ID            string                       `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	Identifier    string                       `gorm:"column:identifier"`
	SubjectUserID string                       `gorm:"column:subject_user_id"`
	SubjectType   attendancedomain.SubjectType `gorm:"column:subject_type"`
	IsActive      bool                         `gorm:"column:is_active"`
	CreatedAt     time.Time                    `gorm:"column:created_at"`
	UpdatedAt     time.Time                    `gorm:"column:updated_at"`
}

func (biometricIdentityModel) TableName() string {
	return "biometric_identities"
}

type BiometricIdentityRepository struct {
	db *gorm.DB
}

func NewBiometricIdentityRepository(db *gorm.DB) *BiometricIdentityRepository {
	return &BiometricIdentityRepository{db: db}
}

func (r *BiometricIdentityRepository) Create(ctx context.Context, identity attendancedomain.BiometricIdentity) (string, error) {
	model := biometricIdentityModel{
		ID:            identity.ID,
		Identifier:    identity.Identifier,
		SubjectUserID: identity.SubjectUserID,
		SubjectType:   identity.SubjectType,
		IsActive:      identity.IsActive,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *BiometricIdentityRepository) FindByIdentifier(ctx context.Context, identifier string) (attendancedomain.BiometricIdentity, error) {
	var model biometricIdentityModel
	err := r.db.WithContext(ctx).Where("identifier = ?", identifier).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attendancedomain.BiometricIdentity{}, domain.ErrNotFound
		}
		return attendancedomain.BiometricIdentity{}, err
	}
	return r.toDomain(model), nil
}

func (r *BiometricIdentityRepository) ListBySubjectUserID(ctx context.Context, subjectUserID string) ([]attendancedomain.BiometricIdentity, error) {
	var models []biometricIdentityModel
	err := r.db.WithContext(ctx).Where("subject_user_id = ?", subjectUserID).Order("created_at, id").Find(&models).Error
	if err != nil {
		return nil, err
	}
	identities := make([]attendancedomain.BiometricIdentity, len(models))
	for i, model := range models {
		identities[i] = r.toDomain(model)
	}
	return identities, nil
}

func (r *BiometricIdentityRepository) Update(ctx context.Context, identity attendancedomain.BiometricIdentity) error {
	result := r.db.WithContext(ctx).Model(&biometricIdentityModel{}).Where("id = ?", identity.ID).Updates(map[string]interface{}{
		"identifier":      identity.Identifier,
		"subject_user_id": identity.SubjectUserID,
		"subject_type":    identity.SubjectType,
		"is_active":       identity.IsActive,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *BiometricIdentityRepository) toDomain(model biometricIdentityModel) attendancedomain.BiometricIdentity {
	return attendancedomain.BiometricIdentity{
		ID:            model.ID,
		Identifier:    model.Identifier,
		SubjectUserID: model.SubjectUserID,
		SubjectType:   model.SubjectType,
		IsActive:      model.IsActive,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

var _ attendancedomain.BiometricIdentityRepository = (*BiometricIdentityRepository)(nil)
