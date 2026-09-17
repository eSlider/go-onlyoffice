package onlyoffice

// Client-side pacing and the process-wide 429 gate. A token bucket applies to
// every HTTP request a *Client makes (via pacedTransport), so listing, folder
// creation, uploads and even `get project` share one budget and cannot burst
// past openresty. A typed transient answer carrying Retry-After opens the
// cooldown gate for the whole process.

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Default pacing: ~4 requests/second with a single-request burst. This keeps
// bulk syncs under openresty's threshold without noticeably slowing one-shot
// tools.
const (
	defaultRateLimit = 4.0
	defaultBurst     = 1
)

// rateLimiter is a deterministic token bucket. A nil *rateLimiter is a no-op,
// which is how OO_RATE_LIMIT=0 disables pacing.
type rateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

// newRateLimiter returns a limiter at rate requests/second with the given
// burst. rate <= 0 disables pacing and returns nil.
func newRateLimiter(rate float64, burst int) *rateLimiter {
	if rate <= 0 {
		return nil
	}
	if burst < 1 {
		burst = 1
	}
	return &rateLimiter{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

// wait consumes one token, blocking until one is available or ctx is done.
func (l *rateLimiter) wait(ctx context.Context) error {
	if l == nil {
		return nil
	}
	for {
		l.mu.Lock()
		now := time.Now()
		l.tokens += now.Sub(l.last).Seconds() * l.rate
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
		l.last = now
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		need := time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
		l.mu.Unlock()
		if need < time.Millisecond {
			need = time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(need):
		}
	}
}

// cooldownGate is the process-wide 429 gate: after a throttled answer every
// request waits until the gate opens, so concurrent and sequential callers do
// not hammer the server.
type cooldownGate struct {
	mu    sync.Mutex
	until time.Time
}

// note extends the gate to at least now+d.
func (g *cooldownGate) note(d time.Duration) {
	if d <= 0 {
		return
	}
	g.mu.Lock()
	if u := time.Now().Add(d); u.After(g.until) {
		g.until = u
	}
	g.mu.Unlock()
}

// wait blocks until the gate opens or ctx is done.
func (g *cooldownGate) wait(ctx context.Context) error {
	g.mu.Lock()
	d := time.Until(g.until)
	g.mu.Unlock()
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

var (
	globalLimiter   *rateLimiter
	globalLimiterOn bool
	globalLimiterMu sync.Mutex
	globalCooldown  = &cooldownGate{}
)

// limiter returns the process-wide rate limiter, initialising it from the
// environment on first use. OO_RATE_LIMIT=0 yields a nil (disabled) limiter.
func limiter() *rateLimiter {
	globalLimiterMu.Lock()
	defer globalLimiterMu.Unlock()
	if !globalLimiterOn {
		globalLimiter = newRateLimiterFromEnv()
		globalLimiterOn = true
	}
	return globalLimiter
}

// newRateLimiterFromEnv builds a limiter from OO_RATE_LIMIT (requests/second,
// default 4; 0 disables) and OO_BURST (default 1). Malformed values fall back
// to the defaults.
func newRateLimiterFromEnv() *rateLimiter {
	rate := defaultRateLimit
	if v, ok := os.LookupEnv("OO_RATE_LIMIT"); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			rate = f
		}
	}
	burst := defaultBurst
	if v, ok := os.LookupEnv("OO_BURST"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			burst = n
		}
	}
	return newRateLimiter(rate, burst)
}

// pacedTransport applies the process-wide cooldown gate and rate limiter to
// every request, and arms the gate on a 429. Installed by NewClient, so all
// HTTP paths — typed Query, the http.go helpers, auth and WebDAV — are paced.
type pacedTransport struct {
	base http.RoundTripper
}

func (t *pacedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := globalCooldown.wait(req.Context()); err != nil {
		return nil, err
	}
	if err := limiter().wait(req.Context()); err != nil {
		return nil, err
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		globalCooldown.note(retryAfterOf(resp))
	}
	return resp, nil
}

// envInt reads a positive integer env var, falling back to def.
func envInt(name string, def int) int {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// envDuration reads a duration env var (Go syntax, e.g. "2s", "2m"),
// falling back to def.
func envDuration(name string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil || d <= 0 {
		return def
	}
	return d
}
