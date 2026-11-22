package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	authhandler "limen/internal/adapter/http/handler/auth"
	userdomain "limen/internal/domain/user"
	"limen/internal/usecase/auth"
)

type httpUserRepo struct{ user userdomain.User }

func (m *httpUserRepo) Create(context.Context, userdomain.User) (string, error) { return "", nil }
func (m *httpUserRepo) CreateWithRoles(context.Context, userdomain.User, []userdomain.UserRole, bool) (string, error) {
	return "", nil
}
func (m *httpUserRepo) FindByID(context.Context, string) (userdomain.User, error) { return m.user, nil }
func (m *httpUserRepo) FindByEmail(context.Context, string) (userdomain.User, error) {
	return m.user, nil
}
func (m *httpUserRepo) List(context.Context, string, string, string, int, int) ([]userdomain.User, int64, error) {
	return nil, 0, nil
}
func (m *httpUserRepo) Update(context.Context, userdomain.User) error { return nil }
func (m *httpUserRepo) Delete(context.Context, string) error          { return nil }

type httpRoleRepo struct{}

func (*httpRoleRepo) Grant(context.Context, userdomain.UserRole) error             { return nil }
func (*httpRoleRepo) Revoke(context.Context, string, string) error                 { return nil }
func (*httpRoleRepo) Replace(context.Context, string, []userdomain.UserRole) error { return nil }
func (*httpRoleRepo) ListByUserID(context.Context, string) ([]userdomain.UserRole, error) {
	return []userdomain.UserRole{{UserID: "user-1", RoleCode: "student"}}, nil
}
func (*httpRoleRepo) ListByUserIDs(context.Context, []string) (map[string][]userdomain.UserRole, error) {
	return nil, nil
}

type httpSessionRepo struct{ session userdomain.Session }

func (m *httpSessionRepo) Create(_ context.Context, session userdomain.Session) (string, error) {
	m.session = session
	return "session-1", nil
}
func (*httpSessionRepo) FindByTokenHash(context.Context, string) (userdomain.Session, error) {
	return userdomain.Session{}, nil
}
func (*httpSessionRepo) Touch(context.Context, string, time.Time, time.Time) error  { return nil }
func (*httpSessionRepo) Revoke(context.Context, string, time.Time) error            { return nil }
func (*httpSessionRepo) RevokeAllByUserID(context.Context, string, time.Time) error { return nil }

func TestLoginEndpointWithMockRepositories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hash, err := auth.HashArgon2idPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	userRepo := &httpUserRepo{user: userdomain.User{ID: "user-1", Email: "user@example.com", FullName: "Test User", Status: userdomain.StatusActive, PasswordHash: hash}}
	sessionRepo := &httpSessionRepo{}
	logger := logrus.New()
	authUC := auth.NewUseCase(userRepo, &httpRoleRepo{}, sessionRepo, logger)
	handler := authhandler.NewAuthHandler(authUC)
	router := gin.New()
	router.POST("/api/v1/auth/login", handler.Login)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var body struct {
		Data struct {
			User struct {
				ID    string   `json:"id"`
				Email string   `json:"email"`
				Roles []string `json:"roles"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.User.ID != "user-1" || body.Data.User.Email != "user@example.com" || len(body.Data.User.Roles) != 1 || body.Data.User.Roles[0] != "student" {
		t.Fatalf("unexpected response body: %#v", body)
	}
	if sessionRepo.session.UserID != "user-1" {
		t.Fatalf("expected mocked session repository call, got %#v", sessionRepo.session)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 2 || cookies[0].Name != "session" || cookies[1].Name != "csrf" {
		t.Fatalf("expected session and csrf cookies, got %#v", cookies)
	}
}

func TestLoginEndpointRejectsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authUC := auth.NewUseCase(&httpUserRepo{}, &httpRoleRepo{}, &httpSessionRepo{}, logrus.New())
	router := gin.New()
	router.POST("/api/v1/auth/login", authhandler.NewAuthHandler(authUC).Login)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"bad","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, response.Code)
	}
}
