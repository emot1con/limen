package schedule

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

type mockScheduleRepo struct {
	schedules map[string]academicdomain.Schedule
	next      int
}

func (m *mockScheduleRepo) Create(_ context.Context, s academicdomain.Schedule) (string, error) {
	m.next++
	id := "sch-1"
	s.ID = id
	m.schedules[id] = s
	return id, nil
}
func (m *mockScheduleRepo) FindByID(_ context.Context, id string) (academicdomain.Schedule, error) {
	s, ok := m.schedules[id]
	if !ok {
		return academicdomain.Schedule{}, domain.ErrNotFound
	}
	return s, nil
}
func (m *mockScheduleRepo) List(_ context.Context, filter academicdomain.ScheduleFilter, _, _ int) ([]academicdomain.Schedule, int64, error) {
	var list []academicdomain.Schedule
	for _, s := range m.schedules {
		if filter.ClassID != "" && s.ClassID != filter.ClassID {
			continue
		}
		list = append(list, s)
	}
	return list, int64(len(list)), nil
}
func (m *mockScheduleRepo) Update(_ context.Context, s academicdomain.Schedule) error {
	m.schedules[s.ID] = s
	return nil
}
func (m *mockScheduleRepo) Delete(_ context.Context, id string) error {
	delete(m.schedules, id)
	return nil
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

func setupScheduleUseCase() (*UseCase, *mockScheduleRepo, *mockClassRepo, *mockSubjectRepo, *mockUserRepo, *mockCache) {
	schRepo := &mockScheduleRepo{schedules: make(map[string]academicdomain.Schedule)}
	clsRepo := &mockClassRepo{classes: make(map[string]academicdomain.Class)}
	subRepo := &mockSubjectRepo{subjects: make(map[string]academicdomain.Subject)}
	usrRepo := &mockUserRepo{users: make(map[string]userdomain.User)}
	memCache := &mockCache{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(schRepo, clsRepo, subRepo, usrRepo, memCache, 10*time.Minute, logger)
	return uc, schRepo, clsRepo, subRepo, usrRepo, memCache
}

func TestSchedule_CreateSchedule(t *testing.T) {
	uc, _, clsRepo, subRepo, usrRepo, _ := setupScheduleUseCase()

	// Invalid format of time
	_, err := uc.CreateSchedule(context.Background(), ScheduleInput{
		ClassID:   "cls-1",
		SubjectID: "sub-1",
		StartTime: "invalid-time",
		EndTime:   "08:00",
		DayOfWeek: 1,
		Kind:      academicdomain.ScheduleKindLesson,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for invalid time format, got %v", err)
	}

	// StartTime not before EndTime
	_, err = uc.CreateSchedule(context.Background(), ScheduleInput{
		ClassID:   "cls-1",
		SubjectID: "sub-1",
		StartTime: "09:00",
		EndTime:   "08:00",
		DayOfWeek: 1,
		Kind:      academicdomain.ScheduleKindLesson,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid when start >= end, got %v", err)
	}

	teacherID := "teacher-1"
	input := ScheduleInput{
		ClassID:       "cls-1",
		SubjectID:     "sub-1",
		TeacherUserID: &teacherID,
		TeacherSet:    true,
		StartTime:     "07:30",
		EndTime:       "09:00",
		DayOfWeek:     1,
		Kind:          academicdomain.ScheduleKindLesson,
	}

	// Class not found
	_, err = uc.CreateSchedule(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing class, got %v", err)
	}

	// Subject not found
	clsRepo.classes["cls-1"] = academicdomain.Class{ID: "cls-1"}
	_, err = uc.CreateSchedule(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing subject, got %v", err)
	}

	// Teacher not found
	subRepo.subjects["sub-1"] = academicdomain.Subject{ID: "sub-1"}
	_, err = uc.CreateSchedule(context.Background(), input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing teacher, got %v", err)
	}

	// Success
	usrRepo.users["teacher-1"] = userdomain.User{ID: "teacher-1"}
	sch, err := uc.CreateSchedule(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sch.ID != "sch-1" || sch.Kind != academicdomain.ScheduleKindLesson {
		t.Fatalf("unexpected created schedule: %+v", sch)
	}
}

func TestSchedule_ListAndUpdateAndDelete(t *testing.T) {
	uc, schRepo, clsRepo, subRepo, _, _ := setupScheduleUseCase()
	clsRepo.classes["cls-1"] = academicdomain.Class{ID: "cls-1"}
	subRepo.subjects["sub-1"] = academicdomain.Subject{ID: "sub-1"}

	st, _ := time.Parse("15:04", "07:30")
	et, _ := time.Parse("15:04", "09:00")
	schRepo.schedules["sch-1"] = academicdomain.Schedule{
		ID:        "sch-1",
		ClassID:   "cls-1",
		SubjectID: "sub-1",
		DayOfWeek: 1,
		StartTime: st,
		EndTime:   et,
		Kind:      academicdomain.ScheduleKindLesson,
	}

	// List schedules
	list, total, err := uc.ListSchedules(context.Background(), academicdomain.ScheduleFilter{ClassID: "cls-1"}, 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(list))
	}

	// Update schedule
	updated, err := uc.UpdateSchedule(context.Background(), "sch-1", ScheduleInput{
		StartTime: "08:00",
		EndTime:   "09:30",
	}, false, false, false, false, true, true, false)
	if err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}
	expectedSt, _ := time.Parse("15:04", "08:00")
	if !updated.StartTime.Equal(expectedSt) {
		t.Fatalf("expected start time 08:00, got %v", updated.StartTime)
	}

	// Delete schedule
	err = uc.DeleteSchedule(context.Background(), "sch-1")
	if err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}
	if _, ok := schRepo.schedules["sch-1"]; ok {
		t.Fatal("expected schedule to be deleted")
	}
}
