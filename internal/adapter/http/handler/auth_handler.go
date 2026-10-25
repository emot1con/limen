package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	"limen/internal/domain"
	"limen/internal/usecase/auth"
)

type AuthHandler struct {
	loginUC         *auth.LoginUseCase
	logoutUC        *auth.LogoutUseCase
	meUC            *auth.MeUseCase
	resetPasswordUC *auth.ResetPasswordUseCase
}

func NewAuthHandler(loginUC *auth.LoginUseCase, logoutUC *auth.LogoutUseCase, meUC *auth.MeUseCase, resetPasswordUC *auth.ResetPasswordUseCase) *AuthHandler {
	return &AuthHandler{
		loginUC:         loginUC,
		logoutUC:        logoutUC,
		meUC:            meUC,
		resetPasswordUC: resetPasswordUC,
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
	output, err := h.loginUC.Execute(c.Request.Context(), auth.LoginInput{
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
	c.SetCookie("session", output.SessionToken, 12*3600, "/", "", true, true)
	c.SetCookie("csrf", csrfToken, 12*3600, "/", "", true, false)
	c.JSON(http.StatusOK, gin.H{
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
	if err := h.logoutUC.Execute(c.Request.Context(), auth.LogoutInput{SessionID: sessionID}); err != nil {
		response.TraceError(c, "auth.logout", err)
		response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session", "", -1, "/", "", true, true)
	c.SetCookie("csrf", "", -1, "/", "", true, false)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "user_id required")
		return
	}
	output, err := h.meUC.Execute(c.Request.Context(), userID)
	if err != nil {
		response.TraceError(c, "auth.me", err)
		response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       output.UserID,
		"email":    output.Email,
		"fullName": output.FullName,
		"status":   output.Status,
		"roles":    output.Roles,
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	userID := c.Param("userId")
	if !ValidUUID(userID) {
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
	err := h.resetPasswordUC.Execute(c.Request.Context(), auth.ResetPasswordInput{
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
	c.Status(http.StatusNoContent)
}

func deriveCSRF(sessionToken string) string {
	return middleware.DeriveCSRF(sessionToken)
}
