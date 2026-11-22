package settings

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"
	"limen/internal/platform/cache"
)

type settingsRepoMock struct {
	getResults []struct {
		value attendancedomain.SchoolSettings
		err   error
	}
	created attendancedomain.SchoolSettings
	updated attendancedomain.SchoolSettings
}

func (m *settingsRepoMock) Create(_ context.Context, value attendancedomain.SchoolSettings) error {
	m.created = value
	return nil
}
func (m *settingsRepoMock) Get(context.Context) (attendancedomain.SchoolSettings, error) {
	result := m.getResults[0]
	if len(m.getResults) > 1 {
		m.getResults = m.getResults[1:]
	}
	return result.value, result.err
}
func (m *settingsRepoMock) Update(_ context.Context, value attendancedomain.SchoolSettings) error {
	m.updated = value
	return nil
}

type cacheMock struct {
	hit                           bool
	cached                        Output
	getCalls, setCalls, bumpCalls int
}

func (m *cacheMock) Get(_ context.Context, _ string, dest any) (bool, error) {
	m.getCalls++
	if m.hit {
		*(dest.(*Output)) = m.cached
	}
	return m.hit, nil
}
func (m *cacheMock) Set(context.Context, string, any, time.Duration) error { m.setCalls++; return nil }
func (m *cacheMock) Delete(context.Context, ...string) error               { return nil }
func (m *cacheMock) GetVersion(context.Context, string) (int64, error)     { return 1, nil }
func (m *cacheMock) BumpVersion(context.Context, string) (int64, error)    { m.bumpCalls++; return 2, nil }

func TestGetUsesCachedSettings(t *testing.T) {
	cached := Output{ID: 1, FirstEntryTime: "07:00", Timezone: "Asia/Jakarta", LateToleranceMinutes: 10}
	c := &cacheMock{hit: true, cached: cached}
	uc := NewUseCase(&settingsRepoMock{}, c, time.Hour, logrus.New())
	out, err := uc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if out != cached {
		t.Fatalf("expected cached output %#v, got %#v", cached, out)
	}
}

func TestUpdateCreatesDefaultSettingsAndInvalidatesCache(t *testing.T) {
	now := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	initial := attendancedomain.SchoolSettings{ID: 1, FirstEntryTime: time.Date(0, 1, 1, 7, 0, 0, 0, time.UTC), Timezone: "Asia/Jakarta"}
	updated := initial
	updated.FirstEntryTime = time.Date(0, 1, 1, 8, 30, 0, 0, time.UTC)
	updated.LateToleranceMinutes = 15
	updated.UpdatedAt = now
	repo := &settingsRepoMock{getResults: []struct {
		value attendancedomain.SchoolSettings
		err   error
	}{{err: domain.ErrNotFound}, {value: initial}, {value: updated}}}
	c := &cacheMock{}
	uc := NewUseCase(repo, c, time.Hour, logrus.New())
	uc.now = func() time.Time { return now }
	late := 15
	out, err := uc.Update(context.Background(), UpdateInput{FirstEntryTime: "08:30", LateToleranceMinutes: &late, UpdatedBy: "admin-1"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.created.ID != 1 || repo.updated.LateToleranceMinutes != 15 || repo.updated.UpdatedBy == nil || *repo.updated.UpdatedBy != "admin-1" {
		t.Fatalf("unexpected repository writes: created=%#v updated=%#v", repo.created, repo.updated)
	}
	if c.bumpCalls != 1 || out.FirstEntryTime != "08:30" || out.LateToleranceMinutes != 15 {
		t.Fatalf("unexpected output/cache state: %#v %#v", out, c)
	}
}

var _ cache.Cache = (*cacheMock)(nil)
