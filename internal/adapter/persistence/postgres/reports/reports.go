package reports

import (
	"context"

	reportsdomain "limen/internal/domain/reports"

	"gorm.io/gorm"
)

type QueryRepository struct {
	db *gorm.DB
}

func NewQueryRepository(db *gorm.DB) *QueryRepository {
	return &QueryRepository{db: db}
}

const attendanceStatusSelect = `SUM(CASE WHEN ar.status = 'hadir' THEN 1 ELSE 0 END) AS present,
	SUM(CASE WHEN ar.status = 'terlambat' THEN 1 ELSE 0 END) AS late,
	SUM(CASE WHEN ar.status = 'izin' THEN 1 ELSE 0 END) AS excused,
	SUM(CASE WHEN ar.status = 'sakit' THEN 1 ELSE 0 END) AS sick,
	SUM(CASE WHEN ar.status = 'alpa' THEN 1 ELSE 0 END) AS absent,
	COUNT(*) AS total`

func (r *QueryRepository) ListAttendanceSummaries(ctx context.Context, filter reportsdomain.AttendanceFilter) ([]reportsdomain.AttendanceSummary, error) {
	query := r.db.WithContext(ctx).Table("attendance_records ar").Where("ar.deleted_at IS NULL")
	if filter.SubjectType != "" {
		query = query.Where("ar.subject_type = ?", filter.SubjectType)
	}
	if filter.StudentUserID != "" {
		query = query.Where("ar.subject_user_id = ?", filter.StudentUserID)
	}
	if filter.ClassID != "" {
		query = query.Where("ar.subject_user_id IN (SELECT user_id FROM students WHERE class_id = ? AND deleted_at IS NULL)", filter.ClassID)
	}
	if filter.DateFrom != nil {
		query = query.Where("ar.attendance_date >= ?", filter.DateFrom.Format("2006-01-02"))
	}
	if filter.DateTo != nil {
		query = query.Where("ar.attendance_date <= ?", filter.DateTo.Format("2006-01-02"))
	}

	var rows []reportsdomain.AttendanceSummary
	switch filter.GroupBy {
	case reportsdomain.AttendanceGroupByDay:
		err := query.
			Select("ar.attendance_date AS date, " + attendanceStatusSelect).
			Group("ar.attendance_date").
			Order("ar.attendance_date").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
	case reportsdomain.AttendanceGroupByClass:
		err := query.
			Select("s.class_id AS key, s.class_id AS class_id, c.name AS class_name, " + attendanceStatusSelect).
			Joins("LEFT JOIN students s ON s.user_id = ar.subject_user_id AND s.deleted_at IS NULL").
			Joins("LEFT JOIN classes c ON c.id = s.class_id AND c.deleted_at IS NULL").
			Group("s.class_id, c.name").
			Order("c.name").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
	default:
		err := query.
			Select("ar.subject_user_id AS key, ar.subject_user_id AS student_user_id, u.full_name AS student_name, s.class_id AS class_id, c.name AS class_name, " + attendanceStatusSelect).
			Joins("LEFT JOIN users u ON u.id = ar.subject_user_id").
			Joins("LEFT JOIN students s ON s.user_id = ar.subject_user_id AND s.deleted_at IS NULL").
			Joins("LEFT JOIN classes c ON c.id = s.class_id AND c.deleted_at IS NULL").
			Group("ar.subject_user_id, u.full_name, s.class_id, c.name").
			Order("u.full_name, ar.subject_user_id").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *QueryRepository) ListGradeSummaries(ctx context.Context, filter reportsdomain.GradeFilter, limit, offset int) ([]reportsdomain.GradeSummary, int64, error) {
	grouped := r.gradeSummaryQuery(ctx, filter)
	var total int64
	if err := r.db.WithContext(ctx).Table("(?) AS t", grouped).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []reportsdomain.GradeSummary
	err := r.gradeSummaryQuery(ctx, filter).
		Order("u.full_name, sub.name").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *QueryRepository) gradeSummaryQuery(ctx context.Context, filter reportsdomain.GradeFilter) *gorm.DB {
	query := r.db.WithContext(ctx).Table("assessment_scores sc").
		Joins("JOIN assessments a ON a.id = sc.assessment_id AND a.deleted_at IS NULL").
		Joins("JOIN users u ON u.id = sc.student_user_id").
		Joins("LEFT JOIN classes c ON c.id = a.class_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN subjects sub ON sub.id = a.subject_id AND sub.deleted_at IS NULL").
		Where("sc.deleted_at IS NULL")
	if filter.ClassID != "" {
		query = query.Where("a.class_id = ?", filter.ClassID)
	}
	if filter.SubjectID != "" {
		query = query.Where("a.subject_id = ?", filter.SubjectID)
	}
	if filter.Period != "" {
		query = query.Where("a.period = ?", filter.Period)
	}
	return query.
		Select(`sc.student_user_id AS student_user_id, u.full_name AS student_name,
			a.class_id AS class_id, c.name AS class_name,
			a.subject_id AS subject_id, sub.name AS subject_name,
			COALESCE(AVG(sc.score), 0) AS average_score, COUNT(*) AS assessment_count`).
		Group("sc.student_user_id, u.full_name, a.class_id, c.name, a.subject_id, sub.name")
}

func (r *QueryRepository) ListReportCardSubjects(ctx context.Context, studentUserID, period string) ([]reportsdomain.ReportCardSubject, error) {
	query := r.db.WithContext(ctx).Table("assessment_scores sc").
		Joins("JOIN assessments a ON a.id = sc.assessment_id AND a.deleted_at IS NULL").
		Joins("LEFT JOIN subjects sub ON sub.id = a.subject_id AND sub.deleted_at IS NULL").
		Where("sc.deleted_at IS NULL AND sc.student_user_id = ?", studentUserID)
	if period != "" {
		query = query.Where("a.period = ?", period)
	}
	var rows []reportsdomain.ReportCardSubject
	err := query.
		Select("a.subject_id AS subject_id, sub.name AS subject_name, COALESCE(AVG(sc.score), 0) AS score, MAX(a.kkm) AS kkm").
		Group("a.subject_id, sub.name").
		Order("sub.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

var _ reportsdomain.QueryRepository = (*QueryRepository)(nil)
