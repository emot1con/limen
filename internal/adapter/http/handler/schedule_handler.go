package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"limen/internal/adapter/http/response"
	academicdomain "limen/internal/domain/academic"
	scheduleuc "limen/internal/usecase/schedule"

	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct{ schedule *scheduleuc.UseCase }

func NewScheduleHandler(schedule *scheduleuc.UseCase) *ScheduleHandler {
	return &ScheduleHandler{schedule: schedule}
}

func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	filter := academicdomain.ScheduleFilter{}
	for _, key := range []string{"classId", "subjectId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	filter.ClassID, filter.SubjectID = c.Query("classId"), c.Query("subjectId")
	if kind := c.Query("kind"); kind != "" {
		filter.Kind = academicdomain.ScheduleKind(kind)
		if !validScheduleKind(filter.Kind) {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "kind", Message: "invalid"})
			return
		}
	}
	if value := c.Query("dayOfWeek"); value != "" {
		day, err := strconv.Atoi(value)
		if err != nil || day < 1 || day > 7 {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "dayOfWeek", Message: "must be 1 to 7"})
			return
		}
		filter.DayOfWeeks = []int16{int16(day)}
	}
	days, ok := scheduleDateDays(c)
	if !ok {
		return
	}
	if len(days) > 0 {
		if len(filter.DayOfWeeks) > 0 && !containsDay(days, filter.DayOfWeeks[0]) {
			c.JSON(http.StatusOK, gin.H{"items": []gin.H{}, "total": 0, "page": page, "pageSize": size})
			return
		}
		if len(filter.DayOfWeeks) == 0 {
			filter.DayOfWeeks = days
		}
	}
	items, total, err := h.schedule.ListSchedules(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = scheduleJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	var req struct {
		ClassID       string  `json:"classId"`
		SubjectID     string  `json:"subjectId"`
		StartTime     string  `json:"startTime"`
		EndTime       string  `json:"endTime"`
		Kind          string  `json:"kind"`
		TeacherUserID *string `json:"teacherUserId"`
		DayOfWeek     int16   `json:"dayOfWeek"`
	}
	if !bindAcademic(c, &req) {
		return
	}
	var fields []response.FieldError
	for key, value := range map[string]string{"classId": req.ClassID, "subjectId": req.SubjectID} {
		if !ValidUUID(value) {
			fields = append(fields, response.FieldError{Field: key, Message: "must be a UUID"})

		}
	}
	if req.TeacherUserID != nil && !ValidUUID(*req.TeacherUserID) {
		fields = append(fields, response.FieldError{Field: "teacherUserId", Message: "must be a UUID"})
	}
	if req.DayOfWeek < 1 || req.DayOfWeek > 7 {
		fields = append(fields, response.FieldError{Field: "dayOfWeek", Message: "must be 1 to 7"})
	}
	if !validScheduleKind(academicdomain.ScheduleKind(req.Kind)) {
		fields = append(fields, response.FieldError{Field: "kind", Message: "invalid"})
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	out, err := h.schedule.CreateSchedule(c.Request.Context(), scheduleuc.ScheduleInput{ClassID: req.ClassID, SubjectID: req.SubjectID, TeacherUserID: req.TeacherUserID, DayOfWeek: req.DayOfWeek, StartTime: req.StartTime, EndTime: req.EndTime, Kind: academicdomain.ScheduleKind(req.Kind)})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusCreated, scheduleJSON(out))
}

