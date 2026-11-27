package assignment

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	assignmentdomain "limen/internal/domain/assignment"
	userdomain "limen/internal/domain/user"
)

type mockAssignmentRepo struct {
	assignments map[string]assignmentdomain.Assignment
	next        int
}

func (m *mockAssignmentRepo) Create(_ context.Context, a assignmentdomain.Assignment) (string, error) {
	m.next++
	id := "asg-1"
	a.ID = id
	m.assignments[id] = a
	return id, nil
}
func (m *mockAssignmentRepo) FindByID(_ context.Context, id string) (assignmentdomain.Assignment, error) {
	a, ok := m.assignments[id]
	if !ok {
		return assignmentdomain.Assignment{}, domain.ErrNotFound
	}
	return a, nil
}
func (m *mockAssignmentRepo) Update(_ context.Context, a assignmentdomain.Assignment) error {
	m.assignments[a.ID] = a
	return nil
}
func (m *mockAssignmentRepo) List(_ context.Context, _ assignmentdomain.AssignmentFilter, _, _ int) ([]assignmentdomain.Assignment, int64, error) {
	var list []assignmentdomain.Assignment
	for _, a := range m.assignments {
		list = append(list, a)
	}
	return list, int64(len(list)), nil
}
func (m *mockAssignmentRepo) Delete(_ context.Context, id string) error {
	delete(m.assignments, id)
	return nil
}

type mockSubmissionRepo struct {
	submissions map[string]assignmentdomain.AssignmentSubmission
}

func (m *mockSubmissionRepo) Create(_ context.Context, s assignmentdomain.AssignmentSubmission) (string, error) {
	key := s.AssignmentID + ":" + s.StudentUserID
	s.ID = "sub-" + key
	m.submissions[key] = s
	return s.ID, nil
}
func (m *mockSubmissionRepo) FindByID(_ context.Context, id string) (assignmentdomain.AssignmentSubmission, error) {
	for _, s := range m.submissions {
		if s.ID == id {
			return s, nil
		}
	}
	return assignmentdomain.AssignmentSubmission{}, domain.ErrNotFound
}
func (m *mockSubmissionRepo) FindByAssignmentAndStudent(_ context.Context, assignmentID, studentUserID string) (assignmentdomain.AssignmentSubmission, error) {
	key := assignmentID + ":" + studentUserID
	s, ok := m.submissions[key]
	if !ok {
		return assignmentdomain.AssignmentSubmission{}, domain.ErrNotFound
	}
	return s, nil
}
func (m *mockSubmissionRepo) Upsert(_ context.Context, s assignmentdomain.AssignmentSubmission) error {
	key := s.AssignmentID + ":" + s.StudentUserID
	s.ID = "sub-" + key
	m.submissions[key] = s
	return nil
}
func (m *mockSubmissionRepo) Update(_ context.Context, s assignmentdomain.AssignmentSubmission) error {
	key := s.AssignmentID + ":" + s.StudentUserID
	m.submissions[key] = s
	return nil
}
func (m *mockSubmissionRepo) FindDetail(_ context.Context, assignmentID, studentUserID string) (assignmentdomain.SubmissionDetail, error) {
	key := assignmentID + ":" + studentUserID
	s, ok := m.submissions[key]
	if !ok {
		return assignmentdomain.SubmissionDetail{}, domain.ErrNotFound
	}
	return assignmentdomain.SubmissionDetail{
		AssignmentSubmission: s,
		StudentName:          "Student One",
	}, nil
}
func (m *mockSubmissionRepo) List(_ context.Context, assignmentID string, _, _ int) ([]assignmentdomain.SubmissionDetail, int64, error) {
	var list []assignmentdomain.SubmissionDetail
	for _, s := range m.submissions {
		if s.AssignmentID == assignmentID {
			list = append(list, assignmentdomain.SubmissionDetail{AssignmentSubmission: s, StudentName: "Student One"})
		}
	}
	return list, int64(len(list)), nil
}
func (m *mockSubmissionRepo) ListByStudentUserID(_ context.Context, studentUserID string) ([]assignmentdomain.AssignmentSubmission, error) {
	var list []assignmentdomain.AssignmentSubmission
	for _, s := range m.submissions {
		if s.StudentUserID == studentUserID {
			list = append(list, s)
		}
	}
	return list, nil
}

type mockClassRepo struct {
	classes map[string]academicdomain.Class
}

func (m *mockClassRepo) Create(context.Context, academicdomain.Class) (string, error) { return "", nil }
func (m *mockClassRepo) FindByID(_ context.Context, id string) (academicdomain.Class, error) {
	c, ok := m.classes[id]
	if !ok {
		return academicdomain.Class{}, domain.ErrNotFound
	}
	return c, nil
}
func (m *mockClassRepo) Update(context.Context, academicdomain.Class) error { return nil }
func (m *mockClassRepo) List(context.Context) ([]academicdomain.Class, error) {
	var list []academicdomain.Class
	for _, c := range m.classes {
		list = append(list, c)
	}
	return list, nil
}

