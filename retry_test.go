package onlyoffice

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTransient(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("boom"), false},
		{errors.New("GET /api/2.0/files/1.json: 429 Too Many Requests"), true},
		{errors.New("auth: 502 bad gateway"), true},
		{errors.New("POST /x: 503 service unavailable"), true},
		{errors.New("PUT /x: 504 gateway timeout"), true},
		{errors.New("GET /x: 404 not found"), false},
		{errors.New("GET /x: 4290 not a code"), false},
	}
	for _, c := range cases {
		if got := Transient(c.err); got != c.want {
			t.Errorf("Transient(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestDoRetryRetriesTransientThenSucceeds(t *testing.T) {
	p := RetryPolicy{Attempts: 5, Base: time.Millisecond, Max: 5 * time.Millisecond}
	attempts := 0
	err := DoRetry(context.Background(), p, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("GET /x: 429 Too Many Requests")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("DoRetry: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestDoRetryStopsOnPermanentError(t *testing.T) {
	p := RetryPolicy{Attempts: 5, Base: time.Millisecond, Max: time.Millisecond}
	attempts := 0
	err := DoRetry(context.Background(), p, func() error {
		attempts++
		return errors.New("GET /x: 404 not found")
	})
	if err == nil {
		t.Fatal("want error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestDoRetryExhaustsAttemptsOnPersistentTransient(t *testing.T) {
	p := RetryPolicy{Attempts: 3, Base: time.Millisecond, Max: time.Millisecond}
	attempts := 0
	err := DoRetry(context.Background(), p, func() error {
		attempts++
		return errors.New("GET /x: 429 Too Many Requests")
	})
	if !Transient(err) {
		t.Fatalf("want transient error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestDoRetryHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := DoRetry(ctx, RetryPolicy{Attempts: 3, Base: time.Millisecond, Max: time.Millisecond}, func() error {
		return errors.New("GET /x: 429 Too Many Requests")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
