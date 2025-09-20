package dlq

import (
	"errors"
	"time"
)

type Option func(*DLQ)

func MaxAttemptsCount(count int) Option {
	return func(d *DLQ) {
		d.MaxAttempts = count
	}
}

func BaseRetryDelay(delay time.Duration) Option {
	return func(d *DLQ) {
		d.baseRetryDelay = delay
	}
}

func MaxRetryDelay(delay time.Duration) Option {
	return func(d *DLQ) {
		d.maxRetryDelay = delay
	}
}

func (d *DLQ) validate() error {
	if d.MaxAttempts <= 0 {
		return errors.New("invalid maxAttempts: must be > 0")
	}

	if d.baseRetryDelay <= 0 {
		return errors.New("invalid baseRetryDelay: must be > 0")
	}

	if d.maxRetryDelay <= 0 {
		return errors.New("invalid maxRetryDelay: must be > 0")
	}

	if d.baseRetryDelay > d.maxRetryDelay {
		return errors.New("baseRetryDelay cannot exceed maxRetryDelay")
	}
	return nil
}