type mockSubjectRepo struct {
	subjects map[string]academicdomain.Subject
}

func (m *mockSubjectRepo) Create(context.Context, academicdomain.Subject) (string, error) {
	return "", nil
}
func (m *mockSubjectRepo) FindByID(_ context.Context, id string) (academicdomain.Subject, error) {
	s, ok := m.subjects[id]
	if !ok {
		return academicdomain.Subject{}, domain.ErrNotFound
	}
	return s, nil
}
func (m *mockSubjectRepo) FindByCode(context.Context, string) (academicdomain.Subject, error) {
	return academicdomain.Subject{}, nil
}
func (m *mockSubjectRepo) List(context.Context) ([]academicdomain.Subject, error) {
	var list []academicdomain.Subject
	for _, s := range m.subjects {
		list = append(list, s)
	}
	return list, nil
}
func (m *mockSubjectRepo) Update(context.Context, academicdomain.Subject) error { return nil }

type mockStudentRepo struct {
	students map[string]academicdomain.Student
}

func (m *mockStudentRepo) Create(context.Context, academicdomain.Student) (string, error) {
	return "", nil
}
func (m *mockStudentRepo) FindByUserID(_ context.Context, userID string) (academicdomain.Student, error) {
	s, ok := m.students[userID]
	if !ok {
		return academicdomain.Student{}, domain.ErrNotFound
	}
	return s, nil
}
func (m *mockStudentRepo) FindByNIS(context.Context, string) (academicdomain.Student, error) {
	return academicdomain.Student{}, nil
}
func (m *mockStudentRepo) Update(context.Context, academicdomain.Student) error { return nil }
func (m *mockStudentRepo) ListByClassID(context.Context, string) ([]academicdomain.Student, error) {
	return nil, nil
}

type mockUserRepo struct {
	users map[string]userdomain.User
}

