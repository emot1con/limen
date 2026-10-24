package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"limen/internal/adapter/http/middleware"
	"limen/internal/adapter/http/response"
	"limen/internal/domain"
	userdomain "limen/internal/domain/user"
	deviceuc "limen/internal/usecase/device"
	settingsuc "limen/internal/usecase/settings"
	useruc "limen/internal/usecase/user"
)

type AdminHandler struct {
	settings *settingsuc.UseCase
	devices  *deviceuc.UseCase
	users    *useruc.UseCase
}

func NewAdminHandler(settings *settingsuc.UseCase, devices *deviceuc.UseCase, users *useruc.UseCase) *AdminHandler {
	return &AdminHandler{settings: settings, devices: devices, users: users}
}

func (h *AdminHandler) GetSettings(c *gin.Context) {
	out, err := h.settings.Get(c.Request.Context())
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, settingsJSON(out))
}

func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req struct {
		FirstEntryTime       string `json:"firstEntryTime"`
		LateToleranceMinutes *int   `json:"lateToleranceMinutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	var fields []response.FieldError
	if req.FirstEntryTime == "" {
		fields = append(fields, response.FieldError{Field: "firstEntryTime", Message: "required HH:mm"})
	}
	if req.LateToleranceMinutes != nil && *req.LateToleranceMinutes < 0 {
		fields = append(fields, response.FieldError{Field: "lateToleranceMinutes", Message: "must be zero or greater"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	actor, _ := middleware.GetUserID(c)
	out, err := h.settings.Update(c.Request.Context(), settingsuc.UpdateInput{FirstEntryTime: req.FirstEntryTime, LateToleranceMinutes: req.LateToleranceMinutes, UpdatedBy: actor})
	if err != nil {
		if errors.Is(err, domain.ErrInvalid) {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "firstEntryTime", Message: "must be HH:mm"})
			return
		}
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, settingsJSON(out))
}

func (h *AdminHandler) ListDevices(c *gin.Context) {
	page, pageSize, ok := pageQuery(c)
	if !ok {
		return
	}
	out, err := h.devices.List(c.Request.Context(), page, pageSize)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	items := make([]gin.H, len(out.Items))
	for i, item := range out.Items {
		items[i] = deviceJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": out.Total, "page": out.Page, "pageSize": out.PageSize})
}

func (h *AdminHandler) CreateDevice(c *gin.Context) {
	var req struct {
		Name     string  `json:"name"`
		Location *string `json:"location"`
		Serial   *string `json:"serial"`
		IsActive *bool   `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "name", Message: "required"})
		return
	}
	out, err := h.devices.Create(c.Request.Context(), deviceuc.CreateInput{Name: req.Name, Location: req.Location, Serial: req.Serial, IsActive: req.IsActive})
	if err != nil {
		writeMappedError(c, err, "serial")
		return
	}
	body := deviceJSON(out.Device)
	body["deviceKey"] = out.DeviceKey
	c.JSON(http.StatusCreated, body)
}

func (h *AdminHandler) GetDevice(c *gin.Context) {
	if !ValidUUID(c.Param("deviceId")) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "deviceId", Message: "must be a UUID"})
		return
	}
	out, err := h.devices.Get(c.Request.Context(), c.Param("deviceId"))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, deviceJSON(out))
}

