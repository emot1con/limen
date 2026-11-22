package assessment

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	assessmentdomain "limen/internal/domain/assessment"
	userdomain "limen/internal/domain/user"
)

type mockAssessmentRepo struct {
	assessments map[string]assessmentdomain.Assessment
	next        int
}

func (m *mockAssessmentRepo) Create(_ context.Context, a assessmentdomain.Assessment) (string, error) {
	m.next++
	id := "asm-1"
	a.ID = id
	m.assessments[id] = a
	return id, nil
}
func (m *mockAssessmentRepo) FindByID(_ context.Context, id string) (assessmentdomain.Assessment, error) {
	a, ok := m.assessments[id]
	if !ok {
		return assessmentdomain.Assessment{}, domain.ErrNotFound
	}
	return a, nil
}
func (m *mockAssessmentRepo) Update(_ context.Context, a assessmentdomain.Assessment) error {
	m.assessments[a.ID] = a
	return nil
}
func (m *mockAssessmentRepo) List(_ context.Context, _ assessmentdomain.AssessmentFilter, _, _ int) ([]assessmentdomain.Assessment, int64, error) {
	var list []assessmentdomain.Assessment
	for _, a := range m.assessments {
		list = append(list, a)
	}
	return list, int64(len(list)), nil
}

type mockScoreRepo struct {
	scores map[string]assessmentdomain.AssessmentScore
}

