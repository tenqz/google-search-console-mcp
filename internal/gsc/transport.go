package gsc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Options bounds upstream work for one installation. Attempts include the initial request.
type Options struct {
	RequestTimeout   time.Duration
	MaxConcurrent    int
	MaxAttempts      int
	MaxResponseBytes int64
}

func (o Options) defaults() Options {
	if o.RequestTimeout <= 0 {
		o.RequestTimeout = 30 * time.Second
	}
	if o.MaxConcurrent <= 0 {
		o.MaxConcurrent = 8
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 3
	}
	if o.MaxResponseBytes <= 0 {
		o.MaxResponseBytes = 8 << 20
	}
	return o
}

// NewClient accepts an HTTP dependency for controlled transports and contract tests.
// The supplied client is copied so the caller's configuration is not mutated.
func NewClient(client *http.Client, options Options) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	opts := options.defaults()
	bounded := *client
	if bounded.Timeout == 0 || bounded.Timeout > opts.RequestTimeout {
		bounded.Timeout = opts.RequestTimeout
	}
	return &Client{http: &bounded, webmasterBase: defaultWebmasterBase, inspectURL: defaultInspectURL, options: opts, slots: make(chan struct{}, opts.MaxConcurrent)}
}

// RequestError exposes stable failure categories without copying upstream bodies or secrets.
type RequestError struct {
	Kind              string `json:"code"`
	Status            int    `json:"httpStatus,omitempty"`
	Retryable         bool   `json:"retryable"`
	RetryAfterSeconds int    `json:"retryAfterSeconds,omitempty"`
	cause             error
}

func (e *RequestError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("Google API HTTP %d (%s)", e.Status, e.Kind)
	}
	if e.Kind == "response_too_large" {
		return "Google response exceeds the size limit; reduce rowLimit or narrow the query"
	}
	return "Google request: " + e.Kind
}

// Unwrap preserves cancellation and transport causes for callers using errors.Is/As.
func (e *RequestError) Unwrap() error { return e.cause }

func (c *Client) getJSON(ctx context.Context, endpoint string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, dest)
}

func (c *Client) postJSON(ctx context.Context, endpoint string, body any, dest any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req, dest)
}

func (c *Client) doJSON(req *http.Request, dest any) (finalErr error) {
	opts := c.options.defaults()
	ctx, cancel := context.WithTimeout(req.Context(), opts.RequestTimeout)
	defer cancel()
	started := time.Now()
	defer func() {
		kind := "ok"
		var upstream *RequestError
		if errors.As(finalErr, &upstream) {
			kind = upstream.Kind
		} else if finalErr != nil {
			kind = "error"
		}
		// Deliberately omit URLs, query text, credentials and upstream bodies.
		slog.Info("google_request", "result", kind, "duration_ms", time.Since(started).Milliseconds())
	}()
	if c.slots != nil {
		select {
		case c.slots <- struct{}{}:
			defer func() { <-c.slots }()
		case <-ctx.Done():
			return contextFailure(ctx.Err())
		}
	}
	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return contextFailure(err)
		}
		clone := req.Clone(ctx)
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return err
			}
			clone.Body = body
		}
		resp, err := c.http.Do(clone)
		var failure *RequestError
		if err != nil {
			if ctx.Err() != nil {
				return contextFailure(ctx.Err())
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return contextFailure(context.DeadlineExceeded)
			}
			failure = &RequestError{Kind: "upstream_unavailable", Retryable: true, cause: err}
		} else {
			raw, readErr := io.ReadAll(io.LimitReader(resp.Body, opts.MaxResponseBytes+1))
			_ = resp.Body.Close()
			if ctx.Err() != nil {
				return contextFailure(ctx.Err())
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if readErr != nil {
					return &RequestError{Kind: "invalid_response", cause: readErr}
				}
				if int64(len(raw)) > opts.MaxResponseBytes {
					return &RequestError{Kind: "response_too_large"}
				}
				if dest == nil {
					return nil
				}
				if err := json.Unmarshal(raw, dest); err != nil {
					return &RequestError{Kind: "invalid_response", cause: err}
				}
				return nil
			}
			kind := "upstream_error"
			switch resp.StatusCode {
			case 400:
				kind = "invalid_request"
			case 401:
				kind = "unauthenticated"
			case 403:
				kind = "permission_denied"
			case 404:
				kind = "not_found"
			case 429:
				kind = "rate_limited"
			}
			failure = &RequestError{Kind: kind, Status: resp.StatusCode, Retryable: resp.StatusCode == 429 || resp.StatusCode == 500 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504, RetryAfterSeconds: retryAfter(resp.Header.Get("Retry-After"), time.Now())}
		}
		if !failure.Retryable || attempt+1 == opts.MaxAttempts {
			return failure
		}
		delay := time.Duration(100*(1<<min(attempt, 5)))*time.Millisecond + time.Duration(rand.IntN(100))*time.Millisecond
		if failure.RetryAfterSeconds > 0 {
			delay = time.Duration(failure.RetryAfterSeconds) * time.Second
		}
		// Do not wait beyond the request budget, or disregard a server's Retry-After.
		if deadline, ok := ctx.Deadline(); ok && delay >= time.Until(deadline) {
			return failure
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return contextFailure(ctx.Err())
		}
	}
	return &RequestError{Kind: "upstream_unavailable", Retryable: true}
}

func contextFailure(err error) *RequestError {
	kind := "cancelled"
	if errors.Is(err, context.DeadlineExceeded) {
		kind = "timeout"
	}
	return &RequestError{Kind: kind, Retryable: kind == "timeout", cause: err}
}

func retryAfter(value string, now time.Time) int {
	if seconds, err := strconv.Atoi(value); err == nil {
		return min(max(seconds, 0), 86400)
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		return min(int(date.Sub(now).Seconds())+1, 86400)
	}
	return 0
}
