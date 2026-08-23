package rl

import (
	"sync"
	"time"
)

var Rl *RateLimiter

type TokenBukcet struct {
	tokens int64

	lastRefill time.Time
}

type RateLimiter struct {
	rate     int64
	capacity int64

	mu      sync.Mutex
	buckets map[string]*TokenBukcet
}

func NewRateLimiter(rate, tokens, capacity int64) {
	Rl = &RateLimiter{
		rate:     rate,
		capacity: tokens,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	tb, ok := rl.buckets[key]

	if !ok {
		tb = &TokenBukcet{
			tokens:     rl.capacity,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = tb
	}

	tb.tokens = min(rl.capacity, int64(time.Since(tb.lastRefill)/time.Second)*rl.rate)

	tb.lastRefill = time.Now()

	if tb.tokens <= 0 {
		return false
	}

	tb.tokens--

	return true
}
