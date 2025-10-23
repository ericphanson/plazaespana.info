package fetch

import (
	"net/url"
	"sync"
	"time"
)

// RequestThrottle enforces minimum delays between requests to the same host.
type RequestThrottle struct {
	minDelay time.Duration
	lastReq  map[string]time.Time
	mu       sync.Mutex
}

// NewRequestThrottle creates a throttle with the given minimum delay.
func NewRequestThrottle(minDelay time.Duration) *RequestThrottle {
	return &RequestThrottle{
		minDelay: minDelay,
		lastReq:  make(map[string]time.Time),
	}
}

// Wait blocks until enough time has passed since the last request to this host.
// Returns the actual delay waited.
func (r *RequestThrottle) Wait(targetURL string) (time.Duration, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return 0, err
	}

	host := u.Host

	r.mu.Lock()
	defer r.mu.Unlock()

	lastReq, exists := r.lastReq[host]
	if !exists {
		// First request to this host
		r.lastReq[host] = time.Now()
		return 0, nil
	}

	elapsed := time.Since(lastReq)
	if elapsed < r.minDelay {
		waitTime := r.minDelay - elapsed
		time.Sleep(waitTime)
		r.lastReq[host] = time.Now()
		return waitTime, nil
	}

	r.lastReq[host] = time.Now()
	return 0, nil
}
