package academic

import (
	"context"
	"fmt"
	"strings"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	userdomain "limen/internal/domain/user"

	"github.com/sirupsen/logrus"
)

type Repository interface {
	ListClasses(context.Context, string, string, int, int) ([]Class, int64, error)
	ListSubjects(context.Context, string, int, int) ([]academicdomain.Subject, int64, error)
	ListStudents(context.Context, string, string, string, int, int) ([]Student, int64, error)
	FindStudent(context.Context, string) (Student, error)
	ListAssignments(context.Context, string, string, string, int, int) ([]academicdomain.TeachingAssignment, int64, error)
	AssignmentExists(context.Context, string, string, string) (bool, error)
}

type Class struct {
	academicdomain.Class
	StudentCount int64
}

type Student struct {
	UserID    string
	FullName  string
	NIS       *string
	ClassID   *string
	ClassName *string
	Status    userdomain.Status
}

type UseCase struct {
	classes     academicdomain.ClassRepository
	subjects    academicdomain.SubjectRepository
	assignments academicdomain.TeachingAssignmentRepository
	users       userdomain.UserRepository
	repo        Repository
	students    academicdomain.StudentRepository
	logger      *logrus.Logger
}

func NewUseCase(classes academicdomain.ClassRepository, subjects academicdomain.SubjectRepository, assignments academicdomain.TeachingAssignmentRepository, users userdomain.UserRepository, students academicdomain.StudentRepository, repo Repository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{classes: classes, subjects: subjects, assignments: assignments, users: users, students: students, repo: repo, logger: logger}
}

func (uc *UseCase) ListClasses(ctx context.Context, level, q string, page, pageSize int) ([]Class, int64, error) {
	items, total, err := uc.repo.ListClasses(ctx, level, q, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "academic_list_classes", nil)
	}
	return items, total, err
}
func (uc *UseCase) ListSubjects(ctx context.Context, q string, page, pageSize int) ([]academicdomain.Subject, int64, error) {
	items, total, err := uc.repo.ListSubjects(ctx, q, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "academic_list_subjects", nil)
	}
	return items, total, err
}
func (uc *UseCase) ListStudents(ctx context.Context, classID, q, sort string, page, pageSize int) ([]Student, int64, error) {
	items, total, err := uc.repo.ListStudents(ctx, classID, q, sort, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "academic_list_students", nil)
	}
	return items, total, err
}
func (uc *UseCase) GetStudent(ctx context.Context, id string) (Student, error) {
	item, err := uc.repo.FindStudent(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_student", logrus.Fields{"student_id": id})
	}
	return item, err
}
func (uc *UseCase) UpdateStudent(ctx context.Context, id string, nis *string, classID *string, nisSet, classSet bool) (Student, error) {
	student, err := uc.students.FindByUserID(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_student_for_update", logrus.Fields{"student_id": id})
		return Student{}, err
	}
	if nisSet {
		if nis == nil || strings.TrimSpace(*nis) == "" {
			return Student{}, domain.ErrInvalid
		}
		value := strings.TrimSpace(*nis)
		student.NIS = &value
	}
	if classSet {
		if classID != nil {
			if _, err := uc.classes.FindByID(ctx, *classID); err != nil {
				uc.logError(err, "academic_validate_student_class", logrus.Fields{"class_id": *classID})
				return Student{}, err
			}
		}
		student.ClassID = classID
	}
	if err := uc.students.Update(ctx, student); err != nil {
		uc.logError(err, "academic_update_student", logrus.Fields{"student_id": id})
		return Student{}, err
	}
	updated, err := uc.repo.FindStudent(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_updated_student", logrus.Fields{"student_id": id})
	}
	return updated, err
}

func (uc *UseCase) ListAssignments(ctx context.Context, teacherID, classID, subjectID string, page, pageSize int) ([]academicdomain.TeachingAssignment, int64, error) {
	items, total, err := uc.repo.ListAssignments(ctx, teacherID, classID, subjectID, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "academic_list_assignments", nil)
	}
	return items, total, err
}

