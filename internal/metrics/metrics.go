package metrics

import "sync/atomic"

var MetricsRecord *Metrics

type Metrics struct {
	RequestsTotal    atomic.Int64
	RequestsInFlight atomic.Int64
	RequestsFailed   atomic.Int64
}

func NewMetrics() {
	m := Metrics{}

	m.RequestsTotal.Store(0)
	m.RequestsFailed.Store(0)
	m.RequestsInFlight.Store(0)

	MetricsRecord = &m
}
