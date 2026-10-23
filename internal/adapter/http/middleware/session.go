package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"limen/internal/adapter/http/response"
	userdomain "limen/internal/domain/user"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	sessionRepo  userdomain.SessionRepository
	userRoleRepo userdomain.UserRoleRepository
}

func NewAuthMiddleware(sessionRepo userdomain.SessionRepository, userRoleRepo userdomain.UserRoleRepository) *AuthMiddleware {
	return &AuthMiddleware{sessionRepo: sessionRepo, userRoleRepo: userRoleRepo}
}

func (am *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := c.Cookie("session")
		if err != nil {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "session token required")
			c.Abort()
			return
		}
		fmt.Println("1")
		tokenHash := hashToken(sessionToken)
		session, err := am.sessionRepo.FindByTokenHash(c.Request.Context(), tokenHash)
		if err != nil {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid session")
			fmt.Println(err)
			c.Abort()
			return
		}
		fmt.Println("2")
		now := time.Now()
		if session.RevokedAt != nil || session.ExpiresAt.Before(now) {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "session expired")
			c.Abort()
			return
		}
		fmt.Println("3")
		idleExpiry := session.CreatedAt.Add(30 * time.Minute)
		absoluteExpiry := session.CreatedAt.Add(12 * time.Hour)
		if session.LastSeenAt != nil {
			idleExpiry = session.LastSeenAt.Add(30 * time.Minute)
		}
		if idleExpiry.Before(now) || absoluteExpiry.Before(now) {
			response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "session expired")
			c.Abort()
			return
		}
		expiresAt := now.Add(30 * time.Minute)
		if absoluteExpiry.Before(expiresAt) {
			expiresAt = absoluteExpiry
		}
		if err := am.sessionRepo.Touch(c.Request.Context(), session.ID, now, expiresAt); err != nil {
			response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			c.Abort()
			return
		}
		userRoles, err := am.userRoleRepo.ListByUserID(c.Request.Context(), session.UserID)
		if err != nil {
			response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			c.Abort()
			return
		}
		roles := make([]string, len(userRoles))
		for i, userRole := range userRoles {
			roles[i] = userRole.RoleCode
		}
		c.Set("user_id", session.UserID)
		c.Set("session_id", session.ID)
		c.Set("user_roles", roles)
		c.Next()
	}
}

func (am *AuthMiddleware) RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := c.Cookie("session")
		if err != nil {
			response.WriteError(c, http.StatusForbidden, "CSRF_INVALID", "csrf validation failed")
			c.Abort()
			return
		}
		csrfCookie, err := c.Cookie("csrf")
		if err != nil {
			response.WriteError(c, http.StatusForbidden, "CSRF_INVALID", "csrf validation failed")
			c.Abort()
			return
		}
		csrfHeader := c.GetHeader("X-CSRF-Token")
		if csrfHeader == "" {
			response.WriteError(c, http.StatusForbidden, "CSRF_INVALID", "csrf validation failed")
			c.Abort()
			return
		}
		expectedCSRF := deriveCSRF(sessionToken)
		if csrfCookie != expectedCSRF || csrfHeader != expectedCSRF {
			response.WriteError(c, http.StatusForbidden, "CSRF_INVALID", "csrf validation failed")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (am *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("user_roles")
		if !exists {
			response.WriteError(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
			c.Abort()
			return
		}
		roleList, ok := userRoles.([]string)
		if !ok {
			response.WriteError(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
			c.Abort()
			return
		}
		for _, required := range roles {
			for _, userRole := range roleList {
				if userRole == required {
					c.Next()
					return
				}
			}
		}
		response.WriteError(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		c.Abort()
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(h[:])
}

func DeriveCSRF(sessionToken string) string {
	h := sha256.Sum256([]byte("csrf:" + sessionToken))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func deriveCSRF(sessionToken string) string {
	return DeriveCSRF(sessionToken)
}

func GetUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", errors.New("user_id not found in context")
	}
	id, ok := userID.(string)
	if !ok {
		return "", errors.New("user_id invalid type")
	}
	return id, nil
}

func GetSessionID(c *gin.Context) (string, error) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return "", errors.New("session_id not found in context")
	}
	id, ok := sessionID.(string)
	if !ok {
		return "", errors.New("session_id invalid type")
	}
	return id, nil
}

func ParseIP(c *gin.Context) net.IP {
	if parsed := net.ParseIP(c.ClientIP()); parsed != nil {
		return parsed
	}
	return net.IPv4(0, 0, 0, 0)
}
