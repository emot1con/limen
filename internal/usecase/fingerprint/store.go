package fingerprint

import (
	"sync"
	"time"
)

const subscriberBuffer = 8

// Store keeps the latest identification result in memory and fans it out to
// Server-Sent Events subscribers.
type Store struct {
	mu          sync.Mutex
	latest      map[string]any
	scanCounter int
	subscribers map[chan map[string]any]struct{}
	now         func() time.Time
}

func NewStore() *Store {
	return &Store{subscribers: make(map[chan map[string]any]struct{}), now: time.Now}
}

// Record stamps the result with scanId and timestamp, stores it as the latest
// value, and pushes it to every subscriber.
func (s *Store) Record(result map[string]any) map[string]any {
	s.mu.Lock()
	s.scanCounter++
	stamped := make(map[string]any, len(result)+2)
	for key, value := range result {
		stamped[key] = value
	}
	stamped["scanId"] = s.scanCounter
	stamped["timestamp"] = s.now().UTC().Format(time.RFC3339)
	s.latest = stamped
	subscribers := make([]chan map[string]any, 0, len(s.subscribers))
	for ch := range s.subscribers {
		subscribers = append(subscribers, ch)
	}
	s.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- stamped:
		default:
		}
	}
	return stamped
}

func (s *Store) Latest() (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest == nil {
		return nil, false
	}
	return s.latest, true
}

func (s *Store) Subscribe() (<-chan map[string]any, func()) {
	ch := make(chan map[string]any, subscriberBuffer)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		s.mu.Unlock()
	}
}
