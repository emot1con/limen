package academic

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	userdomain "limen/internal/domain/user"
)

type QueryRepository struct{ db *gorm.DB }

func NewQueryRepository(db *gorm.DB) *QueryRepository { return &QueryRepository{db: db} }

func (r *QueryRepository) ListClasses(ctx context.Context, level, q string, limit, offset int) ([]academicdomain.ClassDetail, int64, error) {
	query := r.db.WithContext(ctx).Table("classes c").Where("c.deleted_at IS NULL")
	if level != "" {
		query = query.Where("c.level = ?", level)
	}
	if q != "" {
		query = query.Where("c.name ILIKE ?", "%"+q+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID, Name                     string
		Level, HomeroomTeacherUserID *string
		StudentCount                 int64
	}
	err := query.Select("c.id, c.name, c.level, c.homeroom_teacher_user_id, COUNT(s.user_id) AS student_count").Joins("LEFT JOIN students s ON s.class_id = c.id AND s.deleted_at IS NULL").Group("c.id").Order("c.name").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]academicdomain.ClassDetail, len(rows))
	for i, v := range rows {
		out[i] = academicdomain.ClassDetail{Class: academicdomain.Class{ID: v.ID, Name: v.Name, Level: v.Level, HomeroomTeacherUserID: v.HomeroomTeacherUserID}, StudentCount: v.StudentCount}
	}
	return out, total, nil
}
func (r *QueryRepository) ListSubjects(ctx context.Context, q string, limit, offset int) ([]academicdomain.Subject, int64, error) {
	query := r.db.WithContext(ctx).Model(&subjectModel{})
	if q != "" {
		query = query.Where("code ILIKE ? OR name ILIKE ?", "%"+q+"%", "%"+q+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []subjectModel
	if err := query.Order("code").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]academicdomain.Subject, len(models))
	for i, v := range models {
		out[i] = academicdomain.Subject{ID: v.ID, Code: v.Code, Name: v.Name}
	}
	return out, total, nil
}
func (r *QueryRepository) ListStudents(ctx context.Context, classID, q, sort string, limit, offset int) ([]academicdomain.StudentDetail, int64, error) {
	query := r.studentQuery(ctx, classID, q)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "u.full_name"
	if sort == "nis" {
		order = "s.nis"
	}
	var rows []studentRow
	if err := query.Select(studentSelect).Order(order).Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return students(rows), total, nil
}
func (r *QueryRepository) FindStudent(ctx context.Context, id string) (academicdomain.StudentDetail, error) {
	var rows []studentRow
	err := r.studentQuery(ctx, "", "").Where("s.user_id = ?", id).Select(studentSelect).Scan(&rows).Error
	if err != nil {
		return academicdomain.StudentDetail{}, err
	}
	if len(rows) == 0 {
		return academicdomain.StudentDetail{}, domain.ErrNotFound
	}
	return students(rows)[0], nil
}

const studentSelect = "s.user_id, u.full_name, s.nis, s.class_id, c.name AS class_name, u.status"

type studentRow struct {
	UserID, FullName string
	NIS              *string
	ClassID          *string
	ClassName        *string
	Status           userdomain.Status
}

func (r *QueryRepository) studentQuery(ctx context.Context, classID, q string) *gorm.DB {
	query := r.db.WithContext(ctx).Table("students s").Joins("JOIN users u ON u.id = s.user_id").Joins("LEFT JOIN classes c ON c.id = s.class_id AND c.deleted_at IS NULL").Where("s.deleted_at IS NULL")
	if classID != "" {
		query = query.Where("s.class_id = ?", classID)
	}
	if q != "" {
		query = query.Where("u.full_name ILIKE ? OR s.nis ILIKE ?", "%"+q+"%", "%"+q+"%")
	}
	return query
}
func students(rows []studentRow) []academicdomain.StudentDetail {
	out := make([]academicdomain.StudentDetail, len(rows))
	for i, v := range rows {
		out[i] = academicdomain.StudentDetail{UserID: v.UserID, FullName: v.FullName, NIS: v.NIS, ClassID: v.ClassID, ClassName: v.ClassName, Status: v.Status}
	}
	return out
}
func (r *QueryRepository) ListAssignments(ctx context.Context, teacherID, classID, subjectID string, limit, offset int) ([]academicdomain.TeachingAssignment, int64, error) {
	query := r.db.WithContext(ctx).Model(&teachingAssignmentModel{})
	if teacherID != "" {
		query = query.Where("teacher_user_id = ?", teacherID)
	}
	if classID != "" {
		query = query.Where("class_id = ?", classID)
	}
	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []teachingAssignmentModel
	if err := query.Order("created_at").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]academicdomain.TeachingAssignment, len(models))
	for i, v := range models {
		out[i] = academicdomain.TeachingAssignment{ID: v.ID, TeacherUserID: v.TeacherUserID, SubjectID: v.SubjectID, ClassID: v.ClassID}
	}
	return out, total, nil
}
func (r *QueryRepository) AssignmentExists(ctx context.Context, teacherID, subjectID, classID string) (bool, error) {
	var model teachingAssignmentModel
	err := r.db.WithContext(ctx).Where("teacher_user_id = ? AND subject_id = ? AND class_id = ?", teacherID, subjectID, classID).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

var _ academicdomain.QueryRepository = (*QueryRepository)(nil)
