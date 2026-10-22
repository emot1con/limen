package user

import (
	"context"
	"time"

	domain "limen/internal/domain"
	userdomain "limen/internal/domain/user"

	"gorm.io/gorm"
)

type userRoleModel struct {
	UserID    string    `gorm:"column:user_id;primaryKey"`
	RoleCode  string    `gorm:"column:role_code;primaryKey"`
	GrantedAt time.Time `gorm:"column:granted_at"`
	GrantedBy *string   `gorm:"column:granted_by"`
}

func (userRoleModel) TableName() string {
	return "user_roles"
}

type UserRoleRepository struct {
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) *UserRoleRepository {
	return &UserRoleRepository{db: db}
}

func (r *UserRoleRepository) Grant(ctx context.Context, userRole userdomain.UserRole) error {
	model := userRoleModel{
		UserID:    userRole.UserID,
		RoleCode:  userRole.RoleCode,
		GrantedAt: userRole.GrantedAt,
		GrantedBy: userRole.GrantedBy,
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *UserRoleRepository) Revoke(ctx context.Context, userID, roleCode string) error {
	result := r.db.WithContext(ctx).Where("user_id = ? AND role_code = ?", userID, roleCode).Delete(&userRoleModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRoleRepository) ListByUserID(ctx context.Context, userID string) ([]userdomain.UserRole, error) {
	var models []userRoleModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("role_code").Find(&models).Error; err != nil {
		return nil, err
	}
	userRoles := make([]userdomain.UserRole, len(models))
	for i, model := range models {
		userRoles[i] = r.toDomain(model)
	}
	return userRoles, nil
}

func (r *UserRoleRepository) toDomain(model userRoleModel) userdomain.UserRole {
	return userdomain.UserRole{
		UserID:    model.UserID,
		RoleCode:  model.RoleCode,
		GrantedAt: model.GrantedAt,
		GrantedBy: model.GrantedBy,
	}
}

var _ userdomain.UserRoleRepository = (*UserRoleRepository)(nil)
