package rl

import (
	"sync"
	"time"
)

type RateLimiterType string

const (
	TokenBucket RateLimiterType = "token_bucket"
)

var Rl RateLimiter

type TokenBukcet struct {
	tokens int64

	lastRefill time.Time
}

type TokenBucketLimiter struct {
	rate     int64
	capacity int64

	mu      sync.Mutex
	buckets map[string]*TokenBukcet
}

type RateLimiter interface {
	Allow(key string) bool
}

func NewTokenBucketLimiter(rate, capacity int64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:     rate,
		capacity: capacity,
	}
}

func (rl *TokenBucketLimiter) Allow(key string) bool {
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
