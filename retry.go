package onlyoffice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RetryPolicy controls deterministic retries against OnlyOffice: exponential
// backoff without jitter, so repeated runs wait exactly the same schedule.
// OnlyOffice throttles bulk reads/writes with 429 (and occasional 502/503/504
// from openresty), so every bulk tool routes API calls through DoRetry.
type RetryPolicy struct {
	Attempts int           // total attempts, including the first try
	Base     time.Duration // wait before retry N is Base*2^(N-1)
	Max      time.Duration // per-wait cap; <=0 means no cap
}

// DefaultRetryPolicy builds the policy from the environment, falling back to
// 7 attempts, a 2s base and a 2m cap:
//
//   - OO_RETRY_ATTEMPTS (default 7)
//   - OO_RETRY_BASE     (duration, default 2s)
//   - OO_RETRY_MAX      (duration, default 2m)
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Attempts: envInt("OO_RETRY_ATTEMPTS", 7),
		Base:     envDuration("OO_RETRY_BASE", 2*time.Second),
		Max:      envDuration("OO_RETRY_MAX", 2*time.Minute),
	}
}

var transientRe = regexp.MustCompile(`:\s*(429|502|503|504)\b`)

// isTransientStatus reports whether an HTTP status is retriable at the edge.
func isTransientStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// TransientError is a typed transient answer from the HTTP layer. It carries
// the status code and, when present, the server's Retry-After delay so DoRetry
// can wait at least that long.
type TransientError struct {
	Code       int
	RetryAfter time.Duration
	Msg        string
}

func (e *TransientError) Error() string { return e.Msg }

// Transient reports whether err looks like a transient OnlyOffice answer: a
// *TransientError with a retriable code, or an error whose text carries an
// HTTP 429/502/503/504.
func Transient(err error) bool {
	if err == nil {
		return false
	}
	var te *TransientError
	if errors.As(err, &te) {
		return isTransientStatus(te.Code)
	}
	return transientRe.MatchString(err.Error())
}

// statusError wraps a non-2xx answer, tagging transient statuses so DoRetry
// recognises them and honours Retry-After.
func statusError(status int, retryAfter time.Duration, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if isTransientStatus(status) {
		return &TransientError{Code: status, RetryAfter: retryAfter, Msg: msg}
	}
	return errors.New(msg)
}

// retryAfterOf parses the Retry-After header of a response (integer seconds or
// an HTTP-date). Returns 0 when absent or malformed.
func retryAfterOf(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	return parseRetryAfter(resp.Header.Get("Retry-After"))
}

// parseRetryAfter parses a Retry-After value: delay-seconds (RFC 9110) or an
// HTTP-date. Zero, negative and malformed values yield 0.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// retryAfterOfError extracts Retry-After from a typed transient error.
func retryAfterOfError(err error) time.Duration {
	var te *TransientError
	if errors.As(err, &te) {
		return te.RetryAfter
	}
	return 0
}

// backoffDelay returns the deterministic wait before retry `attempt`
// (counting from 1): Base*2^(attempt-1), capped at Max.
func backoffDelay(p RetryPolicy, attempt int) time.Duration {
	if p.Base <= 0 {
		return 0
	}
	wait := p.Base
	for i := 1; i < attempt; i++ {
		if p.Max > 0 && wait >= p.Max {
			return p.Max
		}
		wait *= 2
	}
	if p.Max > 0 && wait > p.Max {
		wait = p.Max
	}
	return wait
}

// DoRetry runs fn until it succeeds, fails non-transiently, or attempts run
// out. Waits are deterministic: Base*2^(N-1) capped at Max, no jitter. A
// transient error's Retry-After wins when it is longer than the backoff, and
// every wait arms the process-wide cooldown gate so concurrent and sequential
// callers back off too.
func DoRetry(ctx context.Context, p RetryPolicy, fn func() error) error {
	if p.Attempts < 1 {
		p.Attempts = 1
	}
	var err error
	for attempt := 1; attempt <= p.Attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = fn()
		if err == nil || !Transient(err) || attempt == p.Attempts {
			return err
		}
		wait := backoffDelay(p, attempt)
		if ra := retryAfterOfError(err); ra > wait {
			wait = ra
		}
		if wait <= 0 {
			continue
		}
		globalCooldown.note(wait)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}

// retryRaw runs a transport attempt under the default transient-retry policy
// and returns its raw payload. All HTTP helpers and Query() go through it, so
// an openresty 429/502/503/504 is retried exactly like every bulk tool.
func retryRaw(ctx context.Context, fn func() (json.RawMessage, error)) (json.RawMessage, error) {
	var raw json.RawMessage
	err := DoRetry(ctx, DefaultRetryPolicy(), func() error {
		var e error
		raw, e = fn()
		return e
	})
	return raw, err
}
