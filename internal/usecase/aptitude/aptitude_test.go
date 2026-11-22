package aptitude

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	counselingdomain "limen/internal/domain/counseling"
)

type mockAptitudeRepo struct {
	results map[string][]counselingdomain.AptitudeResult
}

func (m *mockAptitudeRepo) Create(context.Context, counselingdomain.AptitudeResult) (string, error) {
	return "", nil
}
func (m *mockAptitudeRepo) FindByID(context.Context, string) (counselingdomain.AptitudeResult, error) {
	return counselingdomain.AptitudeResult{}, nil
}
func (m *mockAptitudeRepo) ListByStudentUserID(_ context.Context, userID string) ([]counselingdomain.AptitudeResult, error) {
	return m.results[userID], nil
}

type mockTargetMajorRepo struct {
	majors map[string]counselingdomain.TargetMajor
	next   int
}

func (m *mockTargetMajorRepo) Create(_ context.Context, tm counselingdomain.TargetMajor) (string, error) {
	m.next++
	id := "tm-1"
	tm.ID = id
	m.majors[id] = tm
	return id, nil
}
func (m *mockTargetMajorRepo) FindByID(_ context.Context, id string) (counselingdomain.TargetMajor, error) {
	tm, ok := m.majors[id]
	if !ok {
		return counselingdomain.TargetMajor{}, domain.ErrNotFound
	}
	return tm, nil
}
func (m *mockTargetMajorRepo) ListByStudentUserID(_ context.Context, userID string) ([]counselingdomain.TargetMajor, error) {
	var list []counselingdomain.TargetMajor
	for _, tm := range m.majors {
		if tm.StudentUserID == userID {
			list = append(list, tm)
		}
	}
	return list, nil
}
func (m *mockTargetMajorRepo) Delete(_ context.Context, id string) error {
	delete(m.majors, id)
	return nil
}

type mockPassingGradeRepo struct {
	grades map[string]counselingdomain.PassingGrade
}

func (m *mockPassingGradeRepo) Create(context.Context, counselingdomain.PassingGrade) (string, error) {
	return "", nil
}
func (m *mockPassingGradeRepo) FindByID(_ context.Context, id string) (counselingdomain.PassingGrade, error) {
	pg, ok := m.grades[id]
	if !ok {
		return counselingdomain.PassingGrade{}, domain.ErrNotFound
	}
	return pg, nil
}
func (m *mockPassingGradeRepo) List(_ context.Context, filter counselingdomain.PassingGradeFilter, _, _ int) ([]counselingdomain.PassingGrade, int64, error) {
	var list []counselingdomain.PassingGrade
	for _, pg := range m.grades {
		if filter.University != "" && pg.University != filter.University {
			continue
		}
		if filter.Major != "" && pg.Major != filter.Major {
			continue
		}
		list = append(list, pg)
	}
	return list, int64(len(list)), nil
}
func (m *mockPassingGradeRepo) Update(context.Context, counselingdomain.PassingGrade) error { return nil }
func (m *mockPassingGradeRepo) Delete(context.Context, string) error                        { return nil }

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

type mockCache struct {
	data    map[string]any
	version map[string]int64
	bumps   int
}

func (m *mockCache) Get(_ context.Context, key string, dest any) (bool, error) {
	val, ok := m.data[key]
	if !ok {
		return false, nil
	}
	switch d := dest.(type) {
	case *struct {
		Items []counselingdomain.PassingGrade
		Total int64
	}:
		if v, ok := val.(struct {
			Items []counselingdomain.PassingGrade
			Total int64
		}); ok {
			*d = v
			return true, nil
		}
	case *counselingdomain.PassingGrade:
		if v, ok := val.(counselingdomain.PassingGrade); ok {
			*d = v
			return true, nil
		}
	}
	return false, nil
}
func (m *mockCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	m.data[key] = value
	return nil
}
func (m *mockCache) Delete(context.Context, ...string) error { return nil }
func (m *mockCache) GetVersion(_ context.Context, group string) (int64, error) {
	return m.version[group], nil
}
func (m *mockCache) BumpVersion(_ context.Context, group string) (int64, error) {
	m.bumps++
	m.version[group]++
	return m.version[group], nil
}

