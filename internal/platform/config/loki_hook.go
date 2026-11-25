package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// LokiHook is a non-blocking logrus Hook sending logs to Grafana Loki push API.
type LokiHook struct {
	client     *http.Client
	url        string
	labels     map[string]string
	logChannel chan []byte
}

type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// NewLokiHook creates a new hook that pushes formatted JSON logs to Loki.
func NewLokiHook(lokiURL string, labels map[string]string) *LokiHook {
	pushURL := fmt.Sprintf("%s/loki/api/v1/push", lokiURL)
	h := &LokiHook{
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		url:        pushURL,
		labels:     labels,
		logChannel: make(chan []byte, 1000), // buffered to avoid blocking callers
	}

	go h.worker()
	return h
}

func (h *LokiHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LokiHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}

	select {
	case h.logChannel <- []byte(line):
	default:
		// Drop log if channel full to prevent blocking application
	}
	return nil
}

func (h *LokiHook) worker() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var batch [][]string

	sendBatch := func() {
		if len(batch) == 0 {
			return
		}

		payload := lokiPushRequest{
			Streams: []lokiStream{
				{
					Stream: h.labels,
					Values: batch,
				},
			},
		}

		body, err := json.Marshal(payload)
		if err == nil {
			req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(body))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				resp, err := h.client.Do(req)
				if err == nil {
					_ = resp.Body.Close()
				}
			}
		}

		batch = make([][]string, 0, 50)
	}

	for {
		select {
		case line, ok := <-h.logChannel:
			if !ok {
				sendBatch()
				return
			}
			tsNano := strconv.FormatInt(time.Now().UnixNano(), 10)
			batch = append(batch, []string{tsNano, string(line)})
			if len(batch) >= 50 {
				sendBatch()
			}
		case <-ticker.C:
			sendBatch()
		}
	}
}
