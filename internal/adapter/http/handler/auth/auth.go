package auth

import (
	"limen/internal/adapter/http/handler/common"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	"limen/internal/domain"
	"limen/internal/usecase/auth"
)

type AuthHandler struct {
	authUC *auth.UseCase
}

func NewAuthHandler(authUC *auth.UseCase) *AuthHandler {
	return &AuthHandler{
		authUC: authUC,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	output, err := h.authUC.Login(c.Request.Context(), auth.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		IP:        middleware.ParseIP(c),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalid) {
			response.WriteError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
			return
		}
		response.TraceError(c, "auth.login", err)
		response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	csrfToken := deriveCSRF(output.SessionToken)
	c.SetSameSite(http.SameSiteStrictMode)
	const cookieMaxAge = 12 * 365 * 24 * 3600 // 12 tahun dalam detik
	c.SetCookie("session", output.SessionToken, cookieMaxAge, "/", "", true, true)
	c.SetCookie("csrf", csrfToken, cookieMaxAge, "/", "", true, false)
	response.WriteOK(c, gin.H{
		"user": gin.H{
			"id":       output.UserID,
			"email":    output.Email,
			"fullName": output.FullName,
			"roles":    output.Roles,
			"status":   output.Status,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, err := middleware.GetSessionID(c)
	if err != nil {
		response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "session required")
		return
	}
	if err := h.authUC.Logout(c.Request.Context(), auth.LogoutInput{SessionID: sessionID}); err != nil {
		response.TraceError(c, "auth.logout", err)
		response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session", "", -1, "/", "", true, true)
	c.SetCookie("csrf", "", -1, "/", "", true, false)
	response.WriteNoContent(c)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "user_id required")
		return
	}
	output, err := h.authUC.Me(c.Request.Context(), userID)
	if err != nil {
		response.TraceError(c, "auth.me", err)
		response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteOK(c, gin.H{
		"id":       output.UserID,
		"email":    output.Email,
		"fullName": output.FullName,
		"status":   output.Status,
		"roles":    output.Roles,
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	userID := c.Param("userId")
	if !common.ValidUUID(userID) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "userId", Message: "must be a UUID"})
		return
	}
	var req struct {
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	err := h.authUC.ResetPassword(c.Request.Context(), auth.ResetPasswordInput{
		UserID:      userID,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.WriteError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		response.TraceError(c, "auth.reset_password", err)
		response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	response.WriteNoContent(c)
}

func deriveCSRF(sessionToken string) string {
	return middleware.DeriveCSRF(sessionToken)
}
