package user

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
	"limen/internal/usecase/auth"
)

var roleNames = map[string]string{
	"siswa":      "Siswa",
	"guru_mapel": "Guru Mapel",
	"guru_wali":  "Guru Wali",
	"guru_bk":    "Guru BK",
	"admin":      "Admin",
}

type UseCase struct {
	users    userdomain.UserRepository
	roles    userdomain.RoleRepository
	userRole userdomain.UserRoleRepository
	sessions userdomain.SessionRepository
	now      func() time.Time
	logger   *logrus.Logger
}

func NewUseCase(users userdomain.UserRepository, roles userdomain.RoleRepository, userRole userdomain.UserRoleRepository, sessions userdomain.SessionRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{users: users, roles: roles, userRole: userRole, sessions: sessions, now: time.Now, logger: logger}
}

type User struct {
	ID          string
	Email       string
	FullName    string
	Status      userdomain.Status
	Roles       []string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Profile     any
}

type CreateInput struct {
	Email     string
	Password  string
	FullName  string
	Roles     []string
	GrantedBy string
}

type UpdateInput struct {
	ID        string
	Email     *string
	FullName  *string
	Status    *userdomain.Status
	Roles     *[]string
	GrantedBy string
}

type Page struct {
	Items    []User
	Total    int64
	Page     int
	PageSize int
}

func (uc *UseCase) Create(ctx context.Context, input CreateInput) (User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	name := strings.TrimSpace(input.FullName)
	if email == "" || !strings.Contains(email, "@") || name == "" || !validPassword(input.Password) {
		return User{}, domain.ErrInvalid
	}
	roles, err := normalizeRoles(input.Roles)
	if err != nil {
		uc.logError(err, "user_normalize_roles", nil)
		return User{}, err
	}
	if err := uc.seedRoles(ctx); err != nil {
		uc.logError(err, "user_seed_roles", nil)
		return User{}, err
	}
	hash, err := auth.HashArgon2idPassword(input.Password)
	if err != nil {
		uc.logError(err, "user_hash_password", nil)
		return User{}, err
	}
	now := uc.now()
	newUser := userdomain.User{Email: email, PasswordHash: hash, FullName: name, Status: userdomain.StatusActive, CreatedAt: now, UpdatedAt: now}
	granted := make([]userdomain.UserRole, len(roles))
	for i, role := range roles {
		granted[i] = userdomain.UserRole{RoleCode: role, GrantedAt: now, GrantedBy: optional(input.GrantedBy)}
	}
	id, err := uc.users.CreateWithRoles(ctx, newUser, granted, hasRole(roles, "siswa"))
	if err != nil {
		uc.logError(err, "user_create_with_roles", nil)
		return User{}, err
	}
	return uc.Get(ctx, id)
}

func (uc *UseCase) Get(ctx context.Context, id string) (User, error) {
	item, err := uc.users.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "user_get", logrus.Fields{"user_id": id})
		return User{}, err
	}
	roles, err := uc.userRole.ListByUserID(ctx, id)
	if err != nil {
		uc.logError(err, "user_list_roles", logrus.Fields{"user_id": id})
		return User{}, err
	}
	return toUser(item, roleCodes(roles)), nil
}

func (uc *UseCase) List(ctx context.Context, q string, page, pageSize int) (Page, error) {
	page, pageSize = normalizePage(page, pageSize)
	items, total, err := uc.users.List(ctx, strings.TrimSpace(q), pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "user_list", nil)
		return Page{}, err
	}
	out := make([]User, len(items))
	for i, item := range items {
		roles, err := uc.userRole.ListByUserID(ctx, item.ID)
		if err != nil {
			uc.logError(err, "user_list_roles", logrus.Fields{"user_id": item.ID})
			return Page{}, err
		}
		out[i] = toUser(item, roleCodes(roles))
	}
	return Page{Items: out, Total: total, Page: page, PageSize: pageSize}, nil
}

