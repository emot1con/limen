package auth

import (
	"context"

	userdomain "limen/internal/domain/user"
)

type MeOutput struct {
	UserID   string
	Email    string
	FullName string
	Status   string
	Roles    []string
}

type MeUseCase struct {
	userRepo     userdomain.UserRepository
	userRoleRepo userdomain.UserRoleRepository
}

func NewMeUseCase(userRepo userdomain.UserRepository, userRoleRepo userdomain.UserRoleRepository) *MeUseCase {
	return &MeUseCase{userRepo: userRepo, userRoleRepo: userRoleRepo}
}

func (uc *MeUseCase) Execute(ctx context.Context, userID string) (MeOutput, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return MeOutput{}, err
	}
	userRoles, err := uc.userRoleRepo.ListByUserID(ctx, userID)
	if err != nil {
		return MeOutput{}, err
	}
	roles := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roles[i] = ur.RoleCode
	}
	return MeOutput{
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Status:   string(user.Status),
		Roles:    roles,
	}, nil
}
