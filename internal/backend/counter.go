package backend

import (
	"sync/atomic"

	"github.com/lb/pkg/slidingwindow"
)

type Counter struct {
	passive *slidingwindow.RollingWindow

	active atomic.Int64
}

func NewCounter() *Counter {
	c := &Counter{}

	c.passive = slidingwindow.NewRollingWindow(10)

	return c
}
