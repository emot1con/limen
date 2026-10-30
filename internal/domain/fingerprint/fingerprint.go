package fingerprint

import "context"

// Response is a decoded response from the upstream fingerprint service.
// Body holds the JSON-decoded payload so that fields unknown to this gateway
// are relayed back to clients unchanged.
type Response struct {
	Status int
	Body   any
}

// Gateway forwards biometric operations to the upstream fingerprint service.
type Gateway interface {
	Identify(ctx context.Context, template []byte, requestID string) (Response, error)
	Enroll(ctx context.Context, externalUserID, template, requestID string) (Response, error)
	EnrollRaw(ctx context.Context, externalUserID string, template []byte, requestID string) (Response, error)
}

// Repository persists and retrieves identification audit records.
type Repository interface {
	Save(ctx context.Context, result map[string]any) error
	Latest(ctx context.Context) (map[string]any, error)
}
