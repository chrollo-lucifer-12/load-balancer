package cb

import (
	"testing"
)

func BenchmarkCircuitBreakerCanPass(b *testing.B) {
	cb := NewCircuitBreaker()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.CanPass()
	}
}

func BenchmarkCircuitBreakerOnSuccessClosed(b *testing.B) {
	cb := NewCircuitBreaker()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.OnSuccess()
	}
}

func BenchmarkCircuitBreakerOnFailureClosed(b *testing.B) {
	cb := NewCircuitBreaker()

	// Prevent the circuit from opening during the benchmark.
	cb.errorThreshold = 2.0

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.OnFailure()
	}
}

func BenchmarkCircuitBreakerCanPassConcurrent(b *testing.B) {
	cb := NewCircuitBreaker()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.CanPass()
		}
	})
}

func BenchmarkCircuitBreakerOnSuccessConcurrent(b *testing.B) {
	cb := NewCircuitBreaker()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.OnSuccess()
		}
	})
}

func BenchmarkCircuitBreakerOnFailureConcurrent(b *testing.B) {
	cb := NewCircuitBreaker()

	// Prevent the circuit from opening.
	cb.errorThreshold = 2.0

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.OnFailure()
		}
	})
}
