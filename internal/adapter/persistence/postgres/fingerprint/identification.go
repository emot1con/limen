package fingerprint

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	fingerprintdomain "limen/internal/domain/fingerprint"
)

type IdentificationRepository struct {
	db *gorm.DB
}

func NewIdentificationRepository(db *gorm.DB) *IdentificationRepository {
	return &IdentificationRepository{db: db}
}

// Save appends an audit row. The full payload is stored as jsonb so unknown
// upstream fields are preserved.
func (r *IdentificationRepository) Save(ctx context.Context, result map[string]any) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO identifications
			(external_user_id, matched, score, candidate_count, fingerprint_id, payload_json)
		 VALUES (?, ?, ?, ?, ?, ?::jsonb)`,
		optionalString(result["externalUserId"]),
		boolValue(result["matched"]),
		optionalFloat(result["score"]),
		intValue(result["candidateCount"]),
		optionalInt(result["fingerprintId"]),
		string(payload),
	).Error
}

func (r *IdentificationRepository) Latest(ctx context.Context) (map[string]any, error) {
	row := r.db.WithContext(ctx).
		Raw(`SELECT payload_json, created_at FROM identifications ORDER BY id DESC LIMIT 1`).
		Row()

	var payload []byte
	var createdAt time.Time
	if err := row.Scan(&payload, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	result := map[string]any{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
	}
	result["timestamp"] = createdAt.UTC().Format(time.RFC3339)
	return result, nil
}

func optionalString(value any) *string {
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return &s
	}
	return nil
}

func boolValue(value any) bool {
	b, _ := value.(bool)
	return b
}

func optionalFloat(value any) *float64 {
	if f, ok := toFloat(value); ok {
		return &f
	}
	return nil
}

func intValue(value any) int {
	if f, ok := toFloat(value); ok {
		return int(f)
	}
	return 0
}

func optionalInt(value any) *int64 {
	if f, ok := toFloat(value); ok {
		n := int64(f)
		return &n
	}
	return nil
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}

var _ fingerprintdomain.Repository = (*IdentificationRepository)(nil)
