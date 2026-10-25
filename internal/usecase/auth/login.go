package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
)

type LoginInput struct {
	Email     string
	Password  string
	IP        net.IP
	UserAgent string
}

type LoginOutput struct {
	UserID       string
	Email        string
	FullName     string
	Status       string
	Roles        []string
	SessionToken string
}

type LoginUseCase struct {
	userRepo     userdomain.UserRepository
	userRoleRepo userdomain.UserRoleRepository
	sessionRepo  userdomain.SessionRepository
	logger       *logrus.Logger
}

func NewLoginUseCase(userRepo userdomain.UserRepository, userRoleRepo userdomain.UserRoleRepository, sessionRepo userdomain.SessionRepository, logger *logrus.Logger) *LoginUseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &LoginUseCase{userRepo: userRepo, userRoleRepo: userRoleRepo, sessionRepo: sessionRepo, logger: logger}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return LoginOutput{}, domain.ErrInvalid
		}
		uc.logger.WithError(err).WithField("operation", "login_find_user").Error("usecase failed")
		return LoginOutput{}, err
	}
	if user.Status != userdomain.StatusActive {
		return LoginOutput{}, domain.ErrInvalid
	}
	if !verifyArgon2idPassword(user.PasswordHash, input.Password) {
		return LoginOutput{}, domain.ErrInvalid
	}
	userRoles, err := uc.userRoleRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "login_list_roles", "user_id": user.ID}).Error("usecase failed")
		return LoginOutput{}, err
	}
	roles := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roles[i] = ur.RoleCode
	}
	sessionToken := generateRandomToken(32)
	tokenHash := hashToken(sessionToken)
	now := time.Now()
	idleExpiry := now.Add(30 * time.Minute)
	absoluteExpiry := now.Add(12 * time.Hour)
	expiresAt := idleExpiry
	if absoluteExpiry.Before(expiresAt) {
		expiresAt = absoluteExpiry
	}
	session := userdomain.Session{
		UserID:     user.ID,
		TokenHash:  tokenHash,
		CreatedAt:  now,
		LastSeenAt: &now,
		ExpiresAt:  expiresAt,
		IP:         input.IP,
		UserAgent:  input.UserAgent,
	}
	if _, err := uc.sessionRepo.Create(ctx, session); err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{"operation": "login_create_session", "user_id": user.ID}).Error("usecase failed")
		return LoginOutput{}, err
	}
	loginAt := now
	user.LastLoginAt = &loginAt
	user.UpdatedAt = now
	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logger.WithError(err).WithField("user_id", user.ID).Warn("update last login failed")
	}
	return LoginOutput{
		UserID:       user.ID,
		Email:        user.Email,
		FullName:     user.FullName,
		Status:       string(user.Status),
		Roles:        roles,
		SessionToken: sessionToken,
	}, nil
}

func generateRandomToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(h[:])
}

func verifyArgon2idPassword(hash, password string) bool {
	return VerifyArgon2idPassword(hash, password)
}
