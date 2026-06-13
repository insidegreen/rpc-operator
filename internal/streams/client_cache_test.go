package streams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestEnsureCacheResource_5xxIsTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Error: failed to init cache"))
	}))
	defer srv.Close()

	c := NewHTTPClient()
	err := c.EnsureCacheResource(context.Background(), srv.URL, "nats_cache", "nats_kv: {}\n")
	if err == nil {
		t.Fatal("expected error for 502")
	}
	var rej *ConfigRejectedError
	if errors.As(err, &rej) {
		t.Fatal("502 must NOT be ConfigRejectedError (it is transient)")
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
