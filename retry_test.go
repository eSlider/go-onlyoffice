package onlyoffice

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
		{&TransientError{Code: 429, Msg: "throttled"}, true},
		{&TransientError{Code: 404, Msg: "missing"}, false},
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

func TestBackoffDelayExponentialAndCapped(t *testing.T) {
	p := RetryPolicy{Base: 100 * time.Millisecond, Max: 250 * time.Millisecond}
	want := []time.Duration{
		100 * time.Millisecond, // 2^0
		200 * time.Millisecond, // 2^1
		250 * time.Millisecond, // 2^2 capped
		250 * time.Millisecond,
		250 * time.Millisecond,
	}
	for i, w := range want {
		if got := backoffDelay(p, i+1); got != w {
			t.Errorf("backoffDelay(attempt=%d) = %v, want %v", i+1, got, w)
		}
	}
}

func TestDoRetryWaitsRetryAfterLongerThanBackoff(t *testing.T) {
	const ra = 60 * time.Millisecond
	p := RetryPolicy{Attempts: 2, Base: time.Millisecond, Max: time.Millisecond}
	attempts := 0
	start := time.Now()
	err := DoRetry(context.Background(), p, func() error {
		attempts++
		if attempts == 1 {
			return &TransientError{Code: http.StatusTooManyRequests, RetryAfter: ra, Msg: "GET /x: 429"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("DoRetry: %v", err)
	}
	if elapsed := time.Since(start); elapsed < ra {
		t.Fatalf("waited %v, want >= %v", elapsed, ra)
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"  0 ", 0},
		{"-3", 0},
		{"bogus", 0},
		{"120", 120 * time.Second},
	}
	for _, c := range cases {
		if got := parseRetryAfter(c.in); got != c.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	future := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(future); got < 80*time.Second || got > 95*time.Second {
		t.Errorf("HTTP-date Retry-After = %v, want ~90s", got)
	}
}

// TestGetJSON429CarriesRetryAfter checks that the HTTP layer surfaces a 429 as
// a typed transient error carrying the parsed Retry-After header.
func TestGetJSON429CarriesRetryAfter(t *testing.T) {
	t.Setenv("OO_RETRY_ATTEMPTS", "1")
	resetPacingForTest()
	defer resetPacingForTest()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("busy"))
	}))
	defer srv.Close()

	c := NewClient(Credentials{Url: srv.URL})
	c.token = &Token{Value: "tok", Expires: Time(time.Now().Add(time.Hour))}

	_, err := c.getJSON(context.Background(), "/api/2.0/project/4.json")
	if err == nil {
		t.Fatal("want error")
	}
	var te *TransientError
	if !errors.As(err, &te) {
		t.Fatalf("want *TransientError, got %T: %v", err, err)
	}
	if te.Code != http.StatusTooManyRequests {
		t.Fatalf("code = %d, want 429", te.Code)
	}
	if te.RetryAfter != time.Second {
		t.Fatalf("RetryAfter = %v, want 1s", te.RetryAfter)
	}
}

// TestGetJSONRetries429ThenSucceeds verifies the end-to-end path: a throttled
// answer with Retry-After is retried and the eventual success is returned.
func TestGetJSONRetries429ThenSucceeds(t *testing.T) {
	t.Setenv("OO_RETRY_ATTEMPTS", "3")
	t.Setenv("OO_RETRY_BASE", "10ms")
	t.Setenv("OO_RETRY_MAX", "50ms")
	resetPacingForTest()
	defer resetPacingForTest()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{}}`))
	}))
	defer srv.Close()

	c := NewClient(Credentials{Url: srv.URL})
	c.token = &Token{Value: "tok", Expires: Time(time.Now().Add(time.Hour))}

	raw, err := c.getJSON(context.Background(), "/api/2.0/project/4.json")
	if err != nil {
		t.Fatalf("getJSON: %v", err)
	}
	if string(raw) != `{"response":{}}` {
		t.Fatalf("raw = %s", raw)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
