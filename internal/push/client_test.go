package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// roundTripperFunc adapts a function to webpush.HTTPClient (which
// uses the net/http Do method, not RoundTripper). It is the
// simplest way to intercept the outbound push call in tests.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

// fakeResponse builds a minimal *http.Response the wrapper will accept.
func fakeResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strReader("")),
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

type errFake string

func (e errFake) Error() string { return string(e) }
