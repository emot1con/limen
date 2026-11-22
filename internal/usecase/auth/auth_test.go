package auth

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
)

type loginUserRepoMock struct {
	user    userdomain.User
	err     error
	updated userdomain.User
}

func (m *loginUserRepoMock) Create(context.Context, userdomain.User) (string, error) { return "", nil }
func (m *loginUserRepoMock) CreateWithRoles(context.Context, userdomain.User, []userdomain.UserRole, bool) (string, error) {
	return "", nil
}
func (m *loginUserRepoMock) FindByID(context.Context, string) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (m *loginUserRepoMock) FindByEmail(context.Context, string) (userdomain.User, error) {
	return m.user, m.err
}
func (m *loginUserRepoMock) List(context.Context, string, string, string, int, int) ([]userdomain.User, int64, error) {
	return nil, 0, nil
}
func (m *loginUserRepoMock) Update(_ context.Context, user userdomain.User) error {
	m.updated = user
	return nil
}
func (m *loginUserRepoMock) Delete(context.Context, string) error { return nil }

type loginRoleRepoMock struct {
	roles []userdomain.UserRole
	err   error
}

func (m *loginRoleRepoMock) Grant(context.Context, userdomain.UserRole) error             { return nil }
func (m *loginRoleRepoMock) Revoke(context.Context, string, string) error                 { return nil }
func (m *loginRoleRepoMock) Replace(context.Context, string, []userdomain.UserRole) error { return nil }
func (m *loginRoleRepoMock) ListByUserID(context.Context, string) ([]userdomain.UserRole, error) {
	return m.roles, m.err
}
func (m *loginRoleRepoMock) ListByUserIDs(context.Context, []string) (map[string][]userdomain.UserRole, error) {
	return nil, nil
}

type loginSessionRepoMock struct {
	session userdomain.Session
	err     error
}

func (m *loginSessionRepoMock) Create(_ context.Context, session userdomain.Session) (string, error) {
	m.session = session
	return "session-id", m.err
}
func (m *loginSessionRepoMock) FindByTokenHash(context.Context, string) (userdomain.Session, error) {
	return userdomain.Session{}, nil
}
func (m *loginSessionRepoMock) Touch(context.Context, string, time.Time, time.Time) error { return nil }
func (m *loginSessionRepoMock) Revoke(context.Context, string, time.Time) error           { return nil }
func (m *loginSessionRepoMock) RevokeAllByUserID(context.Context, string, time.Time) error {
	return nil
}

func TestLoginExecuteSuccess(t *testing.T) {
	hash, err := HashArgon2idPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	users := &loginUserRepoMock{user: userdomain.User{ID: "user-1", Email: "user@example.com", FullName: "Test User", Status: userdomain.StatusActive, PasswordHash: hash}}
	roles := &loginRoleRepoMock{roles: []userdomain.UserRole{{RoleCode: "admin"}}}
	sessions := &loginSessionRepoMock{}
	uc := NewUseCase(users, roles, sessions, logrus.New())
	out, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "secret", IP: net.ParseIP("127.0.0.1"), UserAgent: "test-agent"})
	if err != nil {
		t.Fatal(err)
	}
	if out.UserID != "user-1" || out.Email != "user@example.com" || out.FullName != "Test User" {
		t.Fatalf("unexpected output: %#v", out)
	}
	if len(out.SessionToken) == 0 {
		t.Fatal("expected generated session token")
	}
	if len(out.Roles) != 1 || out.Roles[0] != "admin" {
		t.Fatalf("unexpected roles: %#v", out.Roles)
	}
	if sessions.session.UserID != "user-1" || sessions.session.TokenHash == out.SessionToken || sessions.session.UserAgent != "test-agent" {
		t.Fatalf("unexpected persisted session: %#v", sessions.session)
	}
	if users.updated.LastLoginAt == nil {
		t.Fatal("expected last login timestamp update")
	}
}

func TestLoginExecuteRejectsInvalidCredentials(t *testing.T) {
	hash, err := HashArgon2idPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		user     userdomain.User
		findErr  error
		password string
	}{
		{name: "missing user", findErr: domain.ErrNotFound, password: "secret"},
		{name: "wrong password", user: userdomain.User{ID: "user-1", Status: userdomain.StatusActive, PasswordHash: hash}, password: "wrong"},
		{name: "inactive user", user: userdomain.User{ID: "user-1", Status: userdomain.StatusInactive, PasswordHash: hash}, password: "secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewUseCase(&loginUserRepoMock{user: tc.user, err: tc.findErr}, &loginRoleRepoMock{}, &loginSessionRepoMock{}, logrus.New())
			_, err := uc.Login(context.Background(), LoginInput{Email: "user@example.com", Password: tc.password})
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
		})
	}
}

func TestLoginExecuteReturnsSessionError(t *testing.T) {
	hash, err := HashArgon2idPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("session storage unavailable")
	uc := NewUseCase(&loginUserRepoMock{user: userdomain.User{ID: "user-1", Status: userdomain.StatusActive, PasswordHash: hash}}, &loginRoleRepoMock{}, &loginSessionRepoMock{err: wantErr}, logrus.New())
	if _, err := uc.Login(context.Background(), LoginInput{Password: "secret"}); !errors.Is(err, wantErr) {
		t.Fatalf("expected session error, got %v", err)
	}
}
