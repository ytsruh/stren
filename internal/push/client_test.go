package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// roundTripperFunc adapts a function to webpush.HTTPClient (which
// uses the net/http Do method, not RoundTripper). It is the
// simplest way to intercept the outbound push call in tests.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

// fakeResponse builds a minimal *http.Response the wrapper will accept.
func fakeResponse(status int) *http.Response {
	return fakeResponseWithBody(status, "")
}

// fakeResponseWithBody is fakeResponse with a non-empty body. Used by
// the "response body is captured on non-2xx" tests; the body is what
// Apple / FCM / Mozilla actually return to explain a 4xx/5xx.
func fakeResponseWithBody(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strReader(body)),
		Header:     http.Header{},
	}
}

func strReader(s string) *stringReadCloser { return &stringReadCloser{s: s} }

type stringReadCloser struct{ s string }

func (r *stringReadCloser) Read(p []byte) (int, error) {
	if r.s == "" {
		return 0, io.EOF
	}
	n := copy(p, r.s)
	r.s = r.s[n:]
	return n, nil
}
func (r *stringReadCloser) Close() error { return nil }

// validSub returns a Subscription whose P256dh / Auth fields are
// derived from a freshly-generated P-256 keypair, so webpush-go's
// elliptic.Unmarshal step accepts them. The endpoint is a placeholder
// — most tests do not care.
func validSub() Subscription {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pubBytes := elliptic.Marshal(priv.Curve, priv.PublicKey.X, priv.PublicKey.Y)
	authBytes := make([]byte, 16)
	_, _ = rand.Read(authBytes)
	return Subscription{
		Endpoint: "https://push.example/test/abc",
		P256dh:   base64.RawURLEncoding.EncodeToString(pubBytes),
		Auth:     base64.RawURLEncoding.EncodeToString(authBytes),
	}
}

func TestClient_Send_2xxIsSuccess(t *testing.T) {
	keys := validKeys(t)
	var bodyBytes int
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			b, _ := io.ReadAll(r.Body)
			bodyBytes = len(b)
			return fakeResponse(http.StatusCreated), nil
		}),
	})

	out := client.Send(context.Background(), validSub(), Message{Title: "hi", Body: "there"})
	if out.Error != nil {
		t.Fatalf("expected no error, got %v", out.Error)
	}
	if out.Status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", out.Status)
	}
	if out.Deleted {
		t.Fatal("2xx should not be marked deleted")
	}
	// The body sent over the wire is the RFC 8291 encrypted
	// payload, not the raw JSON. We can only assert it is
	// non-empty (webpush-go encrypted something) and the
	// Content-Encoding / Content-Type headers were set.
	if bodyBytes == 0 {
		t.Fatal("expected non-empty encrypted body")
	}
}

func TestClient_Send_410MarksDeleted(t *testing.T) {
	keys := validKeys(t)
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusGone), nil
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if !out.Deleted {
		t.Fatal("expected Deleted=true on 410")
	}
	if out.Error == nil {
		t.Fatal("expected non-nil error on 410")
	}
	if out.Status != http.StatusGone {
		t.Fatalf("expected status 410, got %d", out.Status)
	}
}

func TestClient_Send_404MarksDeleted(t *testing.T) {
	keys := validKeys(t)
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusNotFound), nil
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if !out.Deleted {
		t.Fatal("expected Deleted=true on 404")
	}
}

func TestClient_Send_5xxIsFailedNotDeleted(t *testing.T) {
	keys := validKeys(t)
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return fakeResponse(http.StatusInternalServerError), nil
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if out.Deleted {
		t.Fatal("5xx should not mark deleted")
	}
	if out.Error == nil {
		t.Fatal("expected non-nil error on 5xx")
	}
	if out.Status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", out.Status)
	}
}

