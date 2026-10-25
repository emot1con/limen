package settings

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"limen/internal/domain"
	attendancedomain "limen/internal/domain/attendance"
)

type UseCase struct {
	repo   attendancedomain.SchoolSettingsRepository
	now    func() time.Time
	logger *logrus.Logger
}

func NewUseCase(repo attendancedomain.SchoolSettingsRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{repo: repo, now: time.Now, logger: logger}
}

type Output struct {
	ID                   int16
	FirstEntryTime       string
	Timezone             string
	LateToleranceMinutes int
	UpdatedBy            *string
	UpdatedAt            time.Time
}

type UpdateInput struct {
	FirstEntryTime       string
	LateToleranceMinutes *int
	UpdatedBy            string
}

func (uc *UseCase) Get(ctx context.Context) (Output, error) {
	settings, err := uc.ensure(ctx)
	if err != nil {
		uc.logError(err, "settings_get")
		return Output{}, err
	}
	return toOutput(settings), nil
}

func (uc *UseCase) Update(ctx context.Context, input UpdateInput) (Output, error) {
	entry, err := time.Parse("15:04", input.FirstEntryTime)
	if err != nil {
		return Output{}, domain.ErrInvalid
	}
	settings, err := uc.ensure(ctx)
	if err != nil {
		uc.logError(err, "settings_get")
		return Output{}, err
	}
	settings.FirstEntryTime = entry
	if input.LateToleranceMinutes != nil {
		if *input.LateToleranceMinutes < 0 {
			return Output{}, domain.ErrInvalid
		}
		settings.LateToleranceMinutes = *input.LateToleranceMinutes
	}
	settings.UpdatedBy = &input.UpdatedBy
	settings.UpdatedAt = uc.now()
	if err := uc.repo.Update(ctx, settings); err != nil {
		uc.logError(err, "settings_update")
		return Output{}, err
	}
	updated, err := uc.repo.Get(ctx)
	if err != nil {
		uc.logError(err, "settings_get_updated")
		return Output{}, err
	}
	return toOutput(updated), nil
}

func (uc *UseCase) ensure(ctx context.Context) (attendancedomain.SchoolSettings, error) {
	settings, err := uc.repo.Get(ctx)
	if err == nil {
		return settings, nil
	}
	if err != domain.ErrNotFound {
		return attendancedomain.SchoolSettings{}, err
	}
	entry, _ := time.Parse("15:04", "07:00")
	created := attendancedomain.SchoolSettings{ID: 1, FirstEntryTime: entry, Timezone: "Asia/Jakarta", LateToleranceMinutes: 0, UpdatedAt: uc.now()}
	if err := uc.repo.Create(ctx, created); err != nil {
		settings, getErr := uc.repo.Get(ctx)
		if getErr == nil {
			return settings, nil
		}
		return attendancedomain.SchoolSettings{}, err
	}
	return uc.repo.Get(ctx)
}

func (uc *UseCase) logError(err error, operation string) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists {
		return
	}
	uc.logger.WithError(err).WithField("operation", operation).Error("usecase failed")
}

func toOutput(settings attendancedomain.SchoolSettings) Output {
	return Output{
		ID:                   settings.ID,
		FirstEntryTime:       settings.FirstEntryTime.Format("15:04"),
		Timezone:             settings.Timezone,
		LateToleranceMinutes: settings.LateToleranceMinutes,
		UpdatedBy:            settings.UpdatedBy,
		UpdatedAt:            settings.UpdatedAt,
	}
}
