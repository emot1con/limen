package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
)

type UseCase struct {
	userRepo     userdomain.UserRepository
	userRoleRepo userdomain.UserRoleRepository
	sessionRepo  userdomain.SessionRepository
	logger       *logrus.Logger
}

func NewUseCase(userRepo userdomain.UserRepository, userRoleRepo userdomain.UserRoleRepository, sessionRepo userdomain.SessionRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{
		userRepo:     userRepo,
		userRoleRepo: userRoleRepo,
		sessionRepo:  sessionRepo,
		logger:       logger,
	}
}

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

type LogoutInput struct {
	SessionID string
}

type MeOutput struct {
	UserID   string
	Email    string
	FullName string
	Status   string
	Roles    []string
}

type ResetPasswordInput struct {
	UserID      string
	NewPassword string
}

func (uc *UseCase) Login(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return LoginOutput{}, domain.ErrInvalid
		}
		uc.logError(err, "login_find_user", nil)
		return LoginOutput{}, err
	}
	if user.Status != userdomain.StatusActive {
		return LoginOutput{}, domain.ErrInvalid
	}
	if !VerifyArgon2idPassword(user.PasswordHash, input.Password) {
		return LoginOutput{}, domain.ErrInvalid
	}
	userRoles, err := uc.userRoleRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		uc.logError(err, "login_list_roles", logrus.Fields{"user_id": user.ID})
		return LoginOutput{}, err
	}
	roles := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roles[i] = ur.RoleCode
	}
	sessionToken := generateRandomToken(32)
	tokenHash := hashToken(sessionToken)
	now := time.Now()
	idleExpiry := now.AddDate(12, 0, 0)
	absoluteExpiry := now.AddDate(12, 0, 0)
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
		uc.logError(err, "login_create_session", logrus.Fields{"user_id": user.ID})
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

func (uc *UseCase) Logout(ctx context.Context, input LogoutInput) error {
	if err := uc.sessionRepo.Revoke(ctx, input.SessionID, time.Now()); err != nil {
		uc.logError(err, "logout_revoke_session", nil)
		return err
	}
	return nil
}

func (uc *UseCase) Me(ctx context.Context, userID string) (MeOutput, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		uc.logError(err, "me_find_user", logrus.Fields{"user_id": userID})
		return MeOutput{}, err
	}
	userRoles, err := uc.userRoleRepo.ListByUserID(ctx, userID)
	if err != nil {
		uc.logError(err, "me_list_roles", logrus.Fields{"user_id": userID})
		return MeOutput{}, err
	}
	roles := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roles[i] = ur.RoleCode
	}
	return MeOutput{UserID: user.ID, Email: user.Email, FullName: user.FullName, Status: string(user.Status), Roles: roles}, nil
}

func (uc *UseCase) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	user, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		uc.logError(err, "reset_password_find_user", logrus.Fields{"user_id": input.UserID})
		return err
	}
	hash, err := HashArgon2idPassword(input.NewPassword)
	if err != nil {
		uc.logError(err, "reset_password_hash", logrus.Fields{"user_id": user.ID})
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		uc.logError(err, "reset_password_update_user", logrus.Fields{"user_id": user.ID})
		return err
	}
	if err := uc.sessionRepo.RevokeAllByUserID(ctx, user.ID, time.Now()); err != nil {
		uc.logError(err, "reset_password_revoke_sessions", logrus.Fields{"user_id": user.ID})
		return err
	}
	return nil
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrAlreadyExists) {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
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

type argon2Params struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func HashArgon2idPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	time := uint32(1)
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)
	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, time, threads, b64Salt, b64Hash), nil
}

func VerifyArgon2idPassword(hash, password string) bool {
	parts, salt, storedHash, err := parseArgon2idHash(hash)
	if err != nil {
		return false
	}
	computedHash := argon2.IDKey([]byte(password), salt, parts.time, parts.memory, parts.threads, parts.keyLen)
	return len(computedHash) == len(storedHash) && subtleCompare(computedHash, storedHash)
}

func parseArgon2idHash(hash string) (argon2Params, []byte, []byte, error) {
	var p argon2Params
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return p, nil, nil, errors.New("invalid hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return p, nil, nil, err
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return p, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return p, nil, nil, err
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return p, nil, nil, err
	}
	p.keyLen = uint32(len(storedHash))
	return p, salt, storedHash, nil
}

func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
