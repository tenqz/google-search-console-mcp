package gsc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetriesReplayReadOnlyPost protects body replay and bounded retries for transient failures.
func TestRetriesReplayReadOnlyPost(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != `{"value":1}` {
			t.Errorf("replayed body=%s", raw)
		}
		if calls.Add(1) < 3 {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()
	client := NewClient(ts.Client(), Options{MaxAttempts: 3})
	var result map[string]bool
	if err := client.postJSON(context.Background(), ts.URL, map[string]int{"value": 1}, &result); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || !result["ok"] {
		t.Fatal("retry did not preserve response")
	}
}

// TestPermanentErrorsDoNotRetry protects quotas and avoids returning raw upstream secrets.
func TestPermanentErrorsDoNotRetry(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(403)
		_, _ = w.Write([]byte("secret upstream body"))
	}))
	defer ts.Close()
	err := NewClient(ts.Client(), Options{}).getJSON(context.Background(), ts.URL, nil)
	var failure *RequestError
	if !errors.As(err, &failure) || failure.Status != 403 || failure.Retryable || calls.Load() != 1 || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unexpected error=%v calls=%d", err, calls.Load())
	}
}

// TestRetryAfterBeyondBudgetDoesNotRetry verifies that Retry-After is respected rather than shortened.
func TestRetryAfterBeyondBudgetDoesNotRetry(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(429)
	}))
	defer ts.Close()
	err := NewClient(ts.Client(), Options{RequestTimeout: time.Second}).getJSON(context.Background(), ts.URL, nil)
	var failure *RequestError
	if !errors.As(err, &failure) || failure.RetryAfterSeconds != 60 || calls.Load() != 1 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
}

// TestRequestTimeoutAndSlotCancellation protects time spent both in Google and waiting for a slot.
func TestRequestTimeoutAndSlotCancellation(t *testing.T) {
	entered := make(chan struct{}, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-r.Context().Done() }))
	defer ts.Close()
	client := NewClient(ts.Client(), Options{RequestTimeout: 150 * time.Millisecond, MaxConcurrent: 1})
	done := make(chan error, 1)
	go func() { done <- client.getJSON(context.Background(), ts.URL, nil) }()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := client.getJSON(ctx, ts.URL, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued request=%v", err)
	}
	if err := <-done; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("active request=%v", err)
	}
	if len(client.slots) != 0 {
		t.Fatal("slot leaked")
	}
}

// TestOversizedResponseReturnsActionableError protects large analytics responses from silent truncation.
func TestOversizedResponseReturnsActionableError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"long":"0123456789"}`)) }))
	defer ts.Close()
	err := NewClient(ts.Client(), Options{MaxResponseBytes: 8}).getJSON(context.Background(), ts.URL, &map[string]any{})
	var failure *RequestError
	if !errors.As(err, &failure) || failure.Kind != "response_too_large" {
		t.Fatalf("error=%v", err)
	}
}

// TestOAuthExchangeHasTimeout verifies the token exchange, not only subsequent API requests.
func TestOAuthExchangeHasTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); <-r.Context().Done() }))
	defer ts.Close()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"client_email": "test@example.invalid", "private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), "token_uri": ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClientFromJSON(context.Background(), raw, Options{RequestTimeout: 100 * time.Millisecond, MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err = client.ListSites(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("unbounded token exchange: %v", err)
	}
}

// TestCancelledOAuthCallerDoesNotWaitForGlobalTimeout protects abandoned MCP requests.
func TestCancelledOAuthCallerDoesNotWaitForGlobalTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); <-r.Context().Done() }))
	defer ts.Close()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"client_email": "test@example.invalid", "private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), "token_uri": ts.URL})
	client, err := NewClientFromJSON(context.Background(), raw, Options{RequestTimeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = client.ListSites(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("cancelled token exchange: %v", err)
	}
}

// TestRetryAfterHTTPDate covers the second form allowed by HTTP.
func TestRetryAfterHTTPDate(t *testing.T) {
	now := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	if got := retryAfter(now.Add(10*time.Second).Format(http.TimeFormat), now); got < 10 || got > 11 {
		t.Fatalf("delay=%d", got)
	}
}
