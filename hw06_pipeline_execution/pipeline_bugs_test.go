package hw06pipelineexecution

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func passthroughStage(in In) Out {
	out := make(Bi)
	go func() {
		defer close(out)
		for v := range in {
			out <- v
		}
	}()
	return out
}

// BUG 1: after cancellation forward() drains src until it is CLOSED
// (pipeline.go:17-22). If the upstream is never closed - or is infinite -
// that goroutine never exits: one goroutine leaks per pipeline, and with an
// infinite source it spins at 100% CPU forever.
func TestForwardLeaksGoroutineOnCancel(t *testing.T) {
	const pipelines = 20

	runtime.GC()
	before := runtime.NumGoroutine()

	for i := 0; i < pipelines; i++ {
		in := make(Bi, 1) // deliberately never closed
		done := make(Bi)
		in <- 1

		go func(d Bi) { time.Sleep(5 * time.Millisecond); close(d) }(done)
		for range ExecutePipeline(in, done, passthroughStage) {
		}
	}

	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	leaked := runtime.NumGoroutine() - before

	t.Logf("goroutines: before=%d after=%d, leaked=%d for %d pipelines",
		before, runtime.NumGoroutine(), leaked, pipelines)
	require.Less(t, leaked, 2, "forward() must not outlive the pipeline")
}

// BUG 2: when a value is in flight and done is already closed, the inner
// select at pipeline.go:32-37 has BOTH cases ready and Go picks one at
// random, so a value can still be emitted after the cancel.
func TestNoValueDeliveredAfterDoneClosed(t *testing.T) {
	const runs = 100

	escaped, total := 0, 0
	for i := 0; i < runs; i++ {
		in := make(Bi)
		done := make(Bi)
		go func() {
			for j := 0; ; j++ { // infinite source: always a value in flight
				in <- j
			}
		}()

		out := ExecutePipeline(in, done, passthroughStage, passthroughStage, passthroughStage)

		var cancelled atomic.Bool
		go func() {
			time.Sleep(10 * time.Millisecond)
			close(done)
			cancelled.Store(true)
		}()

		for range out { // consumer reads continuously
			total++
			if cancelled.Load() {
				escaped++
			}
		}
	}

	t.Logf("%d values delivered after done was closed (of %d total, %d pipelines)",
		escaped, total, runs)
	require.Zero(t, escaped, "no value may leave the pipeline after cancel")
}
