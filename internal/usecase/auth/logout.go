package auth

import (
	"context"
	"time"

	userdomain "limen/internal/domain/user"
)

type LogoutInput struct {
	SessionID string
}

type LogoutUseCase struct {
	sessionRepo userdomain.SessionRepository
}

func NewLogoutUseCase(sessionRepo userdomain.SessionRepository) *LogoutUseCase {
	return &LogoutUseCase{sessionRepo: sessionRepo}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, input LogoutInput) error {
	return uc.sessionRepo.Revoke(ctx, input.SessionID, time.Now())
}
