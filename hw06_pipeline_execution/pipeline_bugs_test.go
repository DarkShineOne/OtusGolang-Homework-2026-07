package hw06pipelineexecution

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNoValueDeliveredAfterDoneClosed(t *testing.T) {
	const runs = 100

	passthroughStage := func(in In) Out {
		out := make(Bi)
		go func() {
			defer close(out)
			for v := range in {
				out <- v
			}
		}()
		return out
	}

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
