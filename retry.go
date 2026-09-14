package onlyoffice

import (
	"context"
	"regexp"
	"time"
)

// RetryPolicy controls deterministic retries against OnlyOffice: fixed linear
// backoff without jitter, so repeated runs wait exactly the same schedule.
// OnlyOffice throttles bulk reads/writes with 429 (and occasional 502/503/504
// from openresty), so every bulk tool routes API calls through DoRetry.
type RetryPolicy struct {
	Attempts int           // total attempts, including the first try
	Base     time.Duration // wait before retry N is N*Base
	Max      time.Duration // per-wait cap
}

// DefaultRetryPolicy retries up to 5 times with 1s, 2s, 3s, 4s waits.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Attempts: 5, Base: time.Second, Max: 30 * time.Second}
}

var transientRe = regexp.MustCompile(`:\s*(429|502|503|504)\b`)

// Transient reports whether err looks like a transient OnlyOffice answer
// (an HTTP 429/502/503/504 surfaced as "...: <code> ...").
func Transient(err error) bool {
	if err == nil {
		return false
	}
	return transientRe.MatchString(err.Error())
}

// DoRetry runs fn until it succeeds, fails non-transiently, or attempts run
// out. Waits are deterministic: N*Base capped at Max, no jitter.
func DoRetry(ctx context.Context, p RetryPolicy, fn func() error) error {
	if p.Attempts < 1 {
		p.Attempts = 1
	}
	var err error
	for attempt := 1; attempt <= p.Attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err = fn(); err == nil || !Transient(err) || attempt == p.Attempts {
			return err
		}
		wait := time.Duration(attempt) * p.Base
		if wait > p.Max {
			wait = p.Max
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}
