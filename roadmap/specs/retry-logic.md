# TH-11: Retry Logic & Error Recovery

## Overview

Implement automatic retry with exponential backoff for failed operations.

## Technical Requirements

```go
type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay time.Duration
    Multiplier float64
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    delay := cfg.InitialDelay
    for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        if attempt == cfg.MaxAttempts {
            return err
        }
        time.Sleep(delay)
        delay = time.Duration(float64(delay) * cfg.Multiplier)
        if delay > cfg.MaxDelay {
            delay = cfg.MaxDelay
        }
    }
    return errors.New("max retries exceeded")
}
```

## Acceptance Criteria

- [ ] Exponential backoff implemented
- [ ] Max retry limits
- [ ] Configurable delays
- [ ] Circuit breaker for repeated failures

## Estimated Complexity

Low-Medium
