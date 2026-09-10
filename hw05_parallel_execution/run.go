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
	var errs int32
	c := make(chan Task, n)
	limit := int32(m)

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range c {
				if atomic.LoadInt32(&errs) >= limit {
					return
				}
				if t() != nil {
					atomic.AddInt32(&errs, 1)
				}
			}
		}()
	}

	for _, t := range tasks {
		if atomic.LoadInt32(&errs) >= limit {
			break
		}
		c <- t
	}

	close(c)
	wg.Wait()

	if atomic.LoadInt32(&errs) >= limit {
		return ErrErrorsLimitExceeded
	}

	return nil
}
