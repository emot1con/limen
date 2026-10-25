package auth

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	userdomain "limen/internal/domain/user"
)

type LogoutInput struct {
	SessionID string
}

type LogoutUseCase struct {
	sessionRepo userdomain.SessionRepository
	logger      *logrus.Logger
}

func NewLogoutUseCase(sessionRepo userdomain.SessionRepository, logger *logrus.Logger) *LogoutUseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &LogoutUseCase{sessionRepo: sessionRepo, logger: logger}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, input LogoutInput) error {
	if err := uc.sessionRepo.Revoke(ctx, input.SessionID, time.Now()); err != nil {
		uc.logger.WithError(err).WithField("operation", "logout_revoke_session").Error("usecase failed")
		return err
	}
	return nil
}
