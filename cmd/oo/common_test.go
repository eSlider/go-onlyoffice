package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestFmtCell(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"hello", "hello"},
		{true, "true"},
		{false, "false"},
		{float64(42), "42"},
		{float64(1.5), "1.5"},
		{int64(7), "7"},
		{map[string]any{"a": 1}, `{"a":1}`},
	}
	for _, tc := range tests {
		if got := fmtCell(tc.in); got != tc.want {
			t.Fatalf("fmtCell(%#v)=%q want %q", tc.in, got, tc.want)
		}
	}
	long := strings.Repeat("x", 100)
	got := fmtCell(long)
	if !strings.HasSuffix(got, "…") || len(got) > 82 {
		t.Fatalf("fmtCell(long)=%q len=%d", got, len(got))
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("a\nb", 10); got != "a b" {
		t.Fatalf("truncate newline=%q", got)
	}
	got := truncate(strings.Repeat("z", 20), 10)
	if !strings.HasSuffix(got, "…") || len(got) >= 20 {
		t.Fatalf("truncate=%q len=%d", got, len(got))
	}
}

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]any{"c": 1, "a": 2, "b": 3})
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedKeys=%v want %v", got, want)
		}
	}
}

func TestFlexIDFloat(t *testing.T) {
	if got := flexIDFloat(float64(12)); got != 12 {
		t.Fatalf("float64=%v", got)
	}
	if got := flexIDFloat(7); got != 7 {
		t.Fatalf("int=%v", got)
	}
	if got := flexIDFloat("3.5"); got != 3.5 {
		t.Fatalf("string=%v", got)
	}
}

func TestIdString(t *testing.T) {
	m := map[string]any{"id": float64(99), "s": "x", "i": 5}
	if got := idString(m, "id"); got != "99" {
		t.Fatalf("float id=%q", got)
	}
	if got := idString(m, "s"); got != "x" {
		t.Fatalf("string id=%q", got)
	}
	if got := idString(m, "i"); got != "5" {
		t.Fatalf("int id=%q", got)
	}
	if got := idString(m, "missing"); got != "" {
		t.Fatalf("missing=%q", got)
	}
}

func TestPrintTableJSON(t *testing.T) {
	defer func(old string) { outputFormat = old }(outputFormat)
	outputFormat = "json"
	out := captureStdout(t, func() {
		printTable([]string{"id", "name"}, []map[string]any{{"id": 1, "name": "alpha"}})
	})
	if !strings.Contains(out, `"id": 1`) || !strings.Contains(out, `"name": "alpha"`) {
		t.Fatalf("json output=%q", out)
	}
}

func TestPrintTableEmpty(t *testing.T) {
	defer func(old string) { outputFormat = old }(outputFormat)
	outputFormat = "table"
	out := captureStdout(t, func() { printTable([]string{"id"}, nil) })
	if strings.TrimSpace(out) != "(empty)" {
		t.Fatalf("empty table=%q", out)
	}
}

func TestPrintObjectTable(t *testing.T) {
	defer func(old string) { outputFormat = old }(outputFormat)
	outputFormat = "table"
	out := captureStdout(t, func() {
		printObject(map[string]any{"id": float64(1), "title": "Demo"})
	})
	if !strings.Contains(out, "id") || !strings.Contains(out, "title") || !strings.Contains(out, "Demo") {
		t.Fatalf("table object=%q", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		old, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if ok {
				_ = os.Setenv(key, old)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}
