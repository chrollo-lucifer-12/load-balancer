package metrics

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkRollingWindow_RecordSuccess(b *testing.B) {
	rw := NewRollingWindow(60)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rw.Record(false)
	}
}

func BenchmarkRollingWindow_RecordFailure(b *testing.B) {
	rw := NewRollingWindow(60)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rw.Record(true)
	}
}

func BenchmarkRollingWindow_FailureCount(b *testing.B) {
	rw := NewRollingWindow(60)

	for i := 0; i < 10000; i++ {
		rw.Record(i%2 == 0)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rw.FailureCount()
	}
}

func BenchmarkRollingWindow_ErrorRate(b *testing.B) {
	rw := NewRollingWindow(60)

	for i := 0; i < 10000; i++ {
		rw.Record(i%2 == 0)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rw.ErrorRate()
	}
}

func BenchmarkRollingWindow_RecordParallel(b *testing.B) {
	rw := NewRollingWindow(60)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rw.Record(false)
		}
	})
}

func BenchmarkRollingWindow_ErrorRateParallel(b *testing.B) {
	rw := NewRollingWindow(60)

	for i := 0; i < 10000; i++ {
		rw.Record(i%2 == 0)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rw.ErrorRate()
		}
	})
}

func BenchmarkRollingWindow_MixedParallel(b *testing.B) {
	rw := NewRollingWindow(60)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0

		for pb.Next() {
			switch i % 10 {
			case 0:
				rw.ErrorRate()

			case 1:
				rw.FailureCount()

			default:
				rw.Record(i%3 == 0)
			}

			i++
		}
	})
}

func TestRollingWindow_RecordAndFailureCount(t *testing.T) {
	rw := NewRollingWindow(60)

	rw.Record(false)
	rw.Record(true)
	rw.Record(true)
	rw.Record(false)

	got := rw.FailureCount()
	want := int64(2)

	if got != want {
		t.Fatalf("FailureCount() = %d, want %d", got, want)
	}
}

func TestRollingWindow_ErrorRate(t *testing.T) {
	rw := NewRollingWindow(60)

	// 3 failures, 2 successes
	rw.Record(true)
	rw.Record(true)
	rw.Record(true)
	rw.Record(false)
	rw.Record(false)

	rate, total := rw.ErrorRate()

	wantRate := 0.6
	wantTotal := int64(5)

	if rate != wantRate {
		t.Fatalf("ErrorRate() = %f, want %f", rate, wantRate)
	}

	if total != wantTotal {
		t.Fatalf("total = %d, want %d", total, wantTotal)
	}
}

func TestRollingWindow_Empty(t *testing.T) {
	rw := NewRollingWindow(60)

	failed := rw.FailureCount()

	if failed != 0 {
		t.Fatalf("FailureCount() = %d, want 0", failed)
	}

	rate, total := rw.ErrorRate()

	if rate != 0 {
		t.Fatalf("ErrorRate() = %f, want 0", rate)
	}

	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
}

func TestRollingWindow_Cleanup(t *testing.T) {
	rw := NewRollingWindow(2)

	now := time.Now().Unix()

	// Expired bucket.
	rw.buckets[now-3] = &MetricsBucket{
		Total:  10,
		Failed: 5,
	}

	// Still valid bucket.
	rw.buckets[now-1] = &MetricsBucket{
		Total:  4,
		Failed: 2,
	}

	rate, total := rw.ErrorRate()

	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}

	if rate != 0.5 {
		t.Fatalf("rate = %f, want 0.5", rate)
	}

	if _, ok := rw.buckets[now-3]; ok {
		t.Fatal("expired bucket was not removed")
	}
}

func TestRollingWindow_ConcurrentRecord(t *testing.T) {
	rw := NewRollingWindow(60)

	const goroutines = 100
	const recordsPerGoroutine = 100

	var wg sync.WaitGroup

	for range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range recordsPerGoroutine {
				rw.Record(true)
			}
		}()
	}

	wg.Wait()

	want := int64(goroutines * recordsPerGoroutine)
	got := rw.FailureCount()

	if got != want {
		t.Fatalf("FailureCount() = %d, want %d", got, want)
	}
}
