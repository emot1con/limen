package fingerprint

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	fingerprintdomain "limen/internal/domain/fingerprint"
	userdomain "limen/internal/domain/user"
)

type mockGateway struct {
	identifyResp fingerprintdomain.Response
	identifyErr  error
	enrollResp   fingerprintdomain.Response
	enrollErr    error
}

func (m *mockGateway) Identify(ctx context.Context, template []byte, requestID string) (fingerprintdomain.Response, error) {
	return m.identifyResp, m.identifyErr
}
func (m *mockGateway) Enroll(ctx context.Context, externalUserID, template, requestID string) (fingerprintdomain.Response, error) {
	return m.enrollResp, m.enrollErr
}
func (m *mockGateway) EnrollRaw(ctx context.Context, externalUserID string, template []byte, requestID string) (fingerprintdomain.Response, error) {
	return m.enrollResp, m.enrollErr
}

type mockFingerprintRepo struct {
	saved  map[string]any
	latest map[string]any
}

func (m *mockFingerprintRepo) Save(_ context.Context, result map[string]any) error {
	m.saved = result
	m.latest = result
	return nil
}
func (m *mockFingerprintRepo) Latest(context.Context) (map[string]any, error) {
	if m.latest == nil {
		return nil, domain.ErrNotFound
	}
	return m.latest, nil
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

func setupFingerprintUseCase() (*UseCase, *mockGateway, *mockFingerprintRepo, *mockUserRepo) {
	gw := &mockGateway{}
	repo := &mockFingerprintRepo{}
	userRepo := &mockUserRepo{users: make(map[string]userdomain.User)}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	uc := NewUseCase(gw, repo, userRepo, logger)
	return uc, gw, repo, userRepo
}

func TestFingerprint_Identify(t *testing.T) {
	uc, gw, repo, userRepo := setupFingerprintUseCase()

	// Invalid template length
	_, err := uc.Identify(context.Background(), []byte("short"), "req-1")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for wrong template size, got %v", err)
	}

	validTemplate := make([]byte, FingerprintSize)
	userRepo.users["usr-10"] = userdomain.User{
		ID:       "usr-10",
		FullName: "Budi Santoso",
		Email:    "budi@example.com",
	}

	gw.identifyResp = fingerprintdomain.Response{
		Status: 200,
		Body: map[string]any{
			"matched":        true,
			"externalUserId": "usr-10",
			"score":          92,
		},
	}

	resp, err := uc.Identify(context.Background(), validTemplate, "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, ok := resp.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any body, got %T", resp.Body)
	}
	if body["fullName"] != "Budi Santoso" || body["email"] != "budi@example.com" {
		t.Fatalf("expected enriched user details, got %+v", body)
	}
	if repo.saved == nil {
		t.Fatalf("expected result to be persisted to repo")
	}

	// Test Latest (retrieved from store)
	latest, err := uc.Latest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Latest: %v", err)
	}
	if latest["fullName"] != "Budi Santoso" {
		t.Fatalf("unexpected latest value: %+v", latest)
	}
}

func TestFingerprint_Enroll(t *testing.T) {
	uc, gw, _, _ := setupFingerprintUseCase()

	// Enroll with empty strings
	_, err := uc.Enroll(context.Background(), "", "template", "req-1")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	gw.enrollResp = fingerprintdomain.Response{Status: 200, Body: "ok"}
	resp, err := uc.Enroll(context.Background(), "user-1", "b64template", "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("unexpected status: %d", resp.Status)
	}

	// EnrollRaw with bad size
	_, err = uc.EnrollRaw(context.Background(), "user-1", []byte("too-short"), "req-1")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	// EnrollRaw success
	rawTemplate := bytes.Repeat([]byte{1}, FingerprintSize)
	respRaw, err := uc.EnrollRaw(context.Background(), "user-1", rawTemplate, "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respRaw.Status != 200 {
		t.Fatalf("unexpected status: %d", respRaw.Status)
	}
}

func TestFingerprint_Store_Subscription(t *testing.T) {
	store := NewStore()
	ch, unsubscribe := store.Subscribe()
	defer unsubscribe()

	record := map[string]any{"matched": true, "score": 88}
	recorded := store.Record(record)

	select {
	case received := <-ch:
		if received["score"] != 88 || received["scanId"] != 1 {
			t.Fatalf("unexpected received event: %+v", received)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for store subscriber event")
	}

	if recorded["scanId"] != 1 {
		t.Fatalf("expected scanId=1, got %v", recorded["scanId"])
	}
}