func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	if !ValidUUID(c.Param("scheduleId")) {
		uuidError(c, "scheduleId")
		return
	}
	var raw map[string]json.RawMessage
	if !bindAcademic(c, &raw) {
		return
	}
	var req struct {
		ClassID       *string `json:"classId"`
		SubjectID     *string `json:"subjectId"`
		StartTime     *string `json:"startTime"`
		EndTime       *string `json:"endTime"`
		Kind          *string `json:"kind"`
		TeacherUserID *string `json:"teacherUserId"`
		DayOfWeek     *int16  `json:"dayOfWeek"`
	}
	body, _ := json.Marshal(raw)
	if err := json.Unmarshal(body, &req); err != nil {
		validationBody(c)
		return
	}
	_, classSet := raw["classId"]
	_, subjectSet := raw["subjectId"]
	_, teacherSet := raw["teacherUserId"]
	_, daySet := raw["dayOfWeek"]
	_, startSet := raw["startTime"]
	_, endSet := raw["endTime"]
	_, kindSet := raw["kind"]
	if !classSet && !subjectSet && !teacherSet && !daySet && !startSet && !endSet && !kindSet {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "request", Message: "at least one field is required"})
		return
	}
	input := scheduleuc.ScheduleInput{}
	if classSet {
		if req.ClassID == nil {
			uuidError(c, "classId")
			return
		}
		input.ClassID = *req.ClassID
		if !ValidUUID(input.ClassID) {
			uuidError(c, "classId")
			return
		}
	}
	if subjectSet {
		if req.SubjectID == nil {
			uuidError(c, "subjectId")
			return
		}
		input.SubjectID = *req.SubjectID
		if !ValidUUID(input.SubjectID) {
			uuidError(c, "subjectId")
			return
		}
	}
	if teacherSet {
		input.TeacherUserID = req.TeacherUserID
		if req.TeacherUserID != nil && !ValidUUID(*req.TeacherUserID) {
			uuidError(c, "teacherUserId")
			return
		}
	}
	if daySet {
		if req.DayOfWeek == nil {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "dayOfWeek", Message: "required"})
			return
		}
		input.DayOfWeek = *req.DayOfWeek
		if input.DayOfWeek < 1 || input.DayOfWeek > 7 {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "dayOfWeek", Message: "must be 1 to 7"})
			return
		}
	}
	if startSet {
		if req.StartTime == nil {
			validationBody(c)
			return
		}
		input.StartTime = *req.StartTime
	}
	if endSet {
		if req.EndTime == nil {
			validationBody(c)
			return
		}
		input.EndTime = *req.EndTime
	}
	if kindSet {
		if req.Kind == nil {
			validationBody(c)
			return
		}
		input.Kind = academicdomain.ScheduleKind(*req.Kind)
		if !validScheduleKind(input.Kind) {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "kind", Message: "invalid"})
			return
		}
	}
	out, err := h.schedule.UpdateSchedule(c.Request.Context(), c.Param("scheduleId"), input, classSet, subjectSet, teacherSet, daySet, startSet, endSet, kindSet)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	c.JSON(http.StatusOK, scheduleJSON(out))
}

func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	if !ValidUUID(c.Param("scheduleId")) {
		uuidError(c, "scheduleId")
		return
	}
	if err := h.schedule.DeleteSchedule(c.Request.Context(), c.Param("scheduleId")); err != nil {
		writeMappedError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func scheduleDateDays(c *gin.Context) ([]int16, bool) {
	fromRaw, toRaw := c.Query("dateFrom"), c.Query("dateTo")
	if fromRaw == "" && toRaw == "" {
		return nil, true
	}
	from, fromErr := time.Parse("2006-01-02", fromRaw)
	to, toErr := time.Parse("2006-01-02", toRaw)
	if fromErr != nil || toErr != nil || from.After(to) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "dateFrom", Message: "invalid date range"})
		return nil, false
	}
	seen := map[int16]struct{}{}
	for day := from; !day.After(to) && len(seen) < 7; day = day.AddDate(0, 0, 1) {
		seen[int16((int(day.Weekday())+6)%7+1)] = struct{}{}
	}
	out := make([]int16, 0, len(seen))
	for day := range seen {
		out = append(out, day)
	}
	return out, true
}

func containsDay(days []int16, day int16) bool {
	for _, item := range days {
		if item == day {
			return true
		}
	}
	return false
}

func validScheduleKind(kind academicdomain.ScheduleKind) bool {
	return kind == academicdomain.ScheduleKindLesson || kind == academicdomain.ScheduleKindExam || kind == academicdomain.ScheduleKindTryout
}

func scheduleJSON(item academicdomain.Schedule) gin.H {
	return gin.H{"id": item.ID, "classId": item.ClassID, "subjectId": item.SubjectID, "teacherUserId": item.TeacherUserID, "dayOfWeek": item.DayOfWeek, "startTime": item.StartTime.Format("15:04"), "endTime": item.EndTime.Format("15:04"), "kind": item.Kind}
}
