package academic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	userdomain "limen/internal/domain/user"
)

type mockClassRepo struct {
	classes map[string]academicdomain.Class
	next    int
}

func (m *mockClassRepo) Create(_ context.Context, c academicdomain.Class) (string, error) {
	m.next++
	id := "cls-1"
	c.ID = id
	m.classes[id] = c
	return id, nil
}
func (m *mockClassRepo) FindByID(_ context.Context, id string) (academicdomain.Class, error) {
	c, ok := m.classes[id]
	if !ok {
		return academicdomain.Class{}, domain.ErrNotFound
	}
	return c, nil
}
func (m *mockClassRepo) Update(_ context.Context, c academicdomain.Class) error {
	m.classes[c.ID] = c
	return nil
}
func (m *mockClassRepo) List(context.Context) ([]academicdomain.Class, error) {
	var list []academicdomain.Class
	for _, c := range m.classes {
		list = append(list, c)
	}
	return list, nil
}

type mockSubjectRepo struct {
	subjects map[string]academicdomain.Subject
	next     int
}

func (m *mockSubjectRepo) Create(_ context.Context, s academicdomain.Subject) (string, error) {
	m.next++
	id := "sub-1"
	s.ID = id
	m.subjects[id] = s
	return id, nil
}
func (m *mockSubjectRepo) FindByID(_ context.Context, id string) (academicdomain.Subject, error) {
	s, ok := m.subjects[id]
	if !ok {
		return academicdomain.Subject{}, domain.ErrNotFound
	}
	return s, nil
}
func (m *mockSubjectRepo) FindByCode(_ context.Context, code string) (academicdomain.Subject, error) {
	for _, s := range m.subjects {
		if s.Code == code {
			return s, nil
		}
	}
	return academicdomain.Subject{}, domain.ErrNotFound
}
func (m *mockSubjectRepo) List(context.Context) ([]academicdomain.Subject, error) {
	var list []academicdomain.Subject
	for _, s := range m.subjects {
		list = append(list, s)
	}
	return list, nil
}
func (m *mockSubjectRepo) Update(context.Context, academicdomain.Subject) error { return nil }

type mockTeachingAssignmentRepo struct {
	assignments map[string]academicdomain.TeachingAssignment
	next        int
}

