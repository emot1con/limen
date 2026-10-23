package user

import (
	"context"
	"errors"
	"time"

	domain "limen/internal/domain"
	userdomain "limen/internal/domain/user"

	"gorm.io/gorm"
)

type userModel struct {
	ID           string            `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	Email        string            `gorm:"column:email"`
	PasswordHash string            `gorm:"column:password_hash"`
	FullName     string            `gorm:"column:full_name"`
	Status       userdomain.Status `gorm:"column:status"`
	LastLoginAt  *time.Time        `gorm:"column:last_login_at"`
	CreatedAt    time.Time         `gorm:"column:created_at"`
	UpdatedAt    time.Time         `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt    `gorm:"column:deleted_at;index"`
}

func (userModel) TableName() string {
	return "users"
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user userdomain.User) (string, error) {
	model := userModel{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FullName:     user.FullName,
		Status:       user.Status,
		LastLoginAt:  user.LastLoginAt,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.ID, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (userdomain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userdomain.User{}, domain.ErrNotFound
		}
		return userdomain.User{}, err
	}
	return r.toDomain(model), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (userdomain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userdomain.User{}, domain.ErrNotFound
		}
		return userdomain.User{}, err
	}
	return r.toDomain(model), nil
}

func (r *UserRepository) Update(ctx context.Context, user userdomain.User) error {
	result := r.db.WithContext(ctx).Model(&userModel{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"full_name":     user.FullName,
		"status":        user.Status,
		"last_login_at": user.LastLoginAt,
		"updated_at":    user.UpdatedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) toDomain(model userModel) userdomain.User {
	return userdomain.User{
		ID:           model.ID,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		FullName:     model.FullName,
		Status:       model.Status,
		LastLoginAt:  model.LastLoginAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

var _ userdomain.UserRepository = (*UserRepository)(nil)
