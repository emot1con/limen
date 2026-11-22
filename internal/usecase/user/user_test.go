package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
	"limen/internal/usecase/auth"
)

type userRepoMock struct {
	user                            userdomain.User
	created                         userdomain.User
	createdRoles                    []userdomain.UserRole
	student                         bool
	listItems                       []userdomain.User
	listTotal                       int64
	listQuery, listRole, listStatus string
	listLimit, listOffset           int
	updated                         userdomain.User
	updateCalls                     int
	deletedID                       string
	createErr, listErr, deleteErr   error
}

func (*userRepoMock) Create(context.Context, userdomain.User) (string, error) { return "", nil }
func (m *userRepoMock) CreateWithRoles(_ context.Context, item userdomain.User, roles []userdomain.UserRole, student bool) (string, error) {
	m.created, m.createdRoles, m.student = item, roles, student
	if m.createErr != nil {
		return "", m.createErr
	}
	m.user = item
	m.user.ID = "created-user"
	return m.user.ID, nil
}
func (m *userRepoMock) FindByID(context.Context, string) (userdomain.User, error) { return m.user, nil }
func (*userRepoMock) FindByEmail(context.Context, string) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (m *userRepoMock) List(_ context.Context, q, role, status string, limit, offset int) ([]userdomain.User, int64, error) {
	m.listQuery, m.listRole, m.listStatus, m.listLimit, m.listOffset = q, role, status, limit, offset
	return m.listItems, m.listTotal, m.listErr
}
func (m *userRepoMock) Update(_ context.Context, item userdomain.User) error {
	m.user, m.updated, m.updateCalls = item, item, m.updateCalls+1
	return nil
}
func (m *userRepoMock) Delete(_ context.Context, id string) error {
	m.deletedID = id
	return m.deleteErr
}

type userRoleRepoMock struct {
	roles      []userdomain.UserRole
	replaced   []userdomain.UserRole
	rolesByIDs map[string][]userdomain.UserRole
	listByIDs  []string
	listErr    error
}

func (*userRoleRepoMock) Grant(context.Context, userdomain.UserRole) error { return nil }
func (*userRoleRepoMock) Revoke(context.Context, string, string) error     { return nil }
func (m *userRoleRepoMock) Replace(_ context.Context, _ string, roles []userdomain.UserRole) error {
	m.replaced = roles
	return nil
}
func (m *userRoleRepoMock) ListByUserID(context.Context, string) ([]userdomain.UserRole, error) {
	return m.roles, nil
}
func (m *userRoleRepoMock) ListByUserIDs(_ context.Context, ids []string) (map[string][]userdomain.UserRole, error) {
	m.listByIDs = ids
	return m.rolesByIDs, m.listErr
}

type userSessionRepoMock struct {
	revokedID   string
	revokeCalls int
	revokeErr   error
}

func (*userSessionRepoMock) Create(context.Context, userdomain.Session) (string, error) {
	return "", nil
}
func (*userSessionRepoMock) FindByTokenHash(context.Context, string) (userdomain.Session, error) {
	return userdomain.Session{}, nil
}
func (*userSessionRepoMock) Touch(context.Context, string, time.Time, time.Time) error { return nil }
func (*userSessionRepoMock) Revoke(context.Context, string, time.Time) error           { return nil }
func (m *userSessionRepoMock) RevokeAllByUserID(_ context.Context, id string, _ time.Time) error {
	m.revokedID, m.revokeCalls = id, m.revokeCalls+1
	return m.revokeErr
}

