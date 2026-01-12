package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

type RateLimiter struct {
	globalLimiter *rate.Limiter
	userLimiters  map[string]*rate.Limiter
	mu            sync.RWMutex
	perUserRate   rate.Limit
	perUserBurst  int
}

func NewRateLimiter(globalRPM, globalBurst, perUserRPM, perUserBurst int) *RateLimiter {
	return &RateLimiter{
		globalLimiter: rate.NewLimiter(rate.Limit(globalRPM)/60, globalBurst),
		userLimiters:  make(map[string]*rate.Limiter),
		perUserRate:   rate.Limit(perUserRPM) / 60,
		perUserBurst:  perUserBurst,
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, userID string) error {
	// Check global rate limit
	if !rl.globalLimiter.Allow() {
		return ErrRateLimitExceeded
	}

	// Check per-user rate limit
	limiter := rl.getUserLimiter(userID)
	if !limiter.Allow() {
		return ErrRateLimitExceeded
	}

	return nil
}

func (rl *RateLimiter) getUserLimiter(userID string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.userLimiters[userID]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.userLimiters[userID]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.perUserRate, rl.perUserBurst)
	rl.userLimiters[userID] = limiter

	return limiter
}

// Cleanup old user limiters periodically
func (rl *RateLimiter) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Simple cleanup: clear all limiters periodically
			// In production, you might want more sophisticated cleanup
			if len(rl.userLimiters) > 10000 {
				rl.userLimiters = make(map[string]*rate.Limiter)
			}
			rl.mu.Unlock()
		}
	}()
}
