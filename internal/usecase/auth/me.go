package auth

import (
	"context"

	"github.com/sirupsen/logrus"
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
	logger       *logrus.Logger
}

func NewMeUseCase(userRepo userdomain.UserRepository, userRoleRepo userdomain.UserRoleRepository, logger *logrus.Logger) *MeUseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &MeUseCase{userRepo: userRepo, userRoleRepo: userRoleRepo, logger: logger}
}

func (uc *MeUseCase) Execute(ctx context.Context, userID string) (MeOutput, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "me_find_user", "user_id": userID}).Error("usecase failed")
		return MeOutput{}, err
	}
	userRoles, err := uc.userRoleRepo.ListByUserID(ctx, userID)
	if err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "me_list_roles", "user_id": userID}).Error("usecase failed")
		return MeOutput{}, err
	}
	roles := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roles[i] = ur.RoleCode
	}
	return MeOutput{UserID: user.ID, Email: user.Email, FullName: user.FullName, Status: string(user.Status), Roles: roles}, nil
}
