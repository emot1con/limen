package fingerprint

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	fingerprintdomain "limen/internal/domain/fingerprint"
)

const upstreamTimeout = 10 * time.Second

// Client is an HTTP adapter for the upstream fingerprint service.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		http:    &http.Client{Timeout: upstreamTimeout},
	}
}

func (c *Client) Identify(ctx context.Context, template []byte, requestID string) (fingerprintdomain.Response, error) {
	req, err := c.newRequest(ctx, c.baseURL+"/api/v1/fingerprints/identify", "application/octet-stream", bytes.NewReader(template), requestID)
	if err != nil {
		return fingerprintdomain.Response{}, err
	}
	return c.do(req)
}

func (c *Client) Enroll(ctx context.Context, externalUserID, template, requestID string) (fingerprintdomain.Response, error) {
	payload, err := json.Marshal(map[string]string{"externalUserId": externalUserID, "template": template})
	if err != nil {
		return fingerprintdomain.Response{}, err
	}
	req, err := c.newRequest(ctx, c.baseURL+"/api/v1/fingerprints/enroll", "application/json", bytes.NewReader(payload), requestID)
	if err != nil {
		return fingerprintdomain.Response{}, err
	}
	return c.do(req)
}

func (c *Client) EnrollRaw(ctx context.Context, externalUserID string, template []byte, requestID string) (fingerprintdomain.Response, error) {
	endpoint := c.baseURL + "/api/v1/fingerprints/enroll/raw?externalUserId=" + url.QueryEscape(externalUserID)
	req, err := c.newRequest(ctx, endpoint, "application/octet-stream", bytes.NewReader(template), requestID)
	if err != nil {
		return fingerprintdomain.Response{}, err
	}
	return c.do(req)
}

func (c *Client) newRequest(ctx context.Context, endpoint, contentType string, body io.Reader, requestID string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	return req, nil
}

// do executes the request. Transport failures return an error; non-2xx
// responses are returned as a Response so the caller can relay them verbatim.
func (c *Client) do(req *http.Request) (fingerprintdomain.Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return fingerprintdomain.Response{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fingerprintdomain.Response{}, err
	}
	var body any
	if len(bytes.TrimSpace(raw)) > 0 {
		if unmarshalErr := json.Unmarshal(raw, &body); unmarshalErr != nil {
			body = string(raw)
		}
	}
	return fingerprintdomain.Response{Status: resp.StatusCode, Body: body}, nil
}

var _ fingerprintdomain.Gateway = (*Client)(nil)
