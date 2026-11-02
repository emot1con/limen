package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	notificationuc "limen/internal/usecase/notification"
)

type NotificationHandler struct {
	notifications *notificationuc.UseCase
}

func NewNotificationHandler(notifications *notificationuc.UseCase) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

func (h *NotificationHandler) Create(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	var req struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	if req.Type == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "type", Message: "required"})
		return
	}
	out, err := h.notifications.Create(c.Request.Context(), notificationuc.CreateInput{
		UserID:  userID,
		Type:    req.Type,
		Payload: req.Payload,
	})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, notificationJSON(out))
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	page, pageSize, ok := pageQuery(c)
	if !ok {
		return
	}
	unreadOnly := c.Query("unreadOnly") == "true"
	out, err := h.notifications.List(c.Request.Context(), notificationuc.ListInput{
		UserID:     userID,
		UnreadOnly: unreadOnly,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	items := make([]gin.H, len(out.Items))
	for i, item := range out.Items {
		items[i] = notificationJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": out.Total, "page": out.Page, "pageSize": out.PageSize})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		response.WriteError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	notificationID := c.Param("notificationId")
	if !ValidUUID(notificationID) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "notificationId", Message: "must be valid UUID"})
		return
	}
	out, err := h.notifications.MarkRead(c.Request.Context(), notificationuc.MarkReadInput{
		NotificationID: notificationID,
		UserID:         userID,
	})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, notificationJSON(out))
}

func notificationJSON(item notificationuc.Notification) gin.H {
	return gin.H{"id": item.ID, "type": item.Type, "payload": item.Payload, "readAt": item.ReadAt, "createdAt": item.CreatedAt}
}
