package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"time"

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
}

func NewLoginUseCase(userRepo userdomain.UserRepository, userRoleRepo userdomain.UserRoleRepository, sessionRepo userdomain.SessionRepository) *LoginUseCase {
	return &LoginUseCase{userRepo: userRepo, userRoleRepo: userRoleRepo, sessionRepo: sessionRepo}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return LoginOutput{}, domain.ErrInvalid
		}
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
		return LoginOutput{}, err
	}
	loginAt := now
	user.LastLoginAt = &loginAt
	user.UpdatedAt = now
	_ = uc.userRepo.Update(ctx, user)
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
