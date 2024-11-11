package utils

import (
	"sync"
	"time"
)

func Throttle[T any](fn func(arg T) error, interval time.Duration) func(arg T) error {
	var mu sync.Mutex
	var lastCall time.Time

	return func(arg T) error {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if now.Sub(lastCall) >= interval {
			err := fn(arg)
			if err != nil {
				return err
			}

			lastCall = now
			return nil
		}

		return nil
	}
}
