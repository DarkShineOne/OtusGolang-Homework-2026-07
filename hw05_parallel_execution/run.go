package hw05parallelexecution

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

var ErrAvailableWorkers = errors.New("no available workers")

type Task func() error

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(tasks []Task, n, m int) error {
	if m <= 0 {
		return ErrErrorsLimitExceeded
	}

	if n <= 0 {
		return ErrAvailableWorkers
	}

	var wg sync.WaitGroup
	var errs atomic.Int64
	c := make(chan Task, n)
	limit := int64(n)

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range c {
				if errs.Load() >= limit {
					return
				}
				if t() != nil {
					errs.Add(1)
				}
			}
		}()
	}

	for _, t := range tasks {
		if errs.Load() >= limit {
			break
		}
		c <- t
	}

	close(c)
	wg.Wait()

	if errs.Load() >= limit {
		return ErrErrorsLimitExceeded
	}

	return nil
}
