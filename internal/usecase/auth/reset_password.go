package auth

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	userdomain "limen/internal/domain/user"
)

type ResetPasswordInput struct {
	UserID      string
	NewPassword string
}

type ResetPasswordUseCase struct {
	userRepo    userdomain.UserRepository
	sessionRepo userdomain.SessionRepository
	logger      *logrus.Logger
}

func NewResetPasswordUseCase(userRepo userdomain.UserRepository, sessionRepo userdomain.SessionRepository, logger *logrus.Logger) *ResetPasswordUseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &ResetPasswordUseCase{userRepo: userRepo, sessionRepo: sessionRepo, logger: logger}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) error {
	user, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "reset_password_find_user", "user_id": input.UserID}).Error("usecase failed")
		return err
	}
	hash, err := HashArgon2idPassword(input.NewPassword)
	if err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "reset_password_hash", "user_id": user.ID}).Error("usecase failed")
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "reset_password_update_user", "user_id": user.ID}).Error("usecase failed")
		return err
	}
	if err := uc.sessionRepo.RevokeAllByUserID(ctx, user.ID, time.Now()); err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "reset_password_revoke_sessions", "user_id": user.ID}).Error("usecase failed")
		return err
	}
	return nil
}
