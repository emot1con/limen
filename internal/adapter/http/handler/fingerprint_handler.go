package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"limen/internal/domain"
	fingerprintdomain "limen/internal/domain/fingerprint"
	fingerprintuc "limen/internal/usecase/fingerprint"
)

const (
	maxFingerprintBody = 64 * 1024
	maxJSONBody        = 100 * 1024
)

type FingerprintHandler struct {
	fingerprints *fingerprintuc.UseCase
}

func NewFingerprintHandler(fingerprints *fingerprintuc.UseCase) *FingerprintHandler {
	return &FingerprintHandler{fingerprints: fingerprints}
}

func (h *FingerprintHandler) Identify(c *gin.Context) {
	template, ok := readRawFingerprint(c, "Body harus berupa raw fingerprint binary.")
	if !ok {
		return
	}
	if !validFingerprintSize(c, len(template)) {
		return
	}
	response, err := h.fingerprints.Identify(c.Request.Context(), template, correlationID(c))
	writeGatewayResponse(c, response, err)
}

func (h *FingerprintHandler) Enroll(c *gin.Context) {
	var req struct {
		ExternalUserID string `json:"externalUserId"`
		Template       string `json:"template"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBody)
	if err := c.ShouldBindJSON(&req); err != nil {
		if isBodyTooLarge(err) {
			writeRequestTooLarge(c)
			return
		}
		writeGatewayMessage(c, http.StatusBadRequest, "externalUserId wajib diisi.")
		return
	}
	if strings.TrimSpace(req.ExternalUserID) == "" {
		writeGatewayMessage(c, http.StatusBadRequest, "externalUserId wajib diisi.")
		return
	}
	if strings.TrimSpace(req.Template) == "" {
		writeGatewayMessage(c, http.StatusBadRequest, "template (base64) wajib diisi.")
		return
	}
	response, err := h.fingerprints.Enroll(c.Request.Context(), req.ExternalUserID, req.Template, correlationID(c))
	writeGatewayResponse(c, response, err)
}

func (h *FingerprintHandler) EnrollRaw(c *gin.Context) {
	externalUserID := c.Query("externalUserId")
	if strings.TrimSpace(externalUserID) == "" {
		writeGatewayMessage(c, http.StatusBadRequest, "Query ?externalUserId wajib diisi.")
		return
	}
	template, ok := readRawFingerprint(c, "Body harus berupa raw fingerprint binary (application/octet-stream).")
	if !ok {
		return
	}
	if !validFingerprintSize(c, len(template)) {
		return
	}
	response, err := h.fingerprints.EnrollRaw(c.Request.Context(), externalUserID, template, correlationID(c))
	writeGatewayResponse(c, response, err)
}

func (h *FingerprintHandler) Latest(c *gin.Context) {
	result, _ := h.fingerprints.Latest(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *FingerprintHandler) Stream(c *gin.Context) {
	// SSE is long-lived: clear the server write deadline for this connection.
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}

	_, _ = io.WriteString(c.Writer, ": connected\n\n")
	flush()

	if current, ok := h.fingerprints.Store().Latest(); ok {
		writeSSE(c.Writer, current)
		flush()
	}

	events, unsubscribe := h.fingerprints.Store().Subscribe()
	defer unsubscribe()

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-keepAlive.C:
			_, _ = io.WriteString(c.Writer, ": keepalive\n\n")
			flush()
		case result, open := <-events:
			if !open {
				return
			}
			writeSSE(c.Writer, result)
			flush()
		}
	}
}

func readRawFingerprint(c *gin.Context, nonBinaryMessage string) ([]byte, bool) {
	if c.ContentType() != "application/octet-stream" {
		writeGatewayMessage(c, http.StatusBadRequest, nonBinaryMessage)
		return nil, false
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxFingerprintBody))
	if err != nil {
		if isBodyTooLarge(err) {
			writeRequestTooLarge(c)
			return nil, false
		}
		writeGatewayMessage(c, http.StatusBadRequest, nonBinaryMessage)
		return nil, false
	}
	return body, true
}

func validFingerprintSize(c *gin.Context, size int) bool {
	if size != fingerprintuc.FingerprintSize {
		writeGatewayMessage(c, http.StatusBadRequest, fmt.Sprintf("Ukuran fingerprint tidak sesuai. Expected: %d, Actual: %d", fingerprintuc.FingerprintSize, size))
		return false
	}
	return true
}

func writeGatewayResponse(c *gin.Context, response fingerprintdomain.Response, err error) {
	if err != nil {
		if errors.Is(err, domain.ErrInvalid) {
			writeGatewayMessage(c, http.StatusBadRequest, "Permintaan tidak valid.")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Tidak dapat terhubung ke fingerprint service.", "error": err.Error()})
		return
	}
	if response.Status == 0 {
		response.Status = http.StatusOK
	}
	if response.Body == nil {
		c.Status(response.Status)
		return
	}
	c.JSON(response.Status, response.Body)
}

func writeGatewayMessage(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message})
}

func writeRequestTooLarge(c *gin.Context) {
	c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "error": gin.H{"code": "REQUEST_TOO_LARGE", "message": "Request body is too large."}})
}

func isBodyTooLarge(err error) bool {
	var maxBytes *http.MaxBytesError
	return errors.As(err, &maxBytes)
}

func correlationID(c *gin.Context) string {
	for _, header := range []string{"X-Request-Id", "X-Correlation-Id", "X-Trace-Id"} {
		if value := strings.TrimSpace(c.GetHeader(header)); value != "" {
			return value
		}
	}
	return ""
}

func writeSSE(w io.Writer, result map[string]any) {
	payload, err := json.Marshal(result)
	if err != nil {
		return
	}
	_, _ = io.WriteString(w, "data: ")
	_, _ = w.Write(payload)
	_, _ = io.WriteString(w, "\n\n")
}