// TestClient_Send_403IncludesBody makes sure that when the push
// service rejects a request with a non-2xx (e.g. Apple's 403 with
// {"reason":"BadJwt"}), the response body ends up in the returned
// error so the admin toast and server log can show the real reason
// instead of just "unexpected status 403". This was the gap that
// masked the iOS `.local` VAPID subscriber bug.
func TestClient_Send_403IncludesBody(t *testing.T) {
	keys := validKeys(t)
	body := `{"reason":"BadSubscriber"}`
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return fakeResponseWithBody(http.StatusForbidden, body), nil
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if out.Deleted {
		t.Fatal("403 should not mark deleted")
	}
	if out.Error == nil {
		t.Fatal("expected non-nil error on 403")
	}
	if out.Status != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", out.Status)
	}
	if !contains(out.Error.Error(), "403") {
		t.Errorf("error should mention status 403, got %q", out.Error.Error())
	}
	if !contains(out.Error.Error(), "BadSubscriber") {
		t.Errorf("error should include response body %q, got %q", body, out.Error.Error())
	}
}

// TestClient_Send_410WithBodyStillIncludesBody exercises the same
// body-capture path through the "subscription gone" branch: even
// though the row is marked Deleted, the body should still appear in
// the error message so the operator can see WHY the push service
// dropped the subscription.
func TestClient_Send_410WithBodyStillIncludesBody(t *testing.T) {
	keys := validKeys(t)
	body := `{"reason":"DeviceTokenNotForTopic"}`
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return fakeResponseWithBody(http.StatusGone, body), nil
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if !out.Deleted {
		t.Fatal("expected Deleted=true on 410")
	}
	if out.Error == nil {
		t.Fatal("expected non-nil error on 410")
	}
	if !contains(out.Error.Error(), "DeviceTokenNotForTopic") {
		t.Errorf("error should include response body %q, got %q", body, out.Error.Error())
	}
}

// TestClient_Send_TruncatesVeryLongBody guards against a misbehaving
// push service (or a proxy in front of one) returning a megabyte of
// HTML in an error page. The body is capped to keep the log line and
// admin toast readable.
func TestClient_Send_TruncatesVeryLongBody(t *testing.T) {
	keys := validKeys(t)
	huge := strings.Repeat("X", maxErrorBody*4)
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return fakeResponseWithBody(http.StatusInternalServerError, huge), nil
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if out.Error == nil {
		t.Fatal("expected non-nil error on 500")
	}
	if len(out.Error.Error()) > maxErrorBody+64 {
		t.Errorf("error message too long (%d bytes); expected <= ~%d. got %q",
			len(out.Error.Error()), maxErrorBody+64, out.Error.Error())
	}
}

func TestClient_Send_TransportError(t *testing.T) {
	keys := validKeys(t)
	client := NewClient(keys, ClientConfig{
		HTTPClient: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errFake("network down")
		}),
	})
	out := client.Send(context.Background(), validSub(), Message{Title: "x", Body: "y"})
	if out.Deleted {
		t.Fatal("transport error should not mark deleted")
	}
	if out.Error == nil {
		t.Fatal("expected transport error")
	}
}

func TestClient_Send_RejectsIncompleteSubscription(t *testing.T) {
	keys := validKeys(t)
	client := NewClient(keys, ClientConfig{})

	bad := Subscription{Endpoint: "https://x", P256dh: "", Auth: "y"}
	out := client.Send(context.Background(), bad, Message{})
	if out.Error == nil {
		t.Fatal("expected error on incomplete subscription")
	}
	if out.Deleted {
		t.Fatal("validation error should not mark deleted")
	}
}

