package server

import (
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
	rate    float64
	burst   float64
}

func NewRateLimiter(ratePerSecond int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*bucket),
		rate:    float64(ratePerSecond),
		burst:   float64(ratePerSecond * 2),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.clients[ip]
	now := time.Now()

	if !exists {
		rl.clients[ip] = &bucket{tokens: rl.burst - 1, lastCheck: now}
		return true
	}

	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastCheck = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for ip, b := range rl.clients {
			if b.lastCheck.Before(cutoff) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimitExceeded(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
}
