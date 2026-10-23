package auth

import (
	"context"
	"time"

	userdomain "limen/internal/domain/user"
)

type ResetPasswordInput struct {
	UserID      string
	NewPassword string
}

type ResetPasswordUseCase struct {
	userRepo    userdomain.UserRepository
	sessionRepo userdomain.SessionRepository
}

func NewResetPasswordUseCase(userRepo userdomain.UserRepository, sessionRepo userdomain.SessionRepository) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) error {
	user, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return err
	}
	hash, err := HashArgon2idPassword(input.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}
	return uc.sessionRepo.RevokeAllByUserID(ctx, user.ID, time.Now())
}
