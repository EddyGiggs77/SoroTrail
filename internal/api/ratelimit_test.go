package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mkReq(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func startLimiter(t *testing.T, lim *RateLimiter) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	lim.Start(ctx)
	return func() {
		cancel()
		lim.Stop()
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		trustXFF   bool
		want       string
	}{
		{name: "trusted honors XFF", remoteAddr: "203.0.113.9:5432", xff: "198.51.100.7, 10.0.0.1", trustXFF: true, want: "198.51.100.7"},
		{name: "untrusted ignores XFF", remoteAddr: "203.0.113.9:5432", xff: "198.51.100.7, 10.0.0.1", want: "203.0.113.9"},
		{name: "trusted falls back to remote without XFF", remoteAddr: "203.0.113.9:5432", trustXFF: true, want: "203.0.113.9"},
		{name: "trusted skips invalid XFF parts", remoteAddr: "203.0.113.9:5432", xff: "not-an-ip, 198.51.100.7", trustXFF: true, want: "198.51.100.7"},
		{name: "bare IP remote without port", remoteAddr: "203.0.113.9", trustXFF: true, want: "203.0.113.9"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = c.remoteAddr
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := clientIP(r, c.trustXFF); got != c.want {
				t.Fatalf("clientIP() = %q, want %q (trustXFF=%v, XFF=%q)", got, c.want, c.trustXFF, c.xff)
			}
		})
	}
}

func TestClientKeyUsesRemoteAddrWhenNoCredential(t *testing.T) {
	l := NewRateLimiter(1, 1, false)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.9:1234"
	if got := l.clientKey(r); got != "203.0.113.9" {
		t.Fatalf("clientKey() = %q, want 203.0.113.9", got)
	}
}

// TestCeilSeconds pins the Retry-After rounding. Rounding down would
// tell a throttled client to retry before its budget refills, which
// produces a second 429 and looks like the limiter is broken; a
// negative value would violate RFC 7231's delta-seconds contract
// outright.
func TestCeilSeconds(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{name: "zero returns the documented minimum", in: 0, want: time.Second},
		{name: "sub-second never rounds down to zero", in: time.Millisecond, want: time.Second},
		{name: "one nanosecond rounds up to a full second", in: 1, want: time.Second},
		{name: "sub-second rounds up to one", in: 500 * time.Millisecond, want: time.Second},
		{name: "exact whole second stays as is", in: time.Second, want: time.Second},
		{name: "larger exact multiple stays as is", in: 3 * time.Second, want: 3 * time.Second},
		{name: "fractional duration rounds up", in: 1500 * time.Millisecond, want: 2 * time.Second},
		{name: "just over a second rounds up to two", in: time.Second + time.Nanosecond, want: 2 * time.Second},
		{name: "999 milliseconds round up to one", in: 999 * time.Millisecond, want: time.Second},
		{name: "negative duration clamps to the minimum", in: -500 * time.Millisecond, want: time.Second},
		{name: "large negative never yields a negative header", in: -time.Hour, want: time.Second},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ceilSeconds(c.in)
			assert.Equal(t, c.want, got, "ceilSeconds(%s)", c.in)
			assert.Greater(t, got, time.Duration(0),
				"Retry-After must never be zero or negative (input %s)", c.in)
		})
	}
}

func TestBucketEntryForReturnsSameInstance(t *testing.T) {
	l := NewRateLimiter(1, 1, false)
	e1 := l.bucketEntryFor("k1", 1, 1)
	if e1 == nil {
		t.Fatal("bucketEntryFor() returned nil")
	}
	if e2 := l.bucketEntryFor("k1", 1, 1); e2 != e1 {
		t.Fatal("bucketEntryFor() did not return the same bucket instance")
	}
}