func (uc *UseCase) CreateClass(ctx context.Context, name string, level, teacherID *string) (academicdomain.Class, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return academicdomain.Class{}, domain.ErrInvalid
	}
	if teacherID != nil {
		if _, err := uc.users.FindByID(ctx, *teacherID); err != nil {
			uc.logError(err, "academic_validate_class_teacher", logrus.Fields{"teacher_id": *teacherID})
			return academicdomain.Class{}, err
		}
	}
	fmt.Println(name, level, teacherID)
	id, err := uc.classes.Create(ctx, academicdomain.Class{Name: name, Level: trim(level), HomeroomTeacherUserID: teacherID})
	if err != nil {
		uc.logError(err, "academic_create_class", nil)
		return academicdomain.Class{}, err
	}

	item, err := uc.classes.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_created_class", logrus.Fields{"class_id": id})
	}
	return item, err
}

func (uc *UseCase) UpdateClass(ctx context.Context, id string, name, level *string, teacherID *string, teacherSet bool) (academicdomain.Class, error) {
	item, err := uc.classes.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_class", logrus.Fields{"class_id": id})
		return item, err
	}
	if name != nil {
		item.Name = strings.TrimSpace(*name)
		if item.Name == "" {
			return item, domain.ErrInvalid
		}
	}
	if level != nil {
		item.Level = trim(level)
	}
	if teacherSet {
		if teacherID != nil {
			if _, err := uc.users.FindByID(ctx, *teacherID); err != nil {
				uc.logError(err, "academic_validate_class_teacher", logrus.Fields{"teacher_id": *teacherID})
				return item, err
			}
		}
		item.HomeroomTeacherUserID = teacherID
	}
	if err := uc.classes.Update(ctx, item); err != nil {
		uc.logError(err, "academic_update_class", logrus.Fields{"class_id": id})
		return item, err
	}
	item, err = uc.classes.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_updated_class", logrus.Fields{"class_id": id})
	}
	return item, err
}

func (uc *UseCase) CreateSubject(ctx context.Context, code, name string) (academicdomain.Subject, error) {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if code == "" || name == "" {
		return academicdomain.Subject{}, domain.ErrInvalid
	}
	id, err := uc.subjects.Create(ctx, academicdomain.Subject{Code: code, Name: name})
	if err != nil {
		uc.logError(err, "academic_create_subject", nil)
		return academicdomain.Subject{}, err
	}
	item, err := uc.subjects.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_created_subject", logrus.Fields{"subject_id": id})
	}
	return item, err
}
func (uc *UseCase) CreateAssignment(ctx context.Context, teacherID, subjectID, classID string) (academicdomain.TeachingAssignment, error) {
	if _, err := uc.users.FindByID(ctx, teacherID); err != nil {
		uc.logError(err, "academic_validate_assignment_teacher", logrus.Fields{"teacher_id": teacherID})
		return academicdomain.TeachingAssignment{}, err
	}
	if _, err := uc.subjects.FindByID(ctx, subjectID); err != nil {
		uc.logError(err, "academic_validate_assignment_subject", logrus.Fields{"subject_id": subjectID})
		return academicdomain.TeachingAssignment{}, err
	}
	if _, err := uc.classes.FindByID(ctx, classID); err != nil {
		uc.logError(err, "academic_validate_assignment_class", logrus.Fields{"class_id": classID})
		return academicdomain.TeachingAssignment{}, err
	}
	exists, err := uc.repo.AssignmentExists(ctx, teacherID, subjectID, classID)
	if err != nil {
		uc.logError(err, "academic_check_assignment", nil)
		return academicdomain.TeachingAssignment{}, err
	}
	if exists {
		return academicdomain.TeachingAssignment{}, domain.ErrAlreadyExists
	}
	id, err := uc.assignments.Create(ctx, academicdomain.TeachingAssignment{TeacherUserID: teacherID, SubjectID: subjectID, ClassID: classID})
	if err != nil {
		uc.logError(err, "academic_create_assignment", nil)
		return academicdomain.TeachingAssignment{}, err
	}
	item, err := uc.assignments.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "academic_get_created_assignment", logrus.Fields{"assignment_id": id})
	}
	return item, err
}
func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}

func trim(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}
