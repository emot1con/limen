package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
)

type userRepoMock struct {
	user        userdomain.User
	updated     userdomain.User
	updateCalls int
}

func (m *userRepoMock) Create(context.Context, userdomain.User) (string, error) { return "", nil }
func (m *userRepoMock) CreateWithRoles(context.Context, userdomain.User, []userdomain.UserRole, bool) (string, error) {
	return "", nil
}
func (m *userRepoMock) FindByID(context.Context, string) (userdomain.User, error) { return m.user, nil }
func (m *userRepoMock) FindByEmail(context.Context, string) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (m *userRepoMock) List(context.Context, string, string, string, int, int) ([]userdomain.User, int64, error) {
	return nil, 0, nil
}
func (m *userRepoMock) Update(_ context.Context, user userdomain.User) error {
	m.user = user
	m.updated = user
	m.updateCalls++
	return nil
}
func (m *userRepoMock) Delete(context.Context, string) error { return nil }

type userRoleRepoMock struct {
	roles    []userdomain.UserRole
	replaced []userdomain.UserRole
}

func (m *userRoleRepoMock) Grant(context.Context, userdomain.UserRole) error { return nil }
func (m *userRoleRepoMock) Revoke(context.Context, string, string) error     { return nil }
func (m *userRoleRepoMock) Replace(_ context.Context, _ string, roles []userdomain.UserRole) error {
	m.replaced = roles
	return nil
}
func (m *userRoleRepoMock) ListByUserID(context.Context, string) ([]userdomain.UserRole, error) {
	return m.roles, nil
}
func (m *userRoleRepoMock) ListByUserIDs(context.Context, []string) (map[string][]userdomain.UserRole, error) {
	return nil, nil
}

type userSessionRepoMock struct {
	revokedID   string
	revokeCalls int
}

func (m *userSessionRepoMock) Create(context.Context, userdomain.Session) (string, error) {
	return "", nil
}
func (m *userSessionRepoMock) FindByTokenHash(context.Context, string) (userdomain.Session, error) {
	return userdomain.Session{}, nil
}
func (m *userSessionRepoMock) Touch(context.Context, string, time.Time, time.Time) error { return nil }
func (m *userSessionRepoMock) Revoke(context.Context, string, time.Time) error           { return nil }
func (m *userSessionRepoMock) RevokeAllByUserID(_ context.Context, id string, _ time.Time) error {
	m.revokedID = id
	m.revokeCalls++
	return nil
}

func TestUpdateInactiveUserRevokesSessionsAndReplacesRoles(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	users := &userRepoMock{user: userdomain.User{ID: "user-1", Email: "old@example.com", FullName: "Old Name", Status: userdomain.StatusActive}}
	roles := &userRoleRepoMock{roles: []userdomain.UserRole{{RoleCode: "admin"}}}
	sessions := &userSessionRepoMock{}
	uc := NewUseCase(users, roles, sessions, logrus.New())
	uc.now = func() time.Time { return now }
	inactive := userdomain.StatusInactive
	newEmail := " NEW@EXAMPLE.COM "
	newName := " New Name "
	newRoles := []string{"admin", "guru_bk", "admin"}
	out, err := uc.Update(context.Background(), UpdateInput{ID: "user-1", Email: &newEmail, FullName: &newName, Status: &inactive, Roles: &newRoles, GrantedBy: "admin-1"})
	if err != nil {
		t.Fatal(err)
	}
	if users.updated.Email != "new@example.com" || users.updated.FullName != "New Name" || users.updated.Status != userdomain.StatusInactive || !users.updated.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected updated user: %#v", users.updated)
	}
	if sessions.revokeCalls != 1 || sessions.revokedID != "user-1" {
		t.Fatalf("expected revoked sessions, got %#v", sessions)
	}
	if len(roles.replaced) != 2 || roles.replaced[0].RoleCode != "admin" || roles.replaced[1].RoleCode != "guru_bk" {
		t.Fatalf("unexpected roles: %#v", roles.replaced)
	}
	if out.Email != "new@example.com" || out.Status != userdomain.StatusInactive {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestUpdateRejectsInvalidRolesBeforeWriting(t *testing.T) {
	users := &userRepoMock{user: userdomain.User{ID: "user-1", Status: userdomain.StatusActive}}
	uc := NewUseCase(users, &userRoleRepoMock{}, &userSessionRepoMock{}, logrus.New())
	roles := []string{"unknown"}
	_, err := uc.Update(context.Background(), UpdateInput{ID: "user-1", Roles: &roles})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if users.updateCalls != 0 {
		t.Fatalf("invalid roles must not update user, got %d writes", users.updateCalls)
	}
}
