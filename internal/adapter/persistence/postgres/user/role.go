package user

import (
	"context"
	"errors"

	domain "limen/internal/domain"
	userdomain "limen/internal/domain/user"

	"gorm.io/gorm"
)

type roleModel struct {
	Code string `gorm:"column:code;primaryKey"`
	Name string `gorm:"column:name"`
}

func (roleModel) TableName() string {
	return "roles"
}

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role userdomain.Role) (string, error) {
	model := roleModel{Code: role.Code, Name: role.Name}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return "", err
	}
	return model.Code, nil
}

func (r *RoleRepository) FindByCode(ctx context.Context, code string) (userdomain.Role, error) {
	var model roleModel
	if err := r.db.WithContext(ctx).First(&model, "code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userdomain.Role{}, domain.ErrNotFound
		}
		return userdomain.Role{}, err
	}
	return r.toDomain(model), nil
}

func (r *RoleRepository) List(ctx context.Context) ([]userdomain.Role, error) {
	var models []roleModel
	if err := r.db.WithContext(ctx).Order("code").Find(&models).Error; err != nil {
		return nil, err
	}
	roles := make([]userdomain.Role, len(models))
	for i, model := range models {
		roles[i] = r.toDomain(model)
	}
	return roles, nil
}

func (r *RoleRepository) toDomain(model roleModel) userdomain.Role {
	return userdomain.Role{Code: model.Code, Name: model.Name}
}

var _ userdomain.RoleRepository = (*RoleRepository)(nil)
