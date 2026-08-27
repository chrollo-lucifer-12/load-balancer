package slidingwindow

import (
	"sync"
	"time"
)

type Bucket struct {
	Total  int64
	Failed int64
}

type RollingWindow struct {
	mu         sync.Mutex
	buckets    map[int64]*Bucket
	windowSize int64
}

func NewRollingWindow(windowSizeSeconds int64) *RollingWindow {
	return &RollingWindow{
		buckets:    make(map[int64]*Bucket),
		windowSize: windowSizeSeconds,
	}
}

func (rw *RollingWindow) FailureCount() int64 {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	now := time.Now().Unix()
	rw.cleanup(now)

	var failed int64

	for _, bucket := range rw.buckets {
		failed += bucket.Failed
	}

	return failed
}

func (rw *RollingWindow) Record(isFailure bool) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	now := time.Now().Unix()
	rw.cleanup(now)

	if _, exists := rw.buckets[now]; !exists {
		rw.buckets[now] = &Bucket{}
	}

	rw.buckets[now].Total++
	if isFailure {
		rw.buckets[now].Failed++
	}
}

func (rw *RollingWindow) ErrorRate() (float64, int64) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	now := time.Now().Unix()
	rw.cleanup(now)

	var total, failed int64

	for _, bucket := range rw.buckets {
		total += bucket.Total
		failed += bucket.Failed
	}

	if total == 0 {
		return 0.00, 0
	}

	return float64(failed) / float64(total), total
}

func (rw *RollingWindow) cleanup(now int64) {
	boundary := now - rw.windowSize

	for timestamp := range rw.buckets {
		if timestamp <= boundary {
			delete(rw.buckets, timestamp)
		}
	}
}