func TestCreateNormalizesAndPersistsUserWithRoles(t *testing.T) {
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	users := &userRepoMock{}
	roles := &userRoleRepoMock{roles: []userdomain.UserRole{{RoleCode: "siswa"}, {RoleCode: "admin"}}}
	uc := NewUseCase(users, roles, &userSessionRepoMock{}, logrus.New())
	uc.now = func() time.Time { return now }
	out, err := uc.Create(context.Background(), CreateInput{Email: " USER@Example.COM ", Password: "Password123", FullName: " Test User ", Roles: []string{" siswa ", "admin", "admin"}, GrantedBy: "admin-1"})
	if err != nil {
		t.Fatal(err)
	}
	if users.created.Email != "user@example.com" || users.created.FullName != "Test User" || users.created.Status != userdomain.StatusActive || !auth.VerifyArgon2idPassword(users.created.PasswordHash, "Password123") {
		t.Fatalf("unexpected persisted user: %#v", users.created)
	}
	if !users.created.CreatedAt.Equal(now) || !users.student || len(users.createdRoles) != 2 || users.createdRoles[0].RoleCode != "siswa" || users.createdRoles[0].GrantedBy == nil || *users.createdRoles[0].GrantedBy != "admin-1" {
		t.Fatalf("unexpected roles: %#v", users.createdRoles)
	}
	if out.ID != "created-user" || len(out.Roles) != 2 {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestCreateRejectsInvalidInputs(t *testing.T) {
	cases := []CreateInput{{Email: "bad", Password: "Password123", FullName: "Test", Roles: []string{"admin"}}, {Email: "u@example.com", Password: "weak", FullName: "Test", Roles: []string{"admin"}}, {Email: "u@example.com", Password: "Password123", FullName: "", Roles: []string{"admin"}}, {Email: "u@example.com", Password: "Password123", FullName: "Test", Roles: []string{"invalid"}}}
	for _, input := range cases {
		users := &userRepoMock{}
		_, err := NewUseCase(users, &userRoleRepoMock{}, &userSessionRepoMock{}, logrus.New()).Create(context.Background(), input)
		if !errors.Is(err, domain.ErrInvalid) || users.created.ID != "" {
			t.Fatalf("expected invalid unpersisted input, err=%v user=%#v", err, users.created)
		}
	}
}

func TestListNormalizesPaginationAndMapsRoles(t *testing.T) {
	users := &userRepoMock{listItems: []userdomain.User{{ID: "u1", Status: userdomain.StatusActive}, {ID: "u2", Status: userdomain.StatusInactive}}, listTotal: 25}
	roles := &userRoleRepoMock{rolesByIDs: map[string][]userdomain.UserRole{"u1": {{RoleCode: "admin"}}, "u2": {{RoleCode: "guru_bk"}, {RoleCode: "guru_wali"}}}}
	page, err := NewUseCase(users, roles, &userSessionRepoMock{}, logrus.New()).List(context.Background(), " search ", " admin ", " active ", 3, 500)
	if err != nil {
		t.Fatal(err)
	}
	if users.listQuery != "search" || users.listRole != "admin" || users.listStatus != "active" || users.listLimit != 100 || users.listOffset != 200 {
		t.Fatalf("unexpected list params: %#v", users)
	}
	if len(roles.listByIDs) != 2 || page.Total != 25 || page.Page != 3 || page.PageSize != 100 || page.Items[0].Roles[0] != "admin" || len(page.Items[1].Roles) != 2 {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestListNormalizesInvalidPageValues(t *testing.T) {
	users := &userRepoMock{}
	page, err := NewUseCase(users, &userRoleRepoMock{}, &userSessionRepoMock{}, logrus.New()).List(context.Background(), "", "", "", 0, 0)
	if err != nil || users.listLimit != 20 || users.listOffset != 0 || page.Page != 1 || page.PageSize != 20 {
		t.Fatalf("unexpected page: %#v repo=%#v err=%v", page, users, err)
	}
}

func TestDeleteRevokesSessionsBeforeDeletingUser(t *testing.T) {
	users, sessions := &userRepoMock{}, &userSessionRepoMock{}
	if err := NewUseCase(users, &userRoleRepoMock{}, sessions, logrus.New()).Delete(context.Background(), "user-1"); err != nil {
		t.Fatal(err)
	}
	if sessions.revokedID != "user-1" || sessions.revokeCalls != 1 || users.deletedID != "user-1" {
		t.Fatalf("unexpected delete flow: sessions=%#v users=%#v", sessions, users)
	}
}

func TestDeleteStopsWhenSessionRevocationFails(t *testing.T) {
	users := &userRepoMock{}
	sessions := &userSessionRepoMock{revokeErr: errors.New("session store failed")}
	err := NewUseCase(users, &userRoleRepoMock{}, sessions, logrus.New()).Delete(context.Background(), "user-1")
	if err == nil || users.deletedID != "" {
		t.Fatalf("expected no deletion after revoke error: %v %q", err, users.deletedID)
	}
}

func TestUpdateInactiveUserRevokesSessionsAndReplacesRoles(t *testing.T) {
	users := &userRepoMock{user: userdomain.User{ID: "user-1", Status: userdomain.StatusActive}}
	roles := &userRoleRepoMock{roles: []userdomain.UserRole{{RoleCode: "admin"}}}
	sessions := &userSessionRepoMock{}
	inactive := userdomain.StatusInactive
	newRoles := []string{"admin", "guru_bk", "admin"}
	out, err := NewUseCase(users, roles, sessions, logrus.New()).Update(context.Background(), UpdateInput{ID: "user-1", Status: &inactive, Roles: &newRoles, GrantedBy: "admin-1"})
	if err != nil || out.Status != userdomain.StatusInactive || sessions.revokeCalls != 1 || len(roles.replaced) != 2 {
		t.Fatalf("unexpected update flow: out=%#v sessions=%#v roles=%#v err=%v", out, sessions, roles.replaced, err)
	}
}

func TestUpdateRejectsInvalidRolesBeforeWriting(t *testing.T) {
	users := &userRepoMock{user: userdomain.User{ID: "user-1"}}
	roles := []string{"unknown"}
	_, err := NewUseCase(users, &userRoleRepoMock{}, &userSessionRepoMock{}, logrus.New()).Update(context.Background(), UpdateInput{ID: "user-1", Roles: &roles})
	if !errors.Is(err, domain.ErrInvalid) || users.updateCalls != 0 {
		t.Fatalf("expected invalid roles without writes, err=%v writes=%d", err, users.updateCalls)
	}
}