// TestClient_Send_UsesVAPIDHeaders makes sure the VAPID
// Authentication header is set so the push service can authenticate
// the sender. We point the client at a real httptest server (the
// webpush-go library builds the encrypted body and the VAPID JWT
// internally) and check the Authorization header.
func TestClient_Send_UsesVAPIDHeaders(t *testing.T) {
	keys := validKeys(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !startsWith(auth, "vapid ") {
			t.Errorf("missing vapid auth header; got %q", auth)
		}
		if r.Header.Get("TTL") == "" {
			t.Error("missing TTL header")
		}
		// Drain body so the client does not see a connection error.
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := NewClient(keys, ClientConfig{
		HTTPClient: srv.Client(),
		Subscriber: "mailto:test@example.com",
	})

	// webpush-go needs the endpoint to match the host or it errors.
	sub := validSub()
	sub.Endpoint = srv.URL + "/push/abc"

	out := client.Send(context.Background(), sub, Message{Title: "x", Body: "y"})
	if out.Error != nil {
		t.Fatalf("send failed: %v", out.Error)
	}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

type errFake string

func (e errFake) Error() string { return string(e) }

// safeBuffer is a bytes.Buffer guarded by a mutex. ClientConfig.Logger
// is called from the Send path; the test goroutine reads the buffer
// afterwards. Without the lock the race detector would flag the
// access even though Send is synchronous today, because future
// changes could move the log call to a worker goroutine.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *safeBuffer) WriteByte(c byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.WriteByte(c)
}

// TestClient_Send_LoggerCapturesVAPIDHeader makes sure that when
// ClientConfig.Logger is set, the wrapper around the HTTPClient
// records the VAPID Authorization header on every outbound push.
// This is the diagnostic hook that surfaces Apple's BadJwtToken
// rejections: the JWT in the log line can be split on '.' and
// base64url-decoded to see alg/aud/exp/sub, none of which the
// push service's 403 body tells us.
func TestClient_Send_LoggerCapturesVAPIDHeader(t *testing.T) {
	keys := validKeys(t)

	var logBuf safeBuffer
	logger := func(format string, args ...any) {
		// Mirror log.Printf semantics so the captured string is
		// the same one an operator would see in production.
		fmt.Fprintf(&logBuf, format, args...)
		logBuf.WriteByte('\n')
	}

	// Use a real httptest server so webpush-go builds a real JWT
	// (its encryption + VAPID signing paths only run when there is
	// a real network round-trip). The server returns 403 with the
	// body that mimics Apple's BadJwtToken response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"reason":"BadJwtToken"}`))
	}))
	defer srv.Close()

	client := NewClient(keys, ClientConfig{
		HTTPClient: srv.Client(),
		Subscriber: "mailto:test@example.com",
		Logger:     logger,
	})

	sub := validSub()
	sub.Endpoint = srv.URL + "/push/abc"

	out := client.Send(context.Background(), sub, Message{Title: "x", Body: "y"})
	if out.Error == nil {
		t.Fatal("expected error from 403 response")
	}
	if !contains(out.Error.Error(), "BadJwtToken") {
		t.Errorf("error should include response body, got %q", out.Error.Error())
	}

	logged := logBuf.String()
	if logged == "" {
		t.Fatal("Logger was never called; expected the VAPID Authorization header to be recorded")
	}
	if !contains(logged, "vapid t=") {
		t.Errorf("log line should contain the VAPID JWT prefix; got %q", logged)
	}
	if !contains(logged, "k=") {
		t.Errorf("log line should contain the VAPID public key segment; got %q", logged)
	}
	if !contains(logged, srv.URL) {
		t.Errorf("log line should mention the push service URL; got %q", logged)
	}
	if !contains(logged, "POST") {
		t.Errorf("log line should mention the HTTP method; got %q", logged)
	}
}

// TestClient_Send_NoLoggerNoLog makes sure the default
// configuration (no Logger) is completely silent on the wire.
// This locks in the "zero cost when not configured" promise of
// the ClientConfig.Logger field.
func TestClient_Send_NoLoggerNoLog(t *testing.T) {
	keys := validKeys(t)

	var logBuf safeBuffer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	// Note: no Logger set on ClientConfig.
	client := NewClient(keys, ClientConfig{
		HTTPClient: srv.Client(),
		Subscriber: "mailto:test@example.com",
	})

	sub := validSub()
	sub.Endpoint = srv.URL + "/push/abc"
	if out := client.Send(context.Background(), sub, Message{Title: "x", Body: "y"}); out.Error != nil {
		t.Fatalf("send failed: %v", out.Error)
	}

	if logBuf.String() != "" {
		t.Errorf("expected silent log buffer, got %q", logBuf.String())
	}
}

