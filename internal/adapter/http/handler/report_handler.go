package handler

import (
	"net/http"
	"strings"
	"time"

	"limen/internal/adapter/http/response"
	attendancedomain "limen/internal/domain/attendance"
	reportsdomain "limen/internal/domain/reports"
	reportsuc "limen/internal/usecase/reports"

	"github.com/gin-gonic/gin"
)

type ReportsHandler struct{ reports *reportsuc.UseCase }

func NewReportsHandler(reports *reportsuc.UseCase) *ReportsHandler {
	return &ReportsHandler{reports: reports}
}

func (h *ReportsHandler) Attendance(c *gin.Context) {
	for _, key := range []string{"classId", "studentUserId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	filter := reportsdomain.AttendanceFilter{
		ClassID:       c.Query("classId"),
		StudentUserID: c.Query("studentUserId"),
	}
	if raw := strings.TrimSpace(c.Query("subjectType")); raw != "" {
		if raw != string(attendancedomain.SubjectTypeStudent) && raw != string(attendancedomain.SubjectTypeTeacher) {
			response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "subjectType", Message: "must be student or teacher"})
			return
		}
		filter.SubjectType = attendancedomain.SubjectType(raw)
	}
	var fields []response.FieldError
	if raw := strings.TrimSpace(c.Query("dateFrom")); raw != "" {
		value, err := time.Parse("2006-01-02", raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "dateFrom", Message: "must be a date (YYYY-MM-DD)"})
		} else {
			filter.DateFrom = &value
		}
	}
	if raw := strings.TrimSpace(c.Query("dateTo")); raw != "" {
		value, err := time.Parse("2006-01-02", raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "dateTo", Message: "must be a date (YYYY-MM-DD)"})
		} else {
			filter.DateTo = &value
		}
	}
	if len(fields) > 0 {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", fields...)
		return
	}
	if filter.DateFrom != nil && filter.DateTo != nil && filter.DateFrom.After(*filter.DateTo) {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "dateFrom", Message: "must not be after dateTo"})
		return
	}
	groupBy, ok := parseAttendanceGroupBy(c.Query("groupBy"))
	if !ok {
		response.WriteError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "validation failed", response.FieldError{Field: "groupBy", Message: "must be student, class, or day"})
		return
	}
	filter.GroupBy = groupBy

	items, err := h.reports.Attendance(c.Request.Context(), filter)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = attendanceSummaryJSON(item, groupBy)
	}
	c.JSON(http.StatusOK, gin.H{"groupBy": string(groupBy), "items": out})
}

func (h *ReportsHandler) Grades(c *gin.Context) {
	page, size, ok := pageQuery(c)
	if !ok {
		return
	}
	for _, key := range []string{"classId", "subjectId"} {
		if value := c.Query(key); value != "" && !ValidUUID(value) {
			uuidError(c, key)
			return
		}
	}
	filter := reportsdomain.GradeFilter{ClassID: c.Query("classId"), SubjectID: c.Query("subjectId"), Period: c.Query("period")}
	items, total, err := h.reports.Grades(c.Request.Context(), filter, page, size)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	out := make([]gin.H, len(items))
	for i, item := range items {
		out[i] = gradeSummaryJSON(item)
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total, "page": page, "pageSize": size})
}

func (h *ReportsHandler) ReportCard(c *gin.Context) {
	studentID := c.Param("studentId")
	if !ValidUUID(studentID) {
		uuidError(c, "studentId")
		return
	}
	period := strings.TrimSpace(c.Query("period"))
	items, err := h.reports.ReportCard(c.Request.Context(), studentID, period)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	subjects := make([]gin.H, len(items))
	for i, item := range items {
		subjects[i] = gin.H{"subjectId": item.SubjectID, "subjectName": item.SubjectName, "score": item.Score, "kkm": item.KKM}
	}
	c.JSON(http.StatusOK, gin.H{"studentUserId": studentID, "period": period, "subjects": subjects})
}

func parseAttendanceGroupBy(raw string) (reportsdomain.AttendanceGroupBy, bool) {
	switch strings.TrimSpace(raw) {
	case "", string(reportsdomain.AttendanceGroupByStudent):
		return reportsdomain.AttendanceGroupByStudent, true
	case string(reportsdomain.AttendanceGroupByClass):
		return reportsdomain.AttendanceGroupByClass, true
	case string(reportsdomain.AttendanceGroupByDay):
		return reportsdomain.AttendanceGroupByDay, true
	default:
		return "", false
	}
}

func attendanceSummaryJSON(item reportsdomain.AttendanceSummary, groupBy reportsdomain.AttendanceGroupBy) gin.H {
	out := gin.H{"present": item.Present, "late": item.Late, "excused": item.Excused, "sick": item.Sick, "absent": item.Absent, "total": item.Total}
	switch groupBy {
	case reportsdomain.AttendanceGroupByClass:
		out["classId"] = item.ClassID
		out["className"] = item.ClassName
	case reportsdomain.AttendanceGroupByDay:
		if item.Date != nil {
			out["date"] = item.Date.Format("2006-01-02")
		} else {
			out["date"] = nil
		}
	default:
		out["studentUserId"] = item.StudentUserID
		out["studentName"] = item.StudentName
		out["classId"] = item.ClassID
		out["className"] = item.ClassName
	}
	return out
}

func gradeSummaryJSON(item reportsdomain.GradeSummary) gin.H {
	return gin.H{
		"studentUserId":   item.StudentUserID,
		"studentName":     item.StudentName,
		"classId":         item.ClassID,
		"className":       item.ClassName,
		"subjectId":       item.SubjectID,
		"subjectName":     item.SubjectName,
		"averageScore":    item.AverageScore,
		"assessmentCount": item.AssessmentCount,
	}
}