func (m *mockUserRepo) Create(context.Context, userdomain.User) (string, error) { return "", nil }
func (m *mockUserRepo) CreateWithRoles(context.Context, userdomain.User, []userdomain.UserRole, bool) (string, error) {
	return "", nil
}
func (m *mockUserRepo) FindByID(_ context.Context, id string) (userdomain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return userdomain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (m *mockUserRepo) FindByEmail(context.Context, string) (userdomain.User, error) {
	return userdomain.User{}, nil
}
func (m *mockUserRepo) List(context.Context, string, string, string, int, int) ([]userdomain.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) Update(context.Context, userdomain.User) error { return nil }
func (m *mockUserRepo) Delete(context.Context, string) error          { return nil }

func setupAssignmentUseCase() (*UseCase, *mockAssignmentRepo, *mockSubmissionRepo, *mockClassRepo, *mockSubjectRepo, *mockStudentRepo, *mockUserRepo) {
	asgRepo := &mockAssignmentRepo{assignments: make(map[string]assignmentdomain.Assignment)}
	subRepo := &mockSubmissionRepo{submissions: make(map[string]assignmentdomain.AssignmentSubmission)}
	clsRepo := &mockClassRepo{classes: make(map[string]academicdomain.Class)}
	subDocRepo := &mockSubjectRepo{subjects: make(map[string]academicdomain.Subject)}
	stdRepo := &mockStudentRepo{students: make(map[string]academicdomain.Student)}
	usrRepo := &mockUserRepo{users: make(map[string]userdomain.User)}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(asgRepo, subRepo, clsRepo, subDocRepo, stdRepo, usrRepo, logger)
	return uc, asgRepo, subRepo, clsRepo, subDocRepo, stdRepo, usrRepo
}

func TestAssignment_Create(t *testing.T) {
	uc, _, _, clsRepo, subRepo, _, usrRepo := setupAssignmentUseCase()

	// Missing required fields
	_, err := uc.Create(context.Background(), CreateInput{
		ClassID: "",
		Title:   "HW 1",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	dueAt := time.Now().Add(24 * time.Hour)
	input := CreateInput{
		ClassID:       "cls-1",
		SubjectID:     "sub-1",
		TeacherUserID: "teacher-1",
		Title:         "Homework 1",
		DueAt:         dueAt,
	}

	// Class not found
	_, err = uc.Create(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing class, got %v", err)
	}

	// Subject not found
	clsRepo.classes["cls-1"] = academicdomain.Class{ID: "cls-1"}
	_, err = uc.Create(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing subject, got %v", err)
	}

	// Teacher user not found
	subRepo.subjects["sub-1"] = academicdomain.Subject{ID: "sub-1"}
	_, err = uc.Create(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing teacher user, got %v", err)
	}

	// Success
	usrRepo.users["teacher-1"] = userdomain.User{ID: "teacher-1"}
	asg, err := uc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asg.ID != "asg-1" || asg.Title != "Homework 1" {
		t.Fatalf("unexpected created assignment: %+v", asg)
	}
}

func TestAssignment_Update(t *testing.T) {
	uc, asgRepo, _, clsRepo, subRepo, _, usrRepo := setupAssignmentUseCase()
	asgRepo.assignments["asg-1"] = assignmentdomain.Assignment{ID: "asg-1", ClassID: "cls-1", SubjectID: "sub-1", TeacherUserID: "teacher-1", Title: "Tugas Lama", DueAt: time.Now()}
	clsRepo.classes["cls-2"] = academicdomain.Class{ID: "cls-2"}
	subRepo.subjects["sub-2"] = academicdomain.Subject{ID: "sub-2"}
	usrRepo.users["teacher-1"] = userdomain.User{ID: "teacher-1"}
	dueAt := time.Now().Add(48 * time.Hour).Round(time.Second)
	updated, err := uc.Update(context.Background(), "asg-1", UpdateInput{ClassID: "cls-2", SubjectID: "sub-2", Title: " Tugas Baru ", DueAt: dueAt})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ClassID != "cls-2" || updated.SubjectID != "sub-2" || updated.Title != "Tugas Baru" || !updated.DueAt.Equal(dueAt) {
		t.Fatalf("unexpected updated assignment: %#v", updated)
	}
}

func TestAssignment_ListAndListSubmissions(t *testing.T) {
	uc, asgRepo, subRepo, _, _, _, _ := setupAssignmentUseCase()
	asgRepo.assignments["asg-1"] = assignmentdomain.Assignment{ID: "asg-1", Title: "Assignment 1"}

	// List assignments
	list, total, err := uc.List(context.Background(), assignmentdomain.AssignmentFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 assignment, got %d, total %d", len(list), total)
	}

	// List submissions - assignment not found
	_, _, err = uc.ListSubmissions(context.Background(), "asg-notfound", 1, 10)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// List submissions - success
	subRepo.submissions["asg-1:stu-1"] = assignmentdomain.AssignmentSubmission{AssignmentID: "asg-1", StudentUserID: "stu-1"}
	subs, subTotal, err := uc.ListSubmissions(context.Background(), "asg-1", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subTotal != 1 || len(subs) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(subs))
	}
}

func TestAssignment_GradeSubmission(t *testing.T) {
	uc, asgRepo, _, _, _, stdRepo, _ := setupAssignmentUseCase()
	asgRepo.assignments["asg-1"] = assignmentdomain.Assignment{ID: "asg-1", ClassID: "cls-1"}

	// Invalid score (NaN or Inf)
	_, err := uc.GradeSubmission(context.Background(), "asg-1", "stu-1", "teacher-1", math.NaN(), nil)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for NaN score, got %v", err)
	}

	// Assignment not found
	_, err = uc.GradeSubmission(context.Background(), "asg-none", "stu-1", "teacher-1", 90.0, nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing assignment, got %v", err)
	}

	// Student not found
	_, err = uc.GradeSubmission(context.Background(), "asg-1", "stu-none", "teacher-1", 90.0, nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing student, got %v", err)
	}

	// Student not in the assignment class
	diffClass := "cls-diff"
	stdRepo.students["stu-diff"] = academicdomain.Student{UserID: "stu-diff", ClassID: &diffClass}
	_, err = uc.GradeSubmission(context.Background(), "asg-1", "stu-diff", "teacher-1", 90.0, nil)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for class mismatch, got %v", err)
	}

	// Student has nil class
	stdRepo.students["stu-noclass"] = academicdomain.Student{UserID: "stu-noclass", ClassID: nil}
	_, err = uc.GradeSubmission(context.Background(), "asg-1", "stu-noclass", "teacher-1", 90.0, nil)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for nil student class, got %v", err)
	}

	// Success
	cls1 := "cls-1"
	stdRepo.students["stu-1"] = academicdomain.Student{UserID: "stu-1", ClassID: &cls1}
	note := "Good work!"
	detail, err := uc.GradeSubmission(context.Background(), "asg-1", "stu-1", "teacher-1", 95.5, &note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Score == nil || *detail.Score != 95.5 {
		t.Fatalf("expected score 95.5, got %v", detail.Score)
	}
	if detail.Note == nil || *detail.Note != "Good work!" {
		t.Fatalf("expected note 'Good work!', got %v", detail.Note)
	}
}
