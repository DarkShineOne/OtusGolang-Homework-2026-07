package hw05parallelexecution

import (
	"errors"
	"sync"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(tasks []Task, n, m int) error {
	if m <= 0 {
		return ErrErrorsLimitExceeded
	}

	wg := new(sync.WaitGroup)
	c := make(chan Task, n)
	mu := new(sync.Mutex)
	counter := 0

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for task := range c {
				mu.Lock()
				exceeded := counter > m
				mu.Unlock()

				if exceeded {
					break
				}

				if err := task(); err != nil {
					mu.Lock()
					counter++
					mu.Unlock()
				}
			}
		}()
	}

	for _, task := range tasks {
		mu.Lock()
		exceeded := counter >= m
		mu.Unlock()
		if exceeded {
			break
		}

		c <- task
	}

	close(c)

	wg.Wait()

	if counter >= m {
		return ErrErrorsLimitExceeded
	}

	return nil
}
