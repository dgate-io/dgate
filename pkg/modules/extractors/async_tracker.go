package extractors

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/dop251/goja"
)

var _ goja.AsyncContextTracker = &asyncTracker{}

type asyncTracker struct {
	count    atomic.Int32
	exitChan chan int32
}

type TrackerEvent int

const (
	Exited TrackerEvent = iota
	Resumed
)

func newAsyncTracker() *asyncTracker {
	return &asyncTracker{
		count:    atomic.Int32{},
		exitChan: make(chan int32, 128),
	}
}

// Exited is called when an async function is done
func (t *asyncTracker) Exited() {
	t.exitChan <- t.count.Add(-1)
}

// Grab is called when an async function is scheduled
func (t *asyncTracker) Grab() any {
	t.exitChan <- t.count.Add(1)
	return nil
}

// Resumed is called when an async function is executed (ignore)
func (t *asyncTracker) Resumed(any) {
	t.exitChan <- t.count.Load()
}

func (t *asyncTracker) waitTimeout(
	ctx context.Context, doneFn func() bool,
) error {
	if doneFn() {
		return nil
	} else if t.count.Load() == 0 {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("async tracker: %s", ctx.Err())
		case numLeft := <-t.exitChan:
			if numLeft == 0 || doneFn() {
				return nil
			}
		}
	}
}