func (h *AdminHandler) UpdateDevice(c *gin.Context) {
	if !ValidUUID(c.Param("deviceId")) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "deviceId", Message: "must be a UUID"})
		return
	}
	var req struct {
		Name     *string `json:"name"`
		Location *string `json:"location"`
		Serial   *string `json:"serial"`
		IsActive *bool   `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "name", Message: "required"})
		return
	}
	out, err := h.devices.Update(c.Request.Context(), deviceuc.UpdateInput{ID: c.Param("deviceId"), Name: req.Name, Location: req.Location, Serial: req.Serial, IsActive: req.IsActive})
	if err != nil {
		writeMappedError(c, err, "serial")
		return
	}
	c.JSON(http.StatusOK, deviceJSON(out))
}

func (h *AdminHandler) DeleteDevice(c *gin.Context) {
	if !ValidUUID(c.Param("deviceId")) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "deviceId", Message: "must be a UUID"})
		return
	}
	if err := h.devices.Delete(c.Request.Context(), c.Param("deviceId")); err != nil {
		writeMappedError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, pageSize, ok := pageQuery(c)
	if !ok {
		return
	}
	out, err := h.users.List(c.Request.Context(), c.Query("q"), page, pageSize)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	items := make([]gin.H, len(out.Items))
	for i, item := range out.Items {
		items[i] = userJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": out.Total, "page": out.Page, "pageSize": out.PageSize})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Email    string   `json:"email"`
		Password string   `json:"password"`
		FullName string   `json:"fullName"`
		Roles    []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	if fields := userFields(req.Email, req.Password, req.FullName, req.Roles, true); len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	actor, _ := middleware.GetUserID(c)
	out, err := h.users.Create(c.Request.Context(), useruc.CreateInput{Email: req.Email, Password: req.Password, FullName: req.FullName, Roles: req.Roles, GrantedBy: actor})
	if err != nil {
		writeMappedError(c, err, "email")
		return
	}
	c.JSON(http.StatusCreated, userJSON(out))
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	if !ValidUUID(c.Param("userId")) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "userId", Message: "must be a UUID"})
		return
	}
	out, err := h.users.Get(c.Request.Context(), c.Param("userId"))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, userJSON(out))
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	if !ValidUUID(c.Param("userId")) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "userId", Message: "must be a UUID"})
		return
	}
	if err := h.users.Delete(c.Request.Context(), c.Param("userId")); err != nil {
		writeMappedError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	if !ValidUUID(c.Param("userId")) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "userId", Message: "must be a UUID"})
		return
	}
	var req struct {
		Email    *string   `json:"email"`
		FullName *string   `json:"fullName"`
		Status   *string   `json:"status"`
		Roles    *[]string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "invalid request body"})
		return
	}
	var fields []response.FieldError
	if req.Email != nil && (strings.TrimSpace(*req.Email) == "" || !strings.Contains(*req.Email, "@")) {
		fields = append(fields, response.FieldError{Field: "email", Message: "must be valid email"})
	}
	if req.FullName != nil && strings.TrimSpace(*req.FullName) == "" {
		fields = append(fields, response.FieldError{Field: "fullName", Message: "required"})
	}
	var status *userdomain.Status
	if req.Status != nil {
		value := userdomain.Status(*req.Status)
		if value != userdomain.StatusActive && value != userdomain.StatusInactive {
			fields = append(fields, response.FieldError{Field: "status", Message: "must be active or inactive"})
		} else {
			status = &value
		}
	}
	if req.Roles != nil {
		fields = append(fields, roleFields(*req.Roles)...)
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	actor, _ := middleware.GetUserID(c)
	out, err := h.users.Update(c.Request.Context(), useruc.UpdateInput{ID: c.Param("userId"), Email: req.Email, FullName: req.FullName, Status: status, Roles: req.Roles, GrantedBy: actor})
	if err != nil {
		writeMappedError(c, err, "email")
		return
	}
	c.JSON(http.StatusOK, userJSON(out))
}

func settingsJSON(out settingsuc.Output) gin.H {
	return gin.H{"id": out.ID, "firstEntryTime": out.FirstEntryTime, "timezone": out.Timezone, "lateToleranceMinutes": out.LateToleranceMinutes, "updatedBy": out.UpdatedBy, "updatedAt": out.UpdatedAt}
}

func deviceJSON(item deviceuc.Device) gin.H {
	return gin.H{"id": item.ID, "name": item.Name, "location": item.Location, "serial": item.Serial, "isActive": item.IsActive, "lastSeenAt": item.LastSeenAt, "createdAt": item.CreatedAt, "updatedAt": item.UpdatedAt}
}

func userJSON(item useruc.User) gin.H {
	return gin.H{"id": item.ID, "email": item.Email, "fullName": item.FullName, "status": item.Status, "roles": item.Roles, "profile": item.Profile, "lastLoginAt": item.LastLoginAt, "createdAt": item.CreatedAt, "updatedAt": item.UpdatedAt}
}

func pageQuery(c *gin.Context) (int, int, bool) {
	page := 1
	pageSize := 20
	if raw := c.Query("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "page", Message: "must be 1 or greater"})
			return 0, 0, false
		}
		page = value
	}
	if raw := c.Query("pageSize"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "pageSize", Message: "must be between 1 and 100"})
			return 0, 0, false
		}
		pageSize = value
	}
	return page, pageSize, true
}

func userFields(email, password, fullName string, roles []string, creating bool) []response.FieldError {
	var fields []response.FieldError
	if creating {
		if strings.TrimSpace(email) == "" || !strings.Contains(email, "@") {
			fields = append(fields, response.FieldError{Field: "email", Message: "must be valid email"})
		}
		if !validPassword(password) {
			fields = append(fields, response.FieldError{Field: "password", Message: "minimum 8 characters with uppercase, lowercase, and digit"})
		}
		if strings.TrimSpace(fullName) == "" {
			fields = append(fields, response.FieldError{Field: "fullName", Message: "required"})
		}
		fields = append(fields, roleFields(roles)...)
	}
	return fields
}

func roleFields(roles []string) []response.FieldError {
	if len(roles) == 0 {
		return []response.FieldError{{Field: "roles", Message: "at least one known role required"}}
	}
	known := map[string]struct{}{"siswa": {}, "guru_mapel": {}, "guru_wali": {}, "guru_bk": {}, "admin": {}}
	for _, role := range roles {
		if _, ok := known[strings.TrimSpace(role)]; !ok {
			return []response.FieldError{{Field: "roles", Message: "contains unknown role"}}
		}
	}
	return nil
}

func validPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var lower, upper, digit bool
	for _, r := range password {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		}
	}
	return lower && upper && digit
}

func writeMappedError(c *gin.Context, err error, duplicateField ...string) {
	if errors.Is(err, domain.ErrNotFound) {
		response.WriteError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, domain.ErrAlreadyExists) || isUniqueViolation(err) {
		field := "request"
		if len(duplicateField) > 0 {
			field = duplicateField[0]
		}
		response.WriteError(c, http.StatusConflict, "CONFLICT", "resource already exists", response.FieldError{Field: field, Message: "already exists"})
		return
	}
	if errors.Is(err, domain.ErrInvalid) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed")
		return
	}
	response.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func isUniqueViolation(err error) bool {
	message := err.Error()
	return strings.Contains(message, "23505") || strings.Contains(message, "duplicate key")
}

func ValidUUID(value string) bool {
	matched, _ := regexp.MatchString(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, value)
	return matched
}
