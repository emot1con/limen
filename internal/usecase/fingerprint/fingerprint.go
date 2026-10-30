package fingerprint

import (
	"context"
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	"limen/internal/domain"
	fingerprintdomain "limen/internal/domain/fingerprint"
	userdomain "limen/internal/domain/user"
)

// FingerprintSize is the exact R503 capture size accepted by the gateway.
const FingerprintSize = 18432

type UseCase struct {
	gateway fingerprintdomain.Gateway
	repo    fingerprintdomain.Repository
	users   userdomain.UserRepository
	store   *Store
	logger  *logrus.Logger
}

func NewUseCase(gateway fingerprintdomain.Gateway, repo fingerprintdomain.Repository, users userdomain.UserRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{gateway: gateway, repo: repo, users: users, store: NewStore(), logger: logger}
}

// Store exposes the in-memory latest-result hub used by the SSE endpoint.
func (uc *UseCase) Store() *Store {
	return uc.store
}

func (uc *UseCase) Identify(ctx context.Context, template []byte, requestID string) (fingerprintdomain.Response, error) {
	if len(template) != FingerprintSize {
		return fingerprintdomain.Response{}, domain.ErrInvalid
	}
	response, err := uc.gateway.Identify(ctx, template, requestID)
	if err != nil {
		uc.logError(err, "fingerprint_identify", nil)
		return fingerprintdomain.Response{}, err
	}
	if result, ok := response.Body.(map[string]any); ok {
		enriched := uc.enrich(ctx, result)
		stored := uc.store.Record(enriched)
		if saveErr := uc.repo.Save(ctx, stored); saveErr != nil {
			uc.logger.WithError(saveErr).Warn("fingerprint identification audit save failed")
		}
		response.Body = enriched
	}
	return response, nil
}

func (uc *UseCase) Enroll(ctx context.Context, externalUserID, template, requestID string) (fingerprintdomain.Response, error) {
	externalUserID, template = strings.TrimSpace(externalUserID), strings.TrimSpace(template)
	if externalUserID == "" || template == "" {
		return fingerprintdomain.Response{}, domain.ErrInvalid
	}
	response, err := uc.gateway.Enroll(ctx, externalUserID, template, requestID)
	if err != nil {
		uc.logError(err, "fingerprint_enroll", nil)
		return fingerprintdomain.Response{}, err
	}
	return response, nil
}

func (uc *UseCase) EnrollRaw(ctx context.Context, externalUserID string, template []byte, requestID string) (fingerprintdomain.Response, error) {
	externalUserID = strings.TrimSpace(externalUserID)
	if externalUserID == "" || len(template) != FingerprintSize {
		return fingerprintdomain.Response{}, domain.ErrInvalid
	}
	response, err := uc.gateway.EnrollRaw(ctx, externalUserID, template, requestID)
	if err != nil {
		uc.logError(err, "fingerprint_enroll_raw", nil)
		return fingerprintdomain.Response{}, err
	}
	return response, nil
}

// Latest returns the in-memory result, falling back to the persisted audit row
// after a restart.
func (uc *UseCase) Latest(ctx context.Context) (map[string]any, error) {
	if latest, ok := uc.store.Latest(); ok {
		return latest, nil
	}
	result, err := uc.repo.Latest(ctx)
	if err != nil {
		uc.logError(err, "fingerprint_latest", nil)
		return nil, err
	}
	return result, nil
}

// enrich adds application data (fullName, email) to a matched result. User
// data never enters the upstream fingerprint service.
func (uc *UseCase) enrich(ctx context.Context, result map[string]any) map[string]any {
	matched, _ := result["matched"].(bool)
	externalUserID, _ := result["externalUserId"].(string)
	if !matched || externalUserID == "" {
		return result
	}
	user, err := uc.users.FindByID(ctx, externalUserID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			uc.logger.WithError(err).WithField("external_user_id", externalUserID).Warn("fingerprint enrichment lookup failed")
		}
		return result
	}
	enriched := make(map[string]any, len(result)+2)
	for key, value := range result {
		enriched[key] = value
	}
	enriched["fullName"] = user.FullName
	enriched["email"] = user.Email
	return enriched
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrAlreadyExists) {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}