func (m *mockScoreRepo) Create(context.Context, assessmentdomain.AssessmentScore) (string, error) {
	return "", nil
}
func (m *mockScoreRepo) FindByID(context.Context, string) (assessmentdomain.AssessmentScore, error) {
	return assessmentdomain.AssessmentScore{}, nil
}
func (m *mockScoreRepo) FindByAssessmentAndStudent(context.Context, string, string) (assessmentdomain.AssessmentScore, error) {
	return assessmentdomain.AssessmentScore{}, nil
}
func (m *mockScoreRepo) Upsert(_ context.Context, s assessmentdomain.AssessmentScore) error {
	key := s.AssessmentID + ":" + s.StudentUserID
	s.ID = "score-" + key
	m.scores[key] = s
	return nil
}
func (m *mockScoreRepo) List(_ context.Context, assessmentID string, _, _ int) ([]assessmentdomain.ScoreDetail, int64, error) {
	var list []assessmentdomain.ScoreDetail
	for _, s := range m.scores {
		if s.AssessmentID == assessmentID {
			list = append(list, assessmentdomain.ScoreDetail{AssessmentScore: s, StudentName: "Student One"})
		}
	}
	return list, int64(len(list)), nil
}
func (m *mockScoreRepo) FindDetail(_ context.Context, assessmentID, studentUserID string) (assessmentdomain.ScoreDetail, error) {
	key := assessmentID + ":" + studentUserID
	s, ok := m.scores[key]
	if !ok {
		return assessmentdomain.ScoreDetail{}, domain.ErrNotFound
	}
	return assessmentdomain.ScoreDetail{
		AssessmentScore: s,
		StudentName:     "Student One",
	}, nil
}
func (m *mockScoreRepo) ListByStudentUserID(context.Context, string) ([]assessmentdomain.AssessmentScore, error) {
	return nil, nil
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

func setupAssessmentUseCase() (*UseCase, *mockAssessmentRepo, *mockScoreRepo, *mockClassRepo, *mockSubjectRepo, *mockStudentRepo, *mockUserRepo) {
	asmRepo := &mockAssessmentRepo{assessments: make(map[string]assessmentdomain.Assessment)}
	scrRepo := &mockScoreRepo{scores: make(map[string]assessmentdomain.AssessmentScore)}
	clsRepo := &mockClassRepo{classes: make(map[string]academicdomain.Class)}
	subRepo := &mockSubjectRepo{subjects: make(map[string]academicdomain.Subject)}
	stdRepo := &mockStudentRepo{students: make(map[string]academicdomain.Student)}
	usrRepo := &mockUserRepo{users: make(map[string]userdomain.User)}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(asmRepo, scrRepo, clsRepo, subRepo, stdRepo, usrRepo, logger)
	return uc, asmRepo, scrRepo, clsRepo, subRepo, stdRepo, usrRepo
}

func TestAssessment_Create(t *testing.T) {
	uc, _, _, clsRepo, subRepo, _, usrRepo := setupAssessmentUseCase()

	// Missing fields
	_, err := uc.Create(context.Background(), CreateInput{
		ClassID: "",
		Kind:    "UTS",
		Period:  "2026/2027 Ganjil",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	// Invalid KKM
	badKKM := -10.0
	_, err = uc.Create(context.Background(), CreateInput{
		ClassID:       "cls-1",
		SubjectID:     "sub-1",
		TeacherUserID: "teacher-1",
		Kind:          "UTS",
		Period:        "2026/2027 Ganjil",
		KKM:           &badKKM,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for negative KKM, got %v", err)
	}

	validKKM := 75.0
	input := CreateInput{
		ClassID:       "cls-1",
		SubjectID:     "sub-1",
		TeacherUserID: "teacher-1",
		Kind:          "UTS",
		Period:        "2026/2027 Ganjil",
		KKM:           &validKKM,
	}

	// Missing class
	_, err = uc.Create(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for class, got %v", err)
	}

	// Missing subject
	clsRepo.classes["cls-1"] = academicdomain.Class{ID: "cls-1"}
	_, err = uc.Create(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for subject, got %v", err)
	}

	// Missing teacher
	subRepo.subjects["sub-1"] = academicdomain.Subject{ID: "sub-1"}
	_, err = uc.Create(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for teacher, got %v", err)
	}

	// Success
	usrRepo.users["teacher-1"] = userdomain.User{ID: "teacher-1"}
	res, err := uc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "asm-1" || res.Kind != "UTS" || res.KKM == nil || *res.KKM != 75.0 {
		t.Fatalf("unexpected assessment: %+v", res)
	}
}

func TestAssessment_ListAndListScores(t *testing.T) {
	uc, asmRepo, scrRepo, _, _, _, _ := setupAssessmentUseCase()
	asmRepo.assessments["asm-1"] = assessmentdomain.Assessment{ID: "asm-1", Kind: "UAS"}

	// List assessments
	list, total, err := uc.List(context.Background(), assessmentdomain.AssessmentFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 assessment, got %d", len(list))
	}

	// List scores - assessment not found
	_, _, err = uc.ListScores(context.Background(), "asm-none", 1, 10)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// List scores - success
	scrRepo.scores["asm-1:stu-1"] = assessmentdomain.AssessmentScore{AssessmentID: "asm-1", StudentUserID: "stu-1", Score: 88.0}
	scores, sTotal, err := uc.ListScores(context.Background(), "asm-1", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sTotal != 1 || len(scores) != 1 {
		t.Fatalf("expected 1 score detail, got %d", len(scores))
	}
}

func TestAssessment_UpsertScore(t *testing.T) {
	uc, asmRepo, _, _, _, stdRepo, _ := setupAssessmentUseCase()
	asmRepo.assessments["asm-1"] = assessmentdomain.Assessment{ID: "asm-1", ClassID: "cls-1"}

	// Invalid score NaN
	_, err := uc.UpsertScore(context.Background(), "asm-1", "stu-1", "teacher-1", math.NaN())
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for NaN, got %v", err)
	}

	// Assessment not found
	_, err = uc.UpsertScore(context.Background(), "asm-none", "stu-1", "teacher-1", 85.0)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for assessment, got %v", err)
	}

	// Student not found
	_, err = uc.UpsertScore(context.Background(), "asm-1", "stu-none", "teacher-1", 85.0)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for student, got %v", err)
	}

	// Student class mismatch
	diffClass := "cls-diff"
	stdRepo.students["stu-diff"] = academicdomain.Student{UserID: "stu-diff", ClassID: &diffClass}
	_, err = uc.UpsertScore(context.Background(), "asm-1", "stu-diff", "teacher-1", 85.0)
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for class mismatch, got %v", err)
	}

	// Success
	cls1 := "cls-1"
	stdRepo.students["stu-1"] = academicdomain.Student{UserID: "stu-1", ClassID: &cls1}
	scoreDetail, err := uc.UpsertScore(context.Background(), "asm-1", "stu-1", "teacher-1", 85.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scoreDetail.Score != 85.5 || scoreDetail.StudentName != "Student One" {
		t.Fatalf("unexpected score detail: %+v", scoreDetail)
	}
}
