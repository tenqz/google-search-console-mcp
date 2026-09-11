package gsc

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	"net/http"
	"time"
)

func httpClientFromJSON(ctx context.Context, credentialsJSON []byte, timeout time.Duration) (*http.Client, error) {
	var sa serviceAccountFile
	if err := json.Unmarshal(credentialsJSON, &sa); err != nil {
		return nil, fmt.Errorf("parse google credentials: %w", err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, fmt.Errorf("google credentials must include client_email and private_key")
	}
	tokenURL := sa.TokenURI
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}
	cfg := &jwt.Config{
		Email:        sa.ClientEmail,
		PrivateKey:   []byte(sa.PrivateKey),
		PrivateKeyID: sa.PrivateKeyID,
		Scopes:       []string{webmasterScope},
		TokenURL:     tokenURL,
	}
	base := http.DefaultClient
	if custom, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
		base = custom
	}
	bounded := *base
	bounded.Timeout = timeout
	transport := bounded.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{Timeout: timeout, Transport: &tokenTransport{config: cfg, base: transport, http: &bounded, lifecycle: ctx, lock: make(chan struct{}, 1)}}, nil
}

// tokenTransport binds token refresh and waiting callers to their own cancellation.
// A channel serializes refreshes without an uncancellable mutex wait in TokenSource.
type tokenTransport struct {
	config    *jwt.Config
	base      http.RoundTripper
	http      *http.Client
	lifecycle context.Context
	lock      chan struct{}
	token     *oauth2.Token
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case t.lock <- struct{}{}:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	if t.token == nil || !t.token.Valid() {
		ctx, cancel := context.WithCancel(req.Context())
		stop := context.AfterFunc(t.lifecycle, cancel)
		exchange := *t.http
		base := exchange.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		// jwt.TokenSource uses PostForm without NewRequestWithContext; attach the caller explicitly.
		exchange.Transport = &exchangeTransport{ctx: ctx, base: base}
		token, err := t.config.TokenSource(context.WithValue(ctx, oauth2.HTTPClient, &exchange)).Token()
		stop()
		cancel()
		if err != nil {
			<-t.lock
			return nil, err
		}
		t.token = token
	}
	token := t.token
	<-t.lock
	clone := req.Clone(req.Context())
	token.SetAuthHeader(clone)
	return t.base.RoundTrip(clone)
}

type exchangeTransport struct {
	ctx  context.Context
	base http.RoundTripper
}

func (t *exchangeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req.Clone(t.ctx))
}