func (m *mockTeachingAssignmentRepo) Create(_ context.Context, ta academicdomain.TeachingAssignment) (string, error) {
	m.next++
	id := "ta-1"
	ta.ID = id
	m.assignments[id] = ta
	return id, nil
}
func (m *mockTeachingAssignmentRepo) FindByID(_ context.Context, id string) (academicdomain.TeachingAssignment, error) {
	ta, ok := m.assignments[id]
	if !ok {
		return academicdomain.TeachingAssignment{}, domain.ErrNotFound
	}
	return ta, nil
}
func (m *mockTeachingAssignmentRepo) ListByTeacherUserID(_ context.Context, teacherID string) ([]academicdomain.TeachingAssignment, error) {
	var list []academicdomain.TeachingAssignment
	for _, ta := range m.assignments {
		if ta.TeacherUserID == teacherID {
			list = append(list, ta)
		}
	}
	return list, nil
}
func (m *mockTeachingAssignmentRepo) ListByClassID(_ context.Context, classID string) ([]academicdomain.TeachingAssignment, error) {
	var list []academicdomain.TeachingAssignment
	for _, ta := range m.assignments {
		if ta.ClassID == classID {
			list = append(list, ta)
		}
	}
	return list, nil
}
func (m *mockTeachingAssignmentRepo) Delete(_ context.Context, id string) error {
	delete(m.assignments, id)
	return nil
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
func (m *mockStudentRepo) Update(_ context.Context, s academicdomain.Student) error {
	m.students[s.UserID] = s
	return nil
}
func (m *mockStudentRepo) ListByClassID(context.Context, string) ([]academicdomain.Student, error) {
	return nil, nil
}

type mockQueryRepo struct {
	classes     []academicdomain.ClassDetail
	subjects    []academicdomain.Subject
	students    []academicdomain.StudentDetail
	assignments []academicdomain.TeachingAssignment
	existsMap   map[string]bool
}

func (m *mockQueryRepo) ListClasses(context.Context, string, string, int, int) ([]academicdomain.ClassDetail, int64, error) {
	return m.classes, int64(len(m.classes)), nil
}
func (m *mockQueryRepo) ListSubjects(context.Context, string, int, int) ([]academicdomain.Subject, int64, error) {
	return m.subjects, int64(len(m.subjects)), nil
}
func (m *mockQueryRepo) ListStudents(context.Context, string, string, string, int, int) ([]academicdomain.StudentDetail, int64, error) {
	return m.students, int64(len(m.students)), nil
}
func (m *mockQueryRepo) FindStudent(_ context.Context, id string) (academicdomain.StudentDetail, error) {
	for _, s := range m.students {
		if s.UserID == id {
			return s, nil
		}
	}
	return academicdomain.StudentDetail{}, domain.ErrNotFound
}
func (m *mockQueryRepo) ListAssignments(context.Context, string, string, string, int, int) ([]academicdomain.TeachingAssignment, int64, error) {
	return m.assignments, int64(len(m.assignments)), nil
}
func (m *mockQueryRepo) AssignmentExists(_ context.Context, teacherID, subjectID, classID string) (bool, error) {
	key := teacherID + ":" + subjectID + ":" + classID
	return m.existsMap[key], nil
}

type mockCache struct {
	bumps int
}

func (*mockCache) Get(context.Context, string, any) (bool, error)        { return false, nil }
func (*mockCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (*mockCache) Delete(context.Context, ...string) error               { return nil }
func (*mockCache) GetVersion(context.Context, string) (int64, error)     { return 1, nil }
func (m *mockCache) BumpVersion(context.Context, string) (int64, error) {
	m.bumps++
	return 2, nil
}

func setupAcademicUseCase() (*UseCase, *mockClassRepo, *mockSubjectRepo, *mockTeachingAssignmentRepo, *mockUserRepo, *mockStudentRepo, *mockQueryRepo, *mockCache) {
	clsRepo := &mockClassRepo{classes: make(map[string]academicdomain.Class)}
	subRepo := &mockSubjectRepo{subjects: make(map[string]academicdomain.Subject)}
	taRepo := &mockTeachingAssignmentRepo{assignments: make(map[string]academicdomain.TeachingAssignment)}
	usrRepo := &mockUserRepo{users: make(map[string]userdomain.User)}
	stdRepo := &mockStudentRepo{students: make(map[string]academicdomain.Student)}
	qRepo := &mockQueryRepo{existsMap: make(map[string]bool)}
	cMock := &mockCache{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(clsRepo, subRepo, taRepo, usrRepo, stdRepo, qRepo, cMock, 10*time.Minute, logger)
	return uc, clsRepo, subRepo, taRepo, usrRepo, stdRepo, qRepo, cMock
}

func TestAcademic_ClassOperations(t *testing.T) {
	uc, clsRepo, _, _, usrRepo, _, _, _ := setupAcademicUseCase()

	// Empty name
	_, err := uc.CreateClass(context.Background(), "  ", nil, nil)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty name, got %v", err)
	}

	// Teacher not found
	tID := "t-missing"
	_, err = uc.CreateClass(context.Background(), "X IPA 1", nil, &tID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing teacher, got %v", err)
	}

	// Success create
	usrRepo.users["t-1"] = userdomain.User{ID: "t-1"}
	teacher1 := "t-1"
	cls, err := uc.CreateClass(context.Background(), "X IPA 1", nil, &teacher1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cls.ID != "cls-1" || cls.Name != "X IPA 1" {
		t.Fatalf("unexpected created class: %+v", cls)
	}

	// Update class - empty name error
	emptyName := "  "
	_, err = uc.UpdateClass(context.Background(), "cls-1", &emptyName, nil, nil, false)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty update name, got %v", err)
	}

	// Update class - success
	newName := "X IPA 2"
	updated, err := uc.UpdateClass(context.Background(), "cls-1", &newName, nil, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "X IPA 2" {
		t.Fatalf("unexpected class name: %s", updated.Name)
	}
	if clsRepo.classes["cls-1"].Name != "X IPA 2" {
		t.Fatal("class not updated in repo")
	}
}

func TestAcademic_SubjectOperations(t *testing.T) {
	uc, _, _, _, _, _, _, _ := setupAcademicUseCase()

	// Invalid input
	_, err := uc.CreateSubject(context.Background(), "", "Matematika")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty code, got %v", err)
	}

	// Success
	sub, err := uc.CreateSubject(context.Background(), "MTK", "Matematika")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.ID != "sub-1" || sub.Code != "MTK" {
		t.Fatalf("unexpected subject: %+v", sub)
	}
}

func TestAcademic_StudentOperations(t *testing.T) {
	uc, clsRepo, _, _, _, stdRepo, qRepo, _ := setupAcademicUseCase()

	stdRepo.students["stu-1"] = academicdomain.Student{
		UserID: "stu-1",
	}
	qRepo.students = []academicdomain.StudentDetail{
		{UserID: "stu-1", FullName: "Ahmad"},
	}

	// Get Student
	detail, err := uc.GetStudent(context.Background(), "stu-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.FullName != "Ahmad" {
		t.Fatalf("unexpected student detail: %+v", detail)
	}

	// Update Student - invalid empty NIS
	emptyNIS := "  "
	_, err = uc.UpdateStudent(context.Background(), "stu-1", &emptyNIS, nil, true, false)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty NIS, got %v", err)
	}

	// Update Student - class not found
	missingClass := "cls-none"
	_, err = uc.UpdateStudent(context.Background(), "stu-1", nil, &missingClass, false, true)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing class, got %v", err)
	}

	// Update Student - success
	clsRepo.classes["cls-1"] = academicdomain.Class{ID: "cls-1"}
	nis := "12345"
	targetClass := "cls-1"
	_, err = uc.UpdateStudent(context.Background(), "stu-1", &nis, &targetClass, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdRepo.students["stu-1"].NIS == nil || *stdRepo.students["stu-1"].NIS != "12345" {
		t.Fatal("expected NIS to be updated")
	}
}

func TestAcademic_TeachingAssignment(t *testing.T) {
	uc, clsRepo, subRepo, taRepo, usrRepo, _, qRepo, _ := setupAcademicUseCase()

	// Missing references
	_, err := uc.CreateAssignment(context.Background(), "t-1", "sub-1", "cls-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing teacher, got %v", err)
	}

	usrRepo.users["t-1"] = userdomain.User{ID: "t-1"}
	subRepo.subjects["sub-1"] = academicdomain.Subject{ID: "sub-1"}
	clsRepo.classes["cls-1"] = academicdomain.Class{ID: "cls-1"}

	// Success create
	ta, err := uc.CreateAssignment(context.Background(), "t-1", "sub-1", "cls-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ta.ID != "ta-1" {
		t.Fatalf("unexpected assignment: %+v", ta)
	}
	if taRepo.assignments["ta-1"].TeacherUserID != "t-1" {
		t.Fatal("assignment not saved in repo")
	}

	// Already exists error
	qRepo.existsMap["t-1:sub-1:cls-1"] = true
	_, err = uc.CreateAssignment(context.Background(), "t-1", "sub-1", "cls-1")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}