func (uc *UseCase) Delete(ctx context.Context, id string) error {
	if err := uc.sessions.RevokeAllByUserID(ctx, id, uc.now()); err != nil {
		uc.logError(err, "user_revoke_sessions", logrus.Fields{"user_id": id})
		return err
	}
	if err := uc.users.Delete(ctx, id); err != nil {
		uc.logError(err, "user_delete", logrus.Fields{"user_id": id})
		return err
	}
	return nil
}

func (uc *UseCase) Update(ctx context.Context, input UpdateInput) (User, error) {
	item, err := uc.users.FindByID(ctx, input.ID)
	if err != nil {
		uc.logError(err, "user_get_for_update", logrus.Fields{"user_id": input.ID})
		return User{}, err
	}
	if input.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*input.Email))
		if email == "" || !strings.Contains(email, "@") {
			return User{}, domain.ErrInvalid
		}
		item.Email = email
	}
	if input.FullName != nil {
		name := strings.TrimSpace(*input.FullName)
		if name == "" {
			return User{}, domain.ErrInvalid
		}
		item.FullName = name
	}
	revoke := false
	if input.Status != nil {
		if *input.Status != userdomain.StatusActive && *input.Status != userdomain.StatusInactive {
			return User{}, domain.ErrInvalid
		}
		revoke = item.Status != userdomain.StatusInactive && *input.Status == userdomain.StatusInactive
		item.Status = *input.Status
	}
	item.UpdatedAt = uc.now()
	if err := uc.users.Update(ctx, item); err != nil {
		uc.logError(err, "user_update", logrus.Fields{"user_id": input.ID})
		return User{}, err
	}
	if input.Roles != nil {
		roles, err := normalizeRoles(*input.Roles)
		if err != nil {
			return User{}, err
		}
		if err := uc.seedRoles(ctx); err != nil {
			uc.logError(err, "user_seed_roles", nil)
			return User{}, err
		}
		granted := make([]userdomain.UserRole, len(roles))
		now := uc.now()
		for i, role := range roles {
			granted[i] = userdomain.UserRole{UserID: input.ID, RoleCode: role, GrantedAt: now, GrantedBy: optional(input.GrantedBy)}
		}
		if err := uc.userRole.Replace(ctx, input.ID, granted); err != nil {
			uc.logError(err, "user_replace_roles", logrus.Fields{"user_id": input.ID})
			return User{}, err
		}
	}
	if revoke {
		if err := uc.sessions.RevokeAllByUserID(ctx, input.ID, uc.now()); err != nil {
			uc.logError(err, "user_revoke_sessions", logrus.Fields{"user_id": input.ID})
			return User{}, err
		}
	}
	return uc.Get(ctx, input.ID)
}

func (uc *UseCase) seedRoles(ctx context.Context) error {
	for code, name := range roleNames {
		_, err := uc.roles.FindByCode(ctx, code)
		if err == nil {
			continue
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if _, err := uc.roles.Create(ctx, userdomain.Role{Code: code, Name: name}); err != nil {
			if _, findErr := uc.roles.FindByCode(ctx, code); findErr == nil {
				continue
			}
			return err
		}
	}
	return nil
}

func normalizeRoles(roles []string) ([]string, error) {
	if len(roles) == 0 {
		return nil, domain.ErrInvalid
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		code := strings.TrimSpace(role)
		if _, ok := roleNames[code]; !ok {
			return nil, domain.ErrInvalid
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}

func validPassword(password string) bool {
	if len(password) < 8 || len(password) > 128 {
		return false
	}
	var lower, upper, digit bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	return lower && upper && digit
}

func toUser(item userdomain.User, roles []string) User {
	return User{ID: item.ID, Email: item.Email, FullName: item.FullName, Status: item.Status, Roles: roles, LastLoginAt: item.LastLoginAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, Profile: nil}
}

func roleCodes(roles []userdomain.UserRole) []string {
	out := make([]string, len(roles))
	for i, role := range roles {
		out[i] = role.RoleCode
	}
	return out
}

func hasRole(roles []string, wanted string) bool {
	for _, role := range roles {
		if role == wanted {
			return true
		}
	}
	return false
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

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
