package streams

import (
	"context"
	"errors"
	"testing"
)

func TestFakeClient_CacheResourceLifecycle(t *testing.T) {
	f := NewFakeClient()
	ctx := context.Background()
	url := "http://etl-0.etl.ns.svc:4195"

	if err := f.EnsureCacheResource(ctx, url, "shared", "nats_kv: {}\n"); err != nil {
		t.Fatal(err)
	}
	if !f.HasCacheResource(url, "shared") {
		t.Fatal("resource should exist after ensure")
	}
	if got := f.CacheResourceBody(url, "shared"); got != "nats_kv: {}\n" {
		t.Fatalf("body = %q", got)
	}
	f.DropPod(url)
	if f.HasCacheResource(url, "shared") {
		t.Fatal("DropPod must clear cache resources")
	}
}

func TestFakeClient_DeleteCacheResource(t *testing.T) {
	f := NewFakeClient()
	ctx := context.Background()
	url := "http://etl-0.etl.ns.svc:4195"

	_ = f.EnsureCacheResource(ctx, url, "shared", "nats_kv: {}\n")
	if err := f.DeleteCacheResource(ctx, url, "shared"); err != nil {
		t.Fatal(err)
	}
	if f.HasCacheResource(url, "shared") {
		t.Fatal("resource should be gone after delete")
	}
	// delete missing is a no-op
	if err := f.DeleteCacheResource(ctx, url, "ghost"); err != nil {
		t.Fatal(err)
	}
}

func TestFakeClient_EnsureCacheResourceErr(t *testing.T) {
	f := NewFakeClient()
	f.EnsureCacheErr = errors.New("rejected")
	if err := f.EnsureCacheResource(context.Background(), "u", "l", "c"); err == nil {
		t.Fatal("expected injected error")
	}
	if f.HasCacheResource("u", "l") {
		t.Fatal("resource must NOT be recorded when EnsureCacheErr is set")
	}
}
