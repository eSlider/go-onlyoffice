package onlyoffice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestMain disables the process-wide rate limiter for the suite so existing
// fake-server tests stay fast; the pacing tests configure their own limiters.
func TestMain(m *testing.M) {
	_ = os.Setenv("OO_RATE_LIMIT", "0")
	os.Exit(m.Run())
}

// resetPacingForTest restores the package-level limiter and cooldown gate to a
// clean state. White-box helper for tests that arm the global 429 gate.
func resetPacingForTest() {
	globalLimiterMu.Lock()
	globalLimiter = nil
	globalLimiterOn = false
	globalLimiterMu.Unlock()
	globalCooldown = &cooldownGate{}
}

func TestNewClientUsesPacedTransport(t *testing.T) {
	c := NewClient(Credentials{Url: "https://example.test"})
	if _, ok := c.client.Transport.(*pacedTransport); !ok {
		t.Fatalf("transport = %T, want *pacedTransport", c.client.Transport)
	}
}

func TestNewRateLimiterFromEnv(t *testing.T) {
	t.Setenv("OO_RATE_LIMIT", "")
	t.Setenv("OO_BURST", "")
	if l := newRateLimiterFromEnv(); l == nil || l.rate != defaultRateLimit || l.burst != defaultBurst {
		t.Fatalf("defaults: got %+v, want rate=%v burst=%d", l, defaultRateLimit, defaultBurst)
	}

	t.Setenv("OO_RATE_LIMIT", "0")
	if l := newRateLimiterFromEnv(); l != nil {
		t.Fatalf("OO_RATE_LIMIT=0: got %+v, want nil (off)", l)
	}

	t.Setenv("OO_RATE_LIMIT", "10")
	t.Setenv("OO_BURST", "3")
	if l := newRateLimiterFromEnv(); l == nil || l.rate != 10 || l.burst != 3 {
		t.Fatalf("explicit: got %+v, want rate=10 burst=3", l)
	}
}

func TestNilRateLimiterIsNoOp(t *testing.T) {
	var l *rateLimiter
	start := time.Now()
	if err := l.wait(context.Background()); err != nil {
		t.Fatalf("nil limiter wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("nil limiter blocked for %v", elapsed)
	}
}

func TestRateLimiterPacesBurst(t *testing.T) {
	l := newRateLimiter(50, 1) // 20ms per token, no burst headroom
	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := l.wait(context.Background()); err != nil {
			t.Fatalf("wait %d: %v", i, err)
		}
	}
	// 5 calls with burst 1 -> 4 gaps * 20ms = 80ms.
	if elapsed := time.Since(start); elapsed < 60*time.Millisecond {
		t.Fatalf("elapsed %v, want >= 60ms (not paced)", elapsed)
	}
}

func TestRateLimiterHonoursContext(t *testing.T) {
	l := newRateLimiter(1, 1)
	if err := l.wait(context.Background()); err != nil {
		t.Fatalf("first wait: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := l.wait(ctx); err == nil {
		t.Fatal("want context error while waiting for a token")
	}
}

func TestCooldownGateBlocksAndExpires(t *testing.T) {
	g := &cooldownGate{}
	g.note(50 * time.Millisecond)
	start := time.Now()
	if err := g.wait(context.Background()); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 45*time.Millisecond {
		t.Fatalf("gate did not block: %v", elapsed)
	}
	start = time.Now()
	if err := g.wait(context.Background()); err != nil {
		t.Fatalf("second wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("expired gate still blocked: %v", elapsed)
	}
}

func TestCooldownGateNoteExtendsNotShortens(t *testing.T) {
	g := &cooldownGate{}
	g.note(80 * time.Millisecond)
	g.note(10 * time.Millisecond) // must not shorten the window
	if d := time.Until(g.until); d < 70*time.Millisecond {
		t.Fatalf("note shortened gate: %v left", d)
	}
}

// TestPacedTransportArmsCooldownOn429 verifies that a throttled answer opens the
// process-wide gate and that the next request is blocked for the Retry-After
// window even though it targets a healthy endpoint.
func TestPacedTransportArmsCooldownOn429(t *testing.T) {
	resetPacingForTest()
	defer resetPacingForTest()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/throttle" {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	tr := &pacedTransport{base: http.DefaultTransport}

	req1, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/throttle", nil)
	resp1, err := tr.RoundTrip(req1)
	if err != nil {
		t.Fatalf("throttled round trip: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", resp1.StatusCode)
	}

	start := time.Now()
	req2, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/ok", nil)
	resp2, err := tr.RoundTrip(req2)
	if err != nil {
		t.Fatalf("second round trip: %v", err)
	}
	resp2.Body.Close()
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Fatalf("cooldown did not block next call: %v", elapsed)
	}
}
