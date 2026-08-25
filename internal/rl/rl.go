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

func NewRateLimiter(rType RateLimiterType) RateLimiter {
	switch rType {
	case TokenBucket:
		return NewTokenBucketLimiter(5, 100)
	default:
		return nil
	}
}

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
		buckets:  make(map[string]*TokenBukcet),
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

	now := time.Now()

	elapsed := now.Sub(tb.lastRefill)

	newTokens := int64(elapsed/time.Second) * rl.rate

	if newTokens > 0 {
		tb.tokens = min(rl.capacity, tb.tokens+newTokens)
		tb.lastRefill = now
	}

	if tb.tokens <= 0 {
		return false
	}

	tb.tokens--

	return true
}
