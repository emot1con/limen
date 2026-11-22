package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	userdomain "limen/internal/domain/user"
)

type middlewareSessionRepoMock struct {
	session userdomain.Session
	touched bool
}

func (*middlewareSessionRepoMock) Create(context.Context, userdomain.Session) (string, error) {
	return "", nil
}
func (m *middlewareSessionRepoMock) FindByTokenHash(context.Context, string) (userdomain.Session, error) {
	return m.session, nil
}
func (m *middlewareSessionRepoMock) Touch(context.Context, string, time.Time, time.Time) error {
	m.touched = true
	return nil
}
func (*middlewareSessionRepoMock) Revoke(context.Context, string, time.Time) error { return nil }
func (*middlewareSessionRepoMock) RevokeAllByUserID(context.Context, string, time.Time) error {
	return nil
}

type middlewareRoleRepoMock struct{ roles []userdomain.UserRole }

func (*middlewareRoleRepoMock) Grant(context.Context, userdomain.UserRole) error { return nil }
func (*middlewareRoleRepoMock) Revoke(context.Context, string, string) error     { return nil }
func (*middlewareRoleRepoMock) Replace(context.Context, string, []userdomain.UserRole) error {
	return nil
}
func (m *middlewareRoleRepoMock) ListByUserID(context.Context, string) ([]userdomain.UserRole, error) {
	return m.roles, nil
}
func (*middlewareRoleRepoMock) ListByUserIDs(context.Context, []string) (map[string][]userdomain.UserRole, error) {
	return nil, nil
}

func TestAuthenticationAndCSRFMiddlewareWithMockRepositories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := "session-token"
	now := time.Now()
	sessions := &middlewareSessionRepoMock{session: userdomain.Session{ID: "session-1", UserID: "user-1", CreatedAt: now.Add(-time.Hour), LastSeenAt: &now, ExpiresAt: now.Add(time.Hour)}}
	roles := &middlewareRoleRepoMock{roles: []userdomain.UserRole{{RoleCode: "admin"}}}
	auth := NewAuthMiddleware(sessions, roles)
	router := gin.New()
	router.POST("/protected", auth.Authenticate(), auth.RequireCSRF(), auth.RequireRole("admin"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: token})
	request.AddCookie(&http.Cookie{Name: "csrf", Value: DeriveCSRF(token)})
	request.Header.Set("X-CSRF-Token", DeriveCSRF(token))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
}

func TestCSRFMiddlewareRejectsWrongToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/protected", NewAuthMiddleware(&middlewareSessionRepoMock{}, &middlewareRoleRepoMock{}).RequireCSRF(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	request.AddCookie(&http.Cookie{Name: "csrf", Value: "wrong"})
	request.Header.Set("X-CSRF-Token", "wrong")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, response.Code)
	}
}
