package utils

import (
	"time"
)

func Delay[T any](fn func(arg T) error, wait time.Duration) func(arg T) error {
	return func(arg T) error {
		time.Sleep(wait)
		return fn(arg)
	}
}