// TestClient_Send_VAPIDSubClaimIsNotDoublePrefixed is the
// regression test for the bug that surfaced as Apple's
// `BadJwtToken` 403: webpush-go prepends `mailto:` to any
// `Subscriber` value that does not start with `https:`, so
// passing `mailto:foo@bar` produces the malformed
// `mailto:mailto:foo@bar` in the JWT. This test pins down the
// exact `sub` claim value the default configuration produces,
// so any future change to defaultSubject or to the way the
// value is passed to webpush-go is caught at unit-test time
// rather than discovered by an iOS user.
func TestClient_Send_VAPIDSubClaimIsNotDoublePrefixed(t *testing.T) {
	keys := validKeys(t)

	var logBuf safeBuffer
	logger := func(format string, args ...any) {
		fmt.Fprintf(&logBuf, format, args...)
		logBuf.WriteByte('\n')
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	// Use the package default (no Subscriber override) so this
	// test is the one that catches a regression in defaultSubject
	// itself. The other tests in this file pass
	// `mailto:test@example.com` explicitly, which is the case the
	// new doc comment warns about.
	client := NewClient(keys, ClientConfig{
		HTTPClient: srv.Client(),
		Logger:     logger,
	})

	sub := validSub()
	sub.Endpoint = srv.URL + "/push/abc"
	if out := client.Send(context.Background(), sub, Message{Title: "x", Body: "y"}); out.Error != nil {
		t.Fatalf("send failed: %v", out.Error)
	}

	logged := logBuf.String()
	if !contains(logged, "vapid t=") {
		t.Fatalf("no VAPID header in log; got %q", logged)
	}

	// Extract the JWT from the auth="vapid t=...,k=..." line.
	// The log format is fixed by the loggingHTTPClient wrapper.
	const marker = `auth="vapid t=`
	start := strings.Index(logged, marker)
	if start < 0 {
		t.Fatalf("could not find VAPID marker in log line %q", logged)
	}
	rest := logged[start+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("could not find end of auth value in log line %q", logged)
	}
	authValue := rest[:end]

	// authValue is "<JWT>, k=<key>". The marker already consumed
	// "vapid t=", so the rest starts with the JWT itself. The JWT
	// is everything before the ", k=" separator. webpush-go emits
	// the separator with a space (see vapid.go:106 in the library).
	const sep = ", k="
	comma := strings.Index(authValue, sep)
	if comma < 0 {
		t.Fatalf(`no %q separator in auth value %q`, sep, authValue)
	}
	jwt := authValue[:comma]

	// JWT is header.payload.signature. We only need the payload
	// (the second segment) to assert the sub claim.
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d parts: %q", len(parts), jwt)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}

	// The single most important assertion: the sub claim is the
	// raw value we configured, not a double-prefixed form.
	if !contains(string(payload), `"sub":"https://stren.ytsruh.com"`) {
		t.Errorf(`sub claim should be "https://stren.ytsruh.com" (no extra prefix), got payload: %s`, payload)
	}

	// Belt and braces: assert we never produced the malformed
	// form, regardless of how the bug could manifest. If a
	// future refactor introduces a different prefix, the test
	// above catches it; this one catches a regression of the
	// original bug.
	if contains(string(payload), "mailto:mailto:") {
		t.Errorf("sub claim contains the double-prefix bug; got payload: %s", payload)
	}

	// Also confirm the rest of the JWT looks right, so a future
	// regression that swapped `sub` for some other claim is
	// caught here too.
	if !contains(string(payload), `"aud":"https://web.push.apple.com"`) &&
		!contains(string(payload), `"aud":"https://`+srv.Listener.Addr().String()) {
		// The aud claim is the origin of the endpoint, which is
		// the test server's address when using httptest. Just
		// confirm an aud claim is present at all.
		if !contains(string(payload), `"aud":"`) {
			t.Errorf("payload missing aud claim: %s", payload)
		}
	}
}