func setupAptitudeUseCase() (*UseCase, *mockAptitudeRepo, *mockTargetMajorRepo, *mockPassingGradeRepo, *mockStudentRepo, *mockCache) {
	aptRepo := &mockAptitudeRepo{results: make(map[string][]counselingdomain.AptitudeResult)}
	tmRepo := &mockTargetMajorRepo{majors: make(map[string]counselingdomain.TargetMajor)}
	pgRepo := &mockPassingGradeRepo{grades: make(map[string]counselingdomain.PassingGrade)}
	stdRepo := &mockStudentRepo{students: make(map[string]academicdomain.Student)}
	memCache := &mockCache{data: make(map[string]any), version: make(map[string]int64)}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(aptRepo, tmRepo, pgRepo, stdRepo, memCache, 10*time.Minute, logger)
	return uc, aptRepo, tmRepo, pgRepo, stdRepo, memCache
}

func TestAptitude_ListAptitudeResults(t *testing.T) {
	uc, aptRepo, _, _, stdRepo, _ := setupAptitudeUseCase()

	// Student not found
	_, err := uc.ListAptitudeResults(context.Background(), "stu-notfound")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing student, got %v", err)
	}

	// Success
	stdRepo.students["stu-1"] = academicdomain.Student{UserID: "stu-1"}
	aptRepo.results["stu-1"] = []counselingdomain.AptitudeResult{
		{ID: "apt-1", StudentUserID: "stu-1", Result: "Tech / Engineering"},
	}
	results, err := uc.ListAptitudeResults(context.Background(), "stu-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Result != "Tech / Engineering" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestAptitude_AddTargetMajor(t *testing.T) {
	uc, _, tmRepo, _, stdRepo, _ := setupAptitudeUseCase()

	// Validation - empty major
	_, err := uc.AddTargetMajor(context.Background(), "stu-1", "   ")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty major, got %v", err)
	}

	// Student not found
	_, err = uc.AddTargetMajor(context.Background(), "stu-1", "Computer Science")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Success
	stdRepo.students["stu-1"] = academicdomain.Student{UserID: "stu-1"}
	created, err := uc.AddTargetMajor(context.Background(), "stu-1", "Computer Science")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "tm-1" || created.Major != "Computer Science" {
		t.Fatalf("unexpected target major: %+v", created)
	}
	if tmRepo.majors["tm-1"].StudentUserID != "stu-1" {
		t.Fatalf("target major not properly stored in repo")
	}
}

func TestAptitude_PassingGrades(t *testing.T) {
	uc, _, _, pgRepo, _, _ := setupAptitudeUseCase()
	pgRepo.grades["pg-1"] = counselingdomain.PassingGrade{
		ID:         "pg-1",
		University: "Universitas Indonesia",
		Major:      "Ilmu Komputer",
		Score:      650.5,
		Year:       2026,
	}

	// List passing grades (first call hits repo, second hits cache)
	list, total, err := uc.ListPassingGrades(context.Background(), counselingdomain.PassingGradeFilter{
		University: "Universitas Indonesia",
	}, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Major != "Ilmu Komputer" {
		t.Fatalf("unexpected list passing grades: %+v", list)
	}

	// Cached call
	cachedList, cachedTotal, err := uc.ListPassingGrades(context.Background(), counselingdomain.PassingGradeFilter{
		University: "Universitas Indonesia",
	}, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error on cached list: %v", err)
	}
	if cachedTotal != 1 || len(cachedList) != 1 {
		t.Fatalf("unexpected cached total/list: %+v", cachedList)
	}

	// Get passing grade by ID - not found
	_, err = uc.GetPassingGrade(context.Background(), "pg-none")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Get passing grade by ID - success & cache
	item, err := uc.GetPassingGrade(context.Background(), "pg-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Score != 650.5 {
		t.Fatalf("unexpected score: %f", item.Score)
	}

	cachedItem, err := uc.GetPassingGrade(context.Background(), "pg-1")
	if err != nil || cachedItem.Score != 650.5 {
		t.Fatalf("failed reading cached item: %v", err)
	}
}
