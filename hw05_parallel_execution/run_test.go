package hw05parallelexecution

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRun(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("if were errors in first M tasks, than finished not more N+M tasks", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			err := fmt.Errorf("error from task %d", i)
			tasks = append(tasks, func() error {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				atomic.AddInt32(&runTasksCount, 1)
				return err
			})
		}

		workersCount := 10
		maxErrorsCount := 23
		err := Run(tasks, workersCount, maxErrorsCount)

		require.Truef(t, errors.Is(err, ErrErrorsLimitExceeded), "actual err - %v", err)
		require.LessOrEqual(t, runTasksCount, int32(workersCount+maxErrorsCount), "extra tasks were started")
	})

	t.Run("tasks without errors", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)
		require.NoError(t, err)

		require.Equal(t, int32(tasksCount), runTasksCount, "not all tasks were completed")
		require.LessOrEqual(t, int64(elapsedTime), int64(sumTime/2), "tasks were run sequentially?")
	})

	t.Run("task zero max error", func(t *testing.T) {
		tasksCount := 1
		tasks := make([]Task, 0, tasksCount)

		for i := 0; i < tasksCount; i++ {
			tasks = append(tasks, func() error {
				return nil
			})
		}

		workersCount := 1
		maxErrorsCount := 0

		err := Run(tasks, workersCount, maxErrorsCount)
		require.Error(t, err)
	})

	t.Run("workers exceeds tasks", func(t *testing.T) {
		tasksCount := 5
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 12
		maxErrorsCount := 1

		err := Run(tasks, workersCount, maxErrorsCount)

		require.NoError(t, err)
		require.Equal(t, int32(tasksCount), runTasksCount, "not all tasks were completed")
	})

	t.Run("concurrency without time.Sleep", func(t *testing.T) {
		workersCount := 5
		tasksCount := 20
		maxErrorsCount := 1

		runningTasks := 0
		mu := &sync.Mutex{}
		startCh := make(chan struct{})

		tasks := make([]Task, 0, tasksCount)
		for i := 0; i < tasksCount; i++ {
			tasks = append(tasks, func() error {
				mu.Lock()
				runningTasks++
				mu.Unlock()

				<-startCh

				return nil
			})
		}

		var runErr error
		done := make(chan struct{})

		go func() {
			defer close(done)
			runErr = Run(tasks, workersCount, maxErrorsCount)
		}()

		require.Eventually(t, func() bool {
			return runningTasks == workersCount
		}, time.Second, time.Millisecond, "tasks are not running concurrently")

		close(startCh)

		<-done

		require.NoError(t, runErr)
	})
}
