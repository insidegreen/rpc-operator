package streams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnsureCacheResource_PostSuccess(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewHTTPClient()
	if err := c.EnsureCacheResource(context.Background(), srv.URL, "shared", "nats_kv: {}\n"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/resources/cache/shared" {
		t.Fatalf("got %s %s, want POST /resources/cache/shared", gotMethod, gotPath)
	}
	if gotBody == "" {
		t.Fatal("body not sent")
	}
}

func TestEnsureCacheResource_LintRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("field not recognised"))
	}))
	defer srv.Close()

	c := NewHTTPClient()
	err := c.EnsureCacheResource(context.Background(), srv.URL, "bad", "x: y\n")
	var rej *ConfigRejectedError
	if !errors.As(err, &rej) {
		t.Fatalf("want ConfigRejectedError, got %v", err)
	}
}

func TestEnsureCacheResource_BareGateway502IsTransient(t *testing.T) {
	// A bodyless gateway 502 (e.g. a restarting pod behind the Service) is
	// transient; it must NOT be a *ConfigRejectedError so the controller keeps
	// retrying instead of marking the resource permanently Failed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("502 Bad Gateway"))
	}))
	defer srv.Close()

	c := NewHTTPClient()
	err := c.EnsureCacheResource(context.Background(), srv.URL, "nats_cache", "nats_kv: {}\n")
	if err == nil {
		t.Fatal("expected error for 502")
	}
	var rej *ConfigRejectedError
	if errors.As(err, &rej) {
		t.Fatal("bare 502 must NOT be ConfigRejectedError (it is transient)")
	}
}

func TestEnsureCacheResource_InitFailure502IsConfigRejected(t *testing.T) {
	// Redpanda Connect reports a cache resource that fails to initialise (e.g. a
	// multilevel cache with fewer than two levels) as 502 with a "failed to init"
	// body. This is permanent for an identical config, so EnsureCacheResource must
	// surface it as *ConfigRejectedError for the controller to record in status
	// instead of requeuing forever with no feedback.
	const initBody = "Error: failed to init cache <no label> path root.cache_resources: expected at least two cache levels, found 1\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(initBody))
	}))
	defer srv.Close()

	c := NewHTTPClient()
	err := c.EnsureCacheResource(context.Background(), srv.URL, "sensor-cache", "multilevel: [a]\n")
	var rej *ConfigRejectedError
	if !errors.As(err, &rej) {
		t.Fatalf("want *ConfigRejectedError for 502 init failure, got %T: %v", err, err)
	}
	if rej.Status != http.StatusBadGateway {
		t.Errorf("want status 502, got %d", rej.Status)
	}
	if !strings.Contains(rej.Body, "expected at least two cache levels") {
		t.Errorf("want init body in error, got %q", rej.Body)
	}
}

func TestDeleteCacheResource_IsNoOp(t *testing.T) {
	// DELETE /resources/cache/{label} is not supported by the RPC streams API
	// (returns "verb not supported" 400). The HTTPClient is intentionally a no-op.
	c := NewHTTPClient()
	if err := c.DeleteCacheResource(context.Background(), "http://unused", "gone"); err != nil {
		t.Fatalf("DeleteCacheResource must be a no-op, got %v", err)
	}
}
